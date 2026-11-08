package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/shots"
)

// console sends a command to the running game's console.
func (c *cmd) console() error {
	a, err := c.need(1, "a game", "a command")
	if err != nil {
		return err
	}
	if err := c.call("console.send", control.Params{Game: a[0], Value: strings.Join(a[1:], " ")}, nil, readTimeout); err != nil {
		return err
	}
	return c.emit(map[string]any{"sent": true}, func() { fmt.Fprintln(c.out, "Sent.") })
}

// screenshots lists a game's screenshots, newest first.
func (c *cmd) screenshots() error {
	a, err := c.need(1, "a game")
	if err != nil {
		return err
	}
	return show(c, "screenshots", control.Params{Game: a[0]}, func(g shots.Gallery) {
		rows := [][]string{}
		for _, s := range g.Shots {
			rows = append(rows, []string{s.Source, s.Name, bytefmt.Format(s.Size)})
		}
		c.table("SOURCE\tNAME\tBYTES", rows)
	})
}

// endorse endorses a Nexus mod at a version, or withdraws it with --remove.
func (c *cmd) endorse() error {
	a, err := c.need(1, "a game", "a Nexus mod id", "a version")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], ID: a[1], Version: a[2], Remove: c.removeFlag}
	var status string
	if err := c.call("nexus.endorse", p, &status, readTimeout); err != nil {
		return err
	}
	return c.emit(map[string]string{"status": status}, func() { fmt.Fprintln(c.out, status) })
}

// syncCmd shows what waits in the sync folder, or answers one waiting change.
func (c *cmd) syncCmd() error {
	if len(c.args) > 1 && c.args[1] == "apply" {
		a, err := c.need(2, "a game", "a synced profile id", "theirs or mine")
		if err != nil {
			return err
		}
		if a[2] != "theirs" && a[2] != "mine" {
			return usageError{"sync apply needs theirs or mine"}
		}
		if err := c.call("sync.apply", control.Params{Game: a[0], ID: a[1], Value: a[2]}, nil, installTimeout); err != nil {
			return err
		}
		return c.emit(map[string]any{"applied": a[2]}, func() { fmt.Fprintln(c.out, "Applied "+a[2]+".") })
	}
	return show(c, "sync.status", control.Params{}, func(s control.SyncStatus) {
		if len(s.Offers) == 0 && len(s.Stalled) == 0 {
			fmt.Fprintln(c.out, "Nothing is waiting.")
		}
		for _, o := range s.Offers {
			state := "changed"
			if o.Conflict {
				state = "conflict"
			} else if o.New {
				state = "new"
			}
			fmt.Fprintf(c.out, "%s\t%s\t%s\t%s\tfrom %s\n", state, o.Game, o.Remote, o.Name, o.Machine)
		}
		for _, st := range s.Stalled {
			fmt.Fprintf(c.out, "stalled\t%s\t%s\t%s\tfrom %s\n", st.Game, st.Remote, st.Name, st.Machine)
		}
	})
}
