package hcore

import "github.com/sagernet/sing/common"

// The device's memory, as the platform sees it. Set before Setup; see SetDeviceMemory.
var (
	// deviceMemoryLimit is what the core may use, in bytes; 0 for no limit of its own. The "Memory
	// limit" setting holds the Go runtime to three quarters of it (libbox.SetMemoryLimit).
	deviceMemoryLimit int64
	// deviceLowMemory turns on the low-memory relay path of a core not built for it.
	deviceLowMemory bool
)

// SetDeviceMemory tells the core how much memory it may use and whether the device is short of
// memory. Android passes it: one core serves phones and low-RAM TVs, and without a limit of its own
// the "Memory limit" setting held the Go runtime to nothing there.
func SetDeviceMemory(limit int64, lowMemory bool) {
	deviceMemoryLimit = limit
	deviceLowMemory = lowMemory
	if lowMemory {
		common.SetLowMemory(true)
	}
}
