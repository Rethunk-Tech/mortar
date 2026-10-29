package sysaccent

import (
	"context"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/portal"
)

// Read is the desktop's accent colour as "#rrggbb"; false when the portal is absent or the desktop sets none.
func Read() (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	v, err := portal.ReadSetting(ctx, "org.freedesktop.appearance", "accent-color")
	if err != nil {
		return "", false
	}
	return FromPortal(v)
}
