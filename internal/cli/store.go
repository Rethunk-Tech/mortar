package cli

import (
	"fmt"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func (c *cmd) storeCmd() error {
	if len(c.args) < 2 {
		return usageError{"store needs report or remove"}
	}
	switch c.args[1] {
	case "report":
		return c.storeReport()
	case "remove":
		return c.storeRemove()
	default:
		return usageError{"store needs report or remove"}
	}
}

func (c *cmd) storeReport() error {
	return show(c, "store.report", control.Params{Game: c.game}, func(rep store.Report) { printStoreReport(c, rep) })
}

func (c *cmd) storeRemove() error {
	a, err := c.need(2, "a game", "a store key")
	if err != nil {
		return err
	}
	if err := c.call("store.remove", control.Params{Game: a[0], IDs: a[1:]}, nil, readTimeout); err != nil {
		return err
	}
	return c.emit(map[string]any{"removed": a[1:]}, func() {
		fmt.Fprintf(c.out, "Removed %d store items.\n", len(a[1:]))
	})
}

func printStoreReport(c *cmd, rep store.Report) {
	var rows [][]string
	for gameID, g := range rep {
		for _, it := range g.Unused {
			rows = append(rows, storeReportRow(gameID, it, "unused"))
		}
		for gi, group := range g.Duplicates {
			label := "duplicate " + strconv.Itoa(gi+1)
			for _, it := range group {
				rows = append(rows, storeReportRow(gameID, it, label))
			}
		}
	}
	c.table("GAME\tKEY\tNAME\tVERSION\tSIZE\tLAST USED\tWHY", rows)
	if len(rows) == 0 {
		fmt.Fprintln(c.out, "No unused or duplicate store items.")
	}
}

func storeReportRow(gameID string, it store.Item, why string) []string {
	return []string{gameID, it.Key, it.Name, it.Version, humanBytes(it.Size), it.LastUsed.UTC().Format("2006-01-02"), why}
}
