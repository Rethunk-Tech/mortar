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
