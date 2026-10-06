package bepinex5

import (
	"context"
	"encoding/json"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// Perf asks the bridge for its in-game measurement, which it takes only on a launch Mortar measured.
func (l Loader) Perf(ctx context.Context, p loader.ProfileView, start bool) (json.RawMessage, error) {
	what := "perf"
	if start {
		what = "perf start"
	}
	return l.Query(ctx, loader.Target{}, p, what)
}
