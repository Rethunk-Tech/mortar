package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/archivesvc"
	"github.com/Rethunk-Tech/mortar/internal/bundles"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/templates"
)

func (c *cmd) sub(want string) (string, error) {
	if len(c.args) < 2 {
		return "", usageError{c.args[0] + " needs " + want}
	}
	return c.args[1], nil
}

func (c *cmd) keepCount() (int, error) {
	n, err := strconv.Atoi(c.keepFlag)
	if err != nil || n < 1 {
		return 0, usageError{c.args[0] + " " + c.args[1] + " needs --keep <count>"}
	}
	return n, nil
}

func (c *cmd) templatesCmd() error {
	sub, err := c.sub("list, save, delete or new")
	if err != nil {
		return err
	}
	switch sub {
	case "list":
		a, err := c.need(2, "a game")
		if err != nil {
			return err
		}
		return show(c, "templates", control.Params{Game: a[0]}, func(list []templates.Template) {
			rows := make([][]string, 0, len(list))
			for _, t := range list {
				rows = append(rows, []string{t.Name, strconv.Itoa(len(t.Bundle))})
			}
			c.table("NAME\tMODS", rows)
		})
	case "save":
		a, err := c.need(2, "a game", "a profile", "a template name")
		if err != nil {
			return err
		}
		return show(c, "templates.save", control.Params{Game: a[0], Profile: a[1], Name: strings.Join(a[2:], " ")}, func(t templates.Template) {
			fmt.Fprintf(c.out, "Saved template %s with %d mods.\n", t.Name, len(t.Bundle))
		})
	case "delete":
		a, err := c.need(2, "a game", "a template name")
		if err != nil {
			return err
		}
		name := strings.Join(a[1:], " ")
		if err := c.call("templates.delete", control.Params{Game: a[0], Name: name}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]any{"deleted": name}, func() { fmt.Fprintf(c.out, "Deleted template %s.\n", name) })
	case "new":
		a, err := c.need(2, "a game", "a template name", "a profile name")
		if err != nil {
			return err
		}
		var res bundles.ApplyResult
		p := control.Params{Game: a[0], Name: a[1], Value: strings.Join(a[2:], " ")}
		if err := c.call("templates.new", p, &res, installTimeout); err != nil {
			return err
		}
		return c.emit(res, func() { fmt.Fprintf(c.out, "Created %s from template %s.\n", p.Value, a[1]) })
	}
	return usageError{"unknown templates command " + sub}
}

func (c *cmd) historyUsage() error {
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	return show(c, "history.usage", control.Params{Game: a[0]}, func(rows []profile.HistoryUsage) {
		t := make([][]string, 0, len(rows))
		for _, u := range rows {
			t = append(t, []string{u.ProfileID, u.ProfileName, strconv.Itoa(u.Events), humanBytes(u.SnapshotBytes), humanBytes(u.FileBytes)})
		}
		c.table("ID\tPROFILE\tEVENTS\tSNAPSHOTS\tFILES", t)
	})
}

func (c *cmd) historyTrim() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	keep, err := c.keepCount()
	if err != nil {
		return err
	}
	return show(c, "history.trim", control.Params{Game: a[0], Profile: a[1], Keep: keep}, func(u profile.HistoryUsage) { fmt.Fprintf(c.out, "%s keeps %d events.\n", u.ProfileName, u.Events) })
}

func (c *cmd) backupsUsage(game string) error {
	return show(c, "backups.usage", control.Params{Game: game}, func(u savessvc.BackupsUsage) {
		rows := make([][]string, 0, len(u.PerSave))
		for _, s := range u.PerSave {
			rows = append(rows, []string{s.Save, strconv.Itoa(s.Count), humanBytes(s.Bytes)})
		}
		c.table("SAVE\tBACKUPS\tSIZE", rows)
		fmt.Fprintf(c.out, "Total %s\n", humanBytes(u.TotalBytes))
	})
}

func (c *cmd) backupsTrim(game string) error {
	keep, err := c.keepCount()
	if err != nil {
		return err
	}
	return show(c, "backups.trim", control.Params{Game: game, Keep: keep}, func(r savessvc.TrimResult) {
		fmt.Fprintf(c.out, "Deleted %d backups, freed %s.\n", r.Removed, humanBytes(r.FreedBytes))
	})
}

func (c *cmd) queueRetryFailed() error {
	return show(c, "queue.retry-failed", control.Params{}, func(r queue.RetryAllResult) { fmt.Fprintf(c.out, "Requeued %d failed downloads.\n", r.Requeued) })
}

func (c *cmd) dataLocation() error {
	return show(c, "data.location", control.Params{}, func(l datasvc.DataLocation) {
		fmt.Fprintln(c.out, l.Dir)
		if l.Portable {
			fmt.Fprintln(c.out, "Portable.")
		}
	})
}

func (c *cmd) archiveCmd() error {
	sub, err := c.sub("preview or downloads")
	if err != nil {
		return err
	}
	switch sub {
	case "preview":
		a, err := c.need(2, "an archive path")
		if err != nil {
			return err
		}
		return show(c, "archive.preview", control.Params{Path: a[0]}, func(pv archive.Preview) {
			rows := make([][]string, 0, len(pv.Manifests))
			for _, m := range pv.Manifests {
				rows = append(rows, []string{m.UniqueID, m.Name, m.Version, m.Folder})
			}
			c.table("UNIQUEID\tNAME\tVERSION\tFOLDER", rows)
			fmt.Fprintf(c.out, "%d entries, %s", len(pv.Entries), humanBytes(pv.TotalSize))
			if pv.Fomod {
				fmt.Fprint(c.out, ", FOMOD installer")
			}
			fmt.Fprintln(c.out)
		})
	case "downloads":
		a, err := c.need(2, "a game")
		if err != nil {
			return err
		}
		return show(c, "archive.downloads", control.Params{Game: a[0]}, func(list []archivesvc.DownloadArchive) {
			rows := make([][]string, 0, len(list))
			for _, d := range list {
				rows = append(rows, []string{d.Name, humanBytes(d.Size), strconv.Itoa(d.ModID), d.Path})
			}
			c.table("NAME\tSIZE\tNEXUS ID\tPATH", rows)
		})
	}
	return usageError{"unknown archive command " + sub}
}

func (c *cmd) libraryCmd() error {
	sub, err := c.sub("extra, hidden, old-files or strays")
	if err != nil {
		return err
	}
	switch sub {
	case "extra":
		a, err := c.need(2, "a game")
		if err != nil {
			return err
		}
		return show(c, "library.extra", control.Params{Game: a[0]}, func(pv profile.GameModsPreview) { c.printGameMods(pv) })
	case "hidden":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		return show(c, "library.hidden", control.Params{Game: a[0], Profile: a[1]}, func(mods []profile.HiddenMod) {
			rows := make([][]string, 0, len(mods))
			for _, m := range mods {
				rows = append(rows, []string{m.Key, m.Folder, m.UniqueID, m.Name, m.Version})
			}
			c.table("KEY\tFOLDER\tUNIQUEID\tNAME\tVERSION", rows)
		})
	case "old-files":
		return c.libraryOldFiles()
	case "strays":
		return c.libraryStrays()
	}
	return usageError{"unknown library command " + sub}
}

func (c *cmd) libraryOldFiles() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	if c.keepFlag != "" && c.deleteFlag != "" {
		return usageError{"library old-files takes --keep or --delete, not both"}
	}
	if key := c.keepFlag + c.deleteFlag; key != "" {
		p := control.Params{Game: a[0], Profile: a[1], Key: key, Set: c.keepFlag != ""}
		if err := c.call("library.old-files.resolve", p, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]any{"key": key, "kept": p.Set}, func() {
			if p.Set {
				fmt.Fprintf(c.out, "Kept the old files of %s.\n", key)
			} else {
				fmt.Fprintf(c.out, "Deleted the old files of %s.\n", key)
			}
		})
	}
	return show(c, "library.old-files", control.Params{Game: a[0], Profile: a[1]}, func(sets []profile.OldFiles) {
		rows := make([][]string, 0, len(sets))
		for _, s := range sets {
			rows = append(rows, []string{s.Key, s.Label, strconv.Itoa(len(s.Files))})
		}
		c.table("KEY\tMOD\tFILES", rows)
	})
}

func (c *cmd) libraryStrays() error {
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	switch {
	case c.dismissFlag != "":
		if err := c.call("library.strays.dismiss", control.Params{Game: a[0], Name: c.dismissFlag}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]any{"dismissed": c.dismissFlag}, func() { fmt.Fprintf(c.out, "Dismissed %s.\n", c.dismissFlag) })
	case len(c.moveFlag) > 0:
		if len(a) < 2 {
			return usageError{"library strays --move needs a profile"}
		}
		var res profile.GameModsResult
		if err := c.call("library.strays.move", control.Params{Game: a[0], Profile: a[1], UniqueIDs: c.moveFlag}, &res, installTimeout); err != nil {
			return err
		}
		return c.emit(res, func() {
			fmt.Fprintf(c.out, "Moved %d folders, skipped %d, failed %d.\n", res.Imported, res.Skipped, res.Failed)
		})
	}
	return show(c, "library.strays", control.Params{Game: a[0]}, func(pv profile.GameModsPreview) { c.printGameMods(pv) })
}

func (c *cmd) printGameMods(pv profile.GameModsPreview) {
	rows := make([][]string, 0, len(pv.Mods))
	for _, m := range pv.Mods {
		rows = append(rows, []string{m.Folder, m.UniqueID, m.Name, m.Version, m.Status})
	}
	c.table("FOLDER\tUNIQUEID\tNAME\tVERSION\tSTATUS", rows)
}
