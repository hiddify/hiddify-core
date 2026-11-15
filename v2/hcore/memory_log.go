package hcore

import (
	"fmt"
	"runtime/metrics"
	"sync"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common/memory"
)

// The memory of an iOS network extension is logged while it is high.
//
// iOS ends a network extension that uses more than 50 MB, and the log is all that is left to say
// what grew. So once a minute, while the process's footprint (what iOS counts) is above
// memoryLogFrom, the core logs it with what the Go runtime holds. Below it, a minute costs one
// system call. Reading the counters of runtime/metrics does not stop the program.
const (
	memoryLogFrom  = 40 << 20
	memoryLogEvery = time.Minute
)

var memoryLogOnce sync.Once

// startMemoryLog starts the log of a high footprint, once per process, on iOS.
func startMemoryLog() {
	if !C.IsIos || !memory.TotalAvailable() {
		return
	}
	memoryLogOnce.Do(func() {
		go func() {
			samples := memoryLogSamples()
			for range time.Tick(memoryLogEvery) {
				if footprint := memory.Total(); footprint >= memoryLogFrom {
					Log(LogLevel_WARNING, LogType_CORE, memoryLogLine(footprint, samples))
				}
			}
		}()
	})
}

var memoryLogNames = []string{
	"/memory/classes/heap/objects:bytes", // live and not yet collected objects
	"/memory/classes/heap/unused:bytes",  // spans held for objects, unused
	"/memory/classes/heap/free:bytes",    // free, not yet returned to the system
	"/memory/classes/heap/stacks:bytes",  // goroutine stacks
	"/memory/classes/total:bytes",        // all the runtime got from the system
	"/gc/heap/goal:bytes",                // the heap size at which the next collection starts
	"/sched/goroutines:goroutines",
}

func memoryLogSamples() []metrics.Sample {
	samples := make([]metrics.Sample, len(memoryLogNames))
	for i, name := range memoryLogNames {
		samples[i].Name = name
	}
	return samples
}

// memoryLogLine reads the runtime's counters into samples and describes them.
func memoryLogLine(footprint uint64, samples []metrics.Sample) string {
	metrics.Read(samples)
	value := func(i int) uint64 {
		if samples[i].Value.Kind() != metrics.KindUint64 {
			return 0
		}
		return samples[i].Value.Uint64()
	}
	mb := func(v uint64) string { return fmt.Sprintf("%.1f", float64(v)/(1<<20)) }
	return fmt.Sprintf("memory: footprint %s MB, %s MB left; Go: heap %s MB live, %s MB unused, %s MB free, stacks %s MB, total %s MB, goal %s MB, %d goroutines",
		mb(footprint), mb(memory.Available()), mb(value(0)), mb(value(1)), mb(value(2)), mb(value(3)), mb(value(4)), mb(value(5)), value(6))
}
