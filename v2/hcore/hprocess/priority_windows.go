package hprocess

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	processPowerThrottling          = 4 // PROCESS_INFORMATION_CLASS ProcessPowerThrottling
	processPowerThrottlingVersion   = 1
	processPowerThrottlingExecSpeed = 0x1
)

type processPowerThrottlingState struct {
	Version     uint32
	ControlMask uint32
	StateMask   uint32
}

// RaisePriority keeps the process responsive while the app is in the background: a priority above
// normal, and no EcoQoS (Windows 11 efficiency mode), which slows a process without a visible window.
func RaisePriority() {
	process := windows.CurrentProcess()
	_ = windows.SetPriorityClass(process, windows.ABOVE_NORMAL_PRIORITY_CLASS)
	setProcessInformation := windows.NewLazySystemDLL("kernel32.dll").NewProc("SetProcessInformation")
	if setProcessInformation.Find() != nil {
		return // before Windows 8
	}
	// ControlMask set and StateMask clear: execution speed throttling is off.
	state := processPowerThrottlingState{
		Version:     processPowerThrottlingVersion,
		ControlMask: processPowerThrottlingExecSpeed,
	}
	_, _, _ = setProcessInformation.Call(uintptr(process), processPowerThrottling,
		uintptr(unsafe.Pointer(&state)), unsafe.Sizeof(state))
}
