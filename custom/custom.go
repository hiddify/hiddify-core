package main

/*
#include "stdint.h"
*/
import "C"
import (
	"os"

	"github.com/hiddify/libcore/shared"
	"github.com/sagernet/sing-box/experimental/libbox"
)

var box *libbox.BoxService

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char) {
	Setup(C.GoString(baseDir), C.GoString(workingDir), C.GoString(tempDir))
}

//export parse
func parse(path *C.char) *C.char {
	err := shared.ParseConfig(C.GoString(path))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export create
func create(configPath *C.char) *C.char {
	path := C.GoString(configPath)
	content, err := os.ReadFile(path)
	if err != nil {
		return C.CString(err.Error())
	}
	options, err := parseConfig(string(content))
	if err != nil {
		return C.CString(err.Error())
	}
	overrides := shared.ConfigOverrides{
		LogOutput:      shared.StringAddr("box.log"),
		EnableTun:      shared.BoolAddr(false),
		SetSystemProxy: shared.BoolAddr(true),
	}
	template := shared.DefaultTemplate(overrides)
	options = shared.ApplyOverrides(template, options, overrides)

	shared.SaveCurrentConfig(sWorkingPath, options)

	instance, err := NewService(options)
	if err != nil {
		return C.CString(err.Error())
	}
	box = instance

	return C.CString("")
}

//export start
func start() *C.char {
	if box == nil {
		return C.CString("instance not found")
	}
	err := box.Start()
	if err != nil {
		return C.CString(err.Error())
	}

	return C.CString("")
}

//export stop
func stop() *C.char {
	if box == nil {
		return C.CString("instance not found")
	}

	err := box.Close()
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

func main() {}
