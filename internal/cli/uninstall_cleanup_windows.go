//go:build windows

package cli

import (
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
)

func removePlatformLeftovers() error {
	return errors.Join(settings.RemoveAutostart(), shortcut.RemoveStartMenu())
}
