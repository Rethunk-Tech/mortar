package cli

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
)

// profileImport imports another manager's pack: an r2modman code or its key, or a .r2z, profile folder or modpack
// path the app reads itself.
func (c *cmd) profileImport() error {
	if c.fromFlag != "" {
		return c.profileImportExternal()
	}
	a, err := c.need(2, "an r2modman code, key or pack file")
	if err != nil {
		return err
	}
	in := strings.Join(a, " ")
	p := control.Params{Game: c.game, Profile: c.profileFlag, Name: c.nameFlag, Preview: c.previewFlag, Value: in}
	if _, err := os.Stat(in); err == nil {
		if p.Path, err = filepath.Abs(in); err != nil {
			return err
		}
		p.Value = ""
	}
	if c.previewFlag {
		return show(c, "pack.import", p, func(pv packsvc.Preview) {
			fmt.Fprintf(c.out, "%s (%s): %d packages, %d config files\n", pv.Name, cmp.Or(pv.Game, c.game), len(pv.Packages), pv.Configs)
			for _, pkg := range pv.Packages {
				off := ""
				if pkg.Disabled {
					off = "  (disabled)"
				}
				fmt.Fprintf(c.out, "  %s %s%s\n", pkg.Native, pkg.Version, off)
			}
		})
	}
	var res packsvc.Result
	if err := c.call("pack.import", p, &res, installTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprintf(c.out, "%s\tqueued %d downloads\n", res.Profile, res.Queued)
		if len(res.Unsupported) > 0 {
			fmt.Fprintf(c.out, "Not installable from here: %s\n", strings.Join(res.Unsupported, ", "))
		}
	})
}

// profileExport writes the profile as a Thunderstore modpack zip.
func (c *cmd) profileExport() error {
	a, err := c.need(2, "a game", "a profile", "the modpack zip to write")
	if err != nil {
		return err
	}
	if c.format != "modpack" {
		return usageError{"profile export needs --format modpack"}
	}
	dest, err := filepath.Abs(a[2])
	if err != nil {
		return err
	}
	var res packsvc.ModpackResult
	if err := c.call("pack.exportModpack", control.Params{Game: a[0], Profile: a[1], Path: dest, All: !c.noConfigs}, &res, readTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprintf(c.out, "Wrote %s: %d packages, %d config files.\n", res.Path, len(res.Dependencies), res.Configs)
		if len(res.LeftOut) > 0 {
			fmt.Fprintf(c.out, "Left out (not on Thunderstore): %s\n", strings.Join(res.LeftOut, ", "))
		}
		if len(res.Disabled) > 0 {
			fmt.Fprintf(c.out, "Left out (switched off): %s\n", strings.Join(res.Disabled, ", "))
		}
	})
}

// profileBackup writes the profile to one file, with its mod files unless --no-mods.
func (c *cmd) profileBackup() error {
	a, err := c.need(2, "a game", "a profile", "the backup file to write")
	if err != nil {
		return err
	}
	dest, err := filepath.Abs(a[2])
	if err != nil {
		return err
	}
	return show(c, "profile.backup", control.Params{Game: a[0], Profile: a[1], Path: dest, All: !c.noMods}, func(r map[string]string) {
		fmt.Fprintf(c.out, "Backed up to %s.\n", r["path"])
	})
}

// profileRestore makes a new profile from a backup file and queues its mods' downloads.
func (c *cmd) profileRestore() error {
	a, err := c.need(2, "a backup file")
	if err != nil {
		return err
	}
	path, err := filepath.Abs(a[0])
	if err != nil {
		return err
	}
	var res packsvc.RestoreResult
	if err := c.call("profile.restore", control.Params{Game: c.game, Path: path}, &res, installTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprintf(c.out, "%s\t%s\tqueued %d downloads\n", res.Profile, res.Name, res.Queued)
		if len(res.Unavailable) > 0 {
			fmt.Fprintf(c.out, "Cannot download again (install them by hand): %s\n", strings.Join(res.Unavailable, ", "))
		}
	})
}

// profileImportExternal lists the --from manager's profiles for the game, or imports the one named as a new profile.
func (c *cmd) profileImportExternal() error {
	g, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	p := control.Params{Game: g, Source: c.fromFlag, Preview: c.previewFlag}
	if len(c.args) < 3 {
		return show(c, "external.sources", p, func(sources []migrate.SourceInfo) {
			for _, src := range sources {
				for _, prof := range src.Profiles {
					fmt.Fprintf(c.out, "%s\t%s\t%d mods\n", prof.ID, prof.Name, prof.Mods)
				}
			}
		})
	}
	p.ID = strings.Join(c.args[2:], " ")
	if c.previewFlag {
		return show(c, "external.import", p, func(pv migrate.ProfilePreview) {
			fmt.Fprintf(c.out, "%s (%s): %d mods\n", pv.Name, pv.Source, len(pv.Mods))
			for _, m := range pv.Mods {
				off := ""
				if !m.Enabled {
					off = "  (disabled)"
				}
				fmt.Fprintf(c.out, "  %s %s%s\n", m.Name, m.Version, off)
			}
		})
	}
	var res sharesvc.Result
	if err := c.call("external.import", p, &res, installTimeout); err != nil {
		return err
	}
	return c.emit(res, func() { fmt.Fprintf(c.out, "%s\tqueued %d downloads\n", res.Profile.Name, res.Queued) })
}
