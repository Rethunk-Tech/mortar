//go:build !windows

package backdrop

import (
	"context"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Runner runs gsettings with args and returns its standard output.
type Runner func(ctx context.Context, args ...string) (string, error)

func execRunner(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/gsettings", args...).Output()
	return string(out), err
}

// gsettingsValue reads one GNOME setting; the tool prints strings single-quoted.
func gsettingsValue(run Runner, schema, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := run(ctx, "get", schema, key)
	return strings.Trim(strings.TrimSpace(out), "'"), err
}

// gnomeWallpaper returns the path of the GNOME desktop wallpaper, the dark variant under a dark colour scheme.
// It is empty when GNOME's settings are unavailable or the wallpaper is not a local file.
func gnomeWallpaper(run Runner) string {
	key := "picture-uri"
	if scheme, err := gsettingsValue(run, "org.gnome.desktop.interface", "color-scheme"); err == nil && scheme == "prefer-dark" {
		key = "picture-uri-dark"
	}
	uri, err := gsettingsValue(run, "org.gnome.desktop.background", key)
	if err != nil {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	return u.Path
}
