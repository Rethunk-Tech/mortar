//go:build windows

package sampler

import (
	"context"
	"os"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// openDiagnostic opens the runtime's diagnostics named pipe, which Windows exposes as a file path.
func openDiagnostic(ctx context.Context, path string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return fsx.OpenFile(path, os.O_RDWR, 0)
}
