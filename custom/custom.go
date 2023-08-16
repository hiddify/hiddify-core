package main

/*
#include "stdint.h"
*/
import "C"
import (
	"github.com/hiddify/libcore/shared"
	B "github.com/sagernet/sing-box/experimental/libbox"
)

var box *BoxService

var templateOptions = shared.ConfigTemplateOptions{IncludeTunInbound: false, IncludeMixedInbound: true, IncludeLogOutput: true}

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char) {
	Setup(C.GoString(baseDir), C.GoString(workingDir), C.GoString(tempDir))
}

//export checkConfig
func checkConfig(configPath *C.char) *C.char {
	configContent, err := shared.ConvertToSingbox(C.GoString(configPath), templateOptions)
	if err != nil {
		return C.CString(err.Error())
	}

	err = B.CheckConfig(string(configContent))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export create
func create(configPath *C.char) *C.char {
	path := C.GoString(configPath)

	configContent, err := shared.ConvertToSingbox(path, templateOptions)
	if err != nil {
		return C.CString(err.Error())
	}

	instance, err := NewService(string(configContent))
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
