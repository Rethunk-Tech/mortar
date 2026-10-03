package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Offline runs the read-only data-folder checks when the app is not running.
func Offline() (Report, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return Report{}, err
	}
	return Scan(dir), nil
}

// Scan checks a data folder the same way the CLI's offline doctor does.
func Scan(dir string) Report {
	var checks []Check
	settingsPath := filepath.Join(dir, "settings.json")
	if b, err := fsx.ReadFile(settingsPath); err != nil {
		checks = append(checks, Check{
			ID: "settings", Status: Fail, Detail: "settings.json is unreadable or missing",
			Fix: "restore settings.json from a known-good copy",
		})
	} else if !json.Valid(b) {
		checks = append(checks, Check{
			ID: "settings", Status: Fail, Detail: "settings.json is corrupt",
			Fix: "restore settings.json from a known-good copy",
		})
	} else {
		checks = append(checks, Check{ID: "settings", Status: Pass, Detail: "settings.json is readable"})
	}
	if copies, _ := filepath.Glob(settingsPath + ".*"); len(copies) > 0 {
		checks = append(checks, Check{
			ID: "settingsCopies", Status: Warn,
			Detail: fmt.Sprintf("%d settings temporary/copy files are present", len(copies)),
			Fix:    "keep the newest valid settings.json and remove abandoned copies",
		})
	} else {
		checks = append(checks, Check{ID: "settingsCopies", Status: Pass, Detail: "no settings temporary/copy files"})
	}
	var damaged []string
	_ = filepath.WalkDir(filepath.Join(dir, "profiles"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && !entry.IsDir() && entry.Name() == "profile.json" {
			if b, readErr := fsx.ReadFile(path); readErr != nil || !json.Valid(b) {
				damaged = append(damaged, path)
			}
		}
		return nil
	})
	if len(damaged) > 0 {
		for _, path := range damaged {
			checks = append(checks, Check{
				ID: "profile:" + path, Status: Fail, Detail: "damaged profile.json: " + path,
				Fix: "restore or remove the damaged profile",
			})
		}
	} else {
		checks = append(checks, Check{ID: "profiles", Status: Pass, Detail: "profiles are readable"})
	}
	if info, statErr := os.Stat(filepath.Join(dir, control.FileName)); statErr == nil && time.Since(info.ModTime()) > 24*time.Hour {
		checks = append(checks, Check{
			ID: "control", Status: Warn, Detail: "control.json is stale",
			Fix: "start Mortar once to refresh control.json",
		})
	} else if statErr == nil {
		checks = append(checks, Check{ID: "control", Status: Pass, Detail: "control.json is current"})
	} else {
		checks = append(checks, Check{ID: "control", Status: Pass, Detail: "control.json is absent"})
	}
	free, freeErr := freeSpace(dir)
	if freeErr == nil {
		if free == 0 {
			checks = append(checks, Check{
				ID: "disk", Status: Fail, Detail: "data folder has no free space",
				Fix: "free disk space before starting Mortar",
			})
		} else {
			checks = append(checks, Check{ID: "disk", Status: Pass, Detail: "data folder has free space"})
		}
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, "store", "index.json")); err != nil || !json.Valid(b) {
		checks = append(checks, Check{
			ID: "store", Status: Fail, Detail: "store index.json is unreadable or corrupt",
			Fix: "restore the store index or let Mortar rebuild it",
		})
	} else {
		checks = append(checks, Check{ID: "store", Status: Pass, Detail: "store index.json is readable"})
	}
	return Report{Checks: checks}
}

// Findings are the non-passing checks, in report order.
func Findings(r Report) (findings, fixes []string) {
	for _, c := range r.Checks {
		if c.Status == Pass {
			continue
		}
		findings = append(findings, c.Detail)
		if c.Fix != "" {
			fixes = append(fixes, c.Fix)
		}
	}
	return findings, fixes
}
