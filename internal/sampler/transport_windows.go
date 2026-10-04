//go:build windows

package sampler

import (
	"context"
	"os"
)

func openDiagnostic(ctx context.Context, path string) (diagnosticConn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_RDWR, 0)
}
