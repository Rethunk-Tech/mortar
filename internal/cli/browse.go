package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/browse"
	"github.com/Rethunk-AI/mortar/internal/control"
)

func (c *cmd) browse() error {
	if len(c.args) < 3 {
		return usageError{"browse needs a game, search text"}
	}
	game := c.args[1]
	text := strings.Join(c.args[2:], " ")
	source := c.sourceFlag
	if source == "" {
		source = "nexus"
	}
	page := max(c.pageFlag, 1)
	var result browse.Page
	if err := c.call("browse", control.Params{
		Game: game, Query: text, Value: source, ModID: page, Profile: c.profileFlag,
	}, &result, readTimeout); err != nil {
		return err
	}
	return c.emit(result, func() {
		if len(result.Items) == 0 {
			fmt.Fprintln(c.out, "No mods matched.")
			return
		}
		rows := make([][]string, 0, len(result.Items))
		for _, item := range result.Items {
			mark := ""
			if item.Installed {
				mark = "yes"
			}
			count := strconv.Itoa(item.Endorsements)
			if item.Source == "github" {
				count = strconv.Itoa(item.Stars)
			}
			rows = append(rows, []string{item.Source, item.ID, item.Name, item.Author, count, mark})
		}
		c.table(fmt.Sprintf("SOURCE\tID\tNAME\tAUTHOR\tSCORE\tIN PROFILE  (%d of %d)", len(result.Items), result.Total), rows)
	})
}
