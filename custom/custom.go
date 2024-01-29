package main

/*
#include "stdint.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/hiddify/libcore/bridge"
	"github.com/hiddify/libcore/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

var box *libbox.BoxService
var configOptions *config.ConfigOptions
var activeConfigPath *string
var logFactory *log.Factory

//export setupOnce
func setupOnce(api unsafe.Pointer) {
	bridge.InitializeDartApi(api)
}

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char, statusPort C.longlong, debug bool) (CErr *C.char) {
	defer config.DeferPanicToError("setup", func(err error) {
		CErr = C.CString(err.Error())
		fmt.Printf("Error: %+v\n", err)
	})

	Setup(C.GoString(baseDir), C.GoString(workingDir), C.GoString(tempDir))
	statusPropagationPort = int64(statusPort)

	var defaultWriter io.Writer
	if !debug {
		defaultWriter = io.Discard
	}
	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    false,
		})
	if err != nil {
		return C.CString(err.Error())
	}
	logFactory = &factory
	return C.CString("")
}

//export parse
func parse(path *C.char, tempPath *C.char, debug bool) (CErr *C.char) {
	defer config.DeferPanicToError("parse", func(err error) {
		CErr = C.CString(err.Error())
	})

	config, err := config.ParseConfig(C.GoString(tempPath), debug)
	if err != nil {
		return C.CString(err.Error())
	}
	err = os.WriteFile(C.GoString(path), config, 0644)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export changeConfigOptions
func changeConfigOptions(configOptionsJson *C.char) (CErr *C.char) {
	defer config.DeferPanicToError("changeConfigOptions", func(err error) {
		CErr = C.CString(err.Error())
	})

	configOptions = &config.ConfigOptions{}
	err := json.Unmarshal([]byte(C.GoString(configOptionsJson)), configOptions)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export generateConfig
func generateConfig(path *C.char) (res *C.char) {
	defer config.DeferPanicToError("generateConfig", func(err error) {
		res = C.CString("error" + err.Error())
	})

	config, err := generateConfigFromFile(C.GoString(path), *configOptions)
	if err != nil {
		return C.CString("error" + err.Error())
	}
	return C.CString(config)
}

func generateConfigFromFile(path string, configOpt config.ConfigOptions) (string, error) {
	os.Chdir(filepath.Dir(path))
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	options, err := parseConfig(string(content))
	if err != nil {
		return "", err
	}
	config, err := config.BuildConfigJson(configOpt, options)
	if err != nil {
		return "", err
	}
	return config, nil
}

//export start
func start(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {
	defer config.DeferPanicToError("start", func(err error) {
		CErr = C.CString(err.Error())
	})

	if status != Stopped {
		return C.CString("")
	}
	propagateStatus(Starting)

	path := C.GoString(configPath)
	activeConfigPath = &path

	libbox.SetMemoryLimit(!disableMemoryLimit)
	err := startService(false)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

func startService(delayStart bool) error {
	content, err := os.ReadFile(*activeConfigPath)
	if err != nil {
		return stopAndAlert(EmptyConfiguration, err)
	}
	options, err := parseConfig(string(content))
	if err != nil {
		return stopAndAlert(EmptyConfiguration, err)
	}
	os.Chdir(filepath.Dir(*activeConfigPath))
	var patchedOptions *option.Options
	patchedOptions, err = config.BuildConfig(*configOptions, options)
	if err != nil {
		return fmt.Errorf("error building config: %w", err)
	}

	config.SaveCurrentConfig(sWorkingPath, *patchedOptions)

	err = startCommandServer(*logFactory)
	if err != nil {
		return stopAndAlert(StartCommandServer, err)
	}

	instance, err := NewService(*patchedOptions)
	if err != nil {
		return stopAndAlert(CreateService, err)
	}

	if delayStart {
		time.Sleep(250 * time.Millisecond)
	}

	err = instance.Start()
	if err != nil {
		return stopAndAlert(StartService, err)
	}
	box = instance
	commandServer.SetService(box)

	propagateStatus(Started)
	return nil
}

//export stop
func stop() (CErr *C.char) {
	defer config.DeferPanicToError("stop", func(err error) {
		CErr = C.CString(err.Error())
	})

	if status != Started {
		return C.CString("")
	}
	if box == nil {
		return C.CString("instance not found")
	}
	propagateStatus(Stopping)

	commandServer.SetService(nil)
	err := box.Close()
	if err != nil {
		return C.CString(err.Error())
	}
	box = nil

	err = commandServer.Close()
	if err != nil {
		return C.CString(err.Error())
	}
	commandServer = nil
	propagateStatus(Stopped)

	return C.CString("")
}

//export restart
func restart(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {
	defer config.DeferPanicToError("restart", func(err error) {
		CErr = C.CString(err.Error())
	})
	log.Debug("[Service] Restarting")

	if status != Started {
		return C.CString("")
	}
	if box == nil {
		return C.CString("instance not found")
	}

	err := stop()
	if C.GoString(err) != "" {
		return err
	}

	propagateStatus(Starting)

	time.Sleep(250 * time.Millisecond)

	path := C.GoString(configPath)
	activeConfigPath = &path
	libbox.SetMemoryLimit(!disableMemoryLimit)
	gErr := startService(false)
	if gErr != nil {
		return C.CString(gErr.Error())
	}
	return C.CString("")
}

//export startCommandClient
func startCommandClient(command C.int, port C.longlong) *C.char {
	err := StartCommand(int32(command), int64(port), *logFactory)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export stopCommandClient
func stopCommandClient(command C.int) *C.char {
	err := StopCommand(int32(command))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export selectOutbound
func selectOutbound(groupTag *C.char, outboundTag *C.char) (CErr *C.char) {
	defer config.DeferPanicToError("selectOutbound", func(err error) {
		CErr = C.CString(err.Error())
	})

	err := libbox.NewStandaloneCommandClient().SelectOutbound(C.GoString(groupTag), C.GoString(outboundTag))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export urlTest
func urlTest(groupTag *C.char) (CErr *C.char) {
	defer config.DeferPanicToError("urlTest", func(err error) {
		CErr = C.CString(err.Error())
	})

	err := libbox.NewStandaloneCommandClient().URLTest(C.GoString(groupTag))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

func main() {}
