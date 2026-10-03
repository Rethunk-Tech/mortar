package cli

import (
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/loadorder"
)

func (c *cmd) printLoadOrder(rows []loadorder.Row) {
	t := make([][]string, 0, len(rows))
	for _, row := range rows {
		notes := make([]string, 0, 2)
		if row.Cycle {
			notes = append(notes, "cycle")
		}
		if len(row.MissingRequired) > 0 {
			notes = append(notes, "missing "+strings.Join(row.MissingRequired, ", "))
		}
		t = append(t, []string{
			strconv.Itoa(row.Position),
			row.Name,
			row.UniqueID,
			strings.Join(row.Required, ", "),
			strings.Join(row.Optional, ", "),
			strings.Join(row.Dependents, ", "),
			strings.Join(notes, "; "),
		})
	}
	c.table("#\tNAME\tUNIQUEID\tREQUIRED\tOPTIONAL\tDEPENDENTS\tNOTES", t)
}
