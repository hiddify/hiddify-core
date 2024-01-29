// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

// simple does nothing except block while running the service.
package main

import "C"
import "github.com/hiddify/libcore/admin_service"

//export AdminServiceStart
func AdminServiceStart(arg *C.char) {
	goArg := C.GoString(arg)
	admin_service.StartService(goArg)
}
