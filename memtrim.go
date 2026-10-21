package main

import (
	"context"
	"runtime/debug"
	"runtime/metrics"
	"time"
)

// A burst of work (start-up's catalog and index loads, a Browse, a check) leaves the Go heap holding the pages it
// needed at its peak, and the runtime hands them back to the OS only slowly. trimWhenQuiet watches the process's own
// CPU time and returns them once work has gone quiet: after a second busier than burstCPU, quietSeconds straight
// seconds below it.
const (
	burstCPU     = 0.2
	quietSeconds = 5
)

// quietWatch is trimWhenQuiet's state: whether a burst has been seen since the last trim, and for how long it has been
// quiet since.
type quietWatch struct {
	busy  bool
	quiet int
}

// step takes the CPU seconds used in the last second and reports whether to trim now.
func (w *quietWatch) step(cpu float64) bool {
	if cpu >= burstCPU {
		w.busy, w.quiet = true, 0
		return false
	}
	if !w.busy {
		return false
	}
	w.quiet++
	if w.quiet < quietSeconds {
		return false
	}
	w.busy, w.quiet = false, 0
	return true
}

var cpuSamples = []metrics.Sample{{Name: "/cpu/classes/user:cpu-seconds"}, {Name: "/cpu/classes/gc/total:cpu-seconds"}}

// cpuSeconds is the CPU time the process has spent in Go code and in the collector.
func cpuSeconds() float64 {
	metrics.Read(cpuSamples)
	total := 0.0
	for _, s := range cpuSamples {
		if s.Value.Kind() == metrics.KindFloat64 {
			total += s.Value.Float64()
		}
	}
	return total
}

func trimWhenQuiet(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var w quietWatch
	last := cpuSeconds()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		now := cpuSeconds()
		if w.step(now - last) {
			debug.FreeOSMemory()
			// The collection the trim runs is not work to wait out.
			now = cpuSeconds()
		}
		last = now
	}
}
