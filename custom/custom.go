package main

/*
#include "stdint.h"
*/
import "C"
import (
	"encoding/json"
	"os"
	"time"
	"unsafe"

	"github.com/hiddify/libcore/bridge"
	"github.com/hiddify/libcore/shared"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

var box *libbox.BoxService
var configOptions *shared.ConfigOptions
var activeConfigPath *string

//export setupOnce
func setupOnce(api unsafe.Pointer) {
	bridge.InitializeDartApi(api)
}

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char, statusPort C.longlong) {
	Setup(C.GoString(baseDir), C.GoString(workingDir), C.GoString(tempDir))
	statusPropagationPort = int64(statusPort)
}

//export parse
func parse(path *C.char, tempPath *C.char, debug bool) (CErr *C.char) {
	defer shared.DeferPanicToError("parse", func(err error) {
		CErr = C.CString(err.Error())
	})

	err := shared.ParseConfig(C.GoString(path), C.GoString(tempPath), debug)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export changeConfigOptions
func changeConfigOptions(configOptionsJson *C.char) (CErr *C.char) {
	defer shared.DeferPanicToError("changeConfigOptions", func(err error) {
		CErr = C.CString(err.Error())
	})

	configOptions = &shared.ConfigOptions{}
	err := json.Unmarshal([]byte(C.GoString(configOptionsJson)), configOptions)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export start
func start(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {
	defer shared.DeferPanicToError("start", func(err error) {
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
	options = shared.BuildConfig(*configOptions, options)

	shared.SaveCurrentConfig(sWorkingPath, options)

	err = startCommandServer()
	if err != nil {
		return stopAndAlert(StartCommandServer, err)
	}

	instance, err := NewService(options)
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
	defer shared.DeferPanicToError("stop", func(err error) {
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
	defer shared.DeferPanicToError("restart", func(err error) {
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

	time.Sleep(250 * time.Millisecond)

	err = start(configPath, disableMemoryLimit)
	if C.GoString(err) != "" {
		return err
	}

	return C.CString("")
}

//export startCommandClient
func startCommandClient(command C.int, port C.longlong) *C.char {
	err := StartCommand(int32(command), int64(port))
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
	defer shared.DeferPanicToError("selectOutbound", func(err error) {
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
	defer shared.DeferPanicToError("urlTest", func(err error) {
		CErr = C.CString(err.Error())
	})

	err := libbox.NewStandaloneCommandClient().URLTest(C.GoString(groupTag))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

func main() {}
