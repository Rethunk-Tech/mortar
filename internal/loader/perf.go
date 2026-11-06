package loader

import (
	"context"
	"encoding/json"
)

// InGamePerf is a loader whose companion, on a measured launch, measures frame times, memory and each mod's
// main-thread cost per frame, and answers for them as JSON; start restarts the measurement window.
type InGamePerf interface {
	Perf(ctx context.Context, p ProfileView, start bool) (json.RawMessage, error)
}

// InGamePerfStop is an InGamePerf whose companion can end the measurement, keeping the last summary readable; the
// SMAPI bridge's profiler has no stop.
type InGamePerfStop interface {
	InGamePerf
	// PerfStop answers {"measured":bool}: false only when measuring never started.
	PerfStop(ctx context.Context, p ProfileView) (json.RawMessage, error)
}
