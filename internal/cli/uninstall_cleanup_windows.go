//go:build windows

package cli

import (
	"errors"

	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/shortcut"
)

func removePlatformLeftovers() error {
	return errors.Join(settings.RemoveAutostart(), shortcut.RemoveStartMenu())
}
