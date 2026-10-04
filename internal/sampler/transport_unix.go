//go:build !windows

package sampler

import (
	"context"
	"net"
)

func openDiagnostic(ctx context.Context, path string) (diagnosticConn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, "unix", path)
}
