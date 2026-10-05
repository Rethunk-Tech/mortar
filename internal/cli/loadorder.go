package cli

import (
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func loadOrderNames(rows []loadorder.Row) map[mod.ID]string {
	names := make(map[mod.ID]string, len(rows))
	for _, row := range rows {
		if row.ID != "" && row.Name != "" {
			names[row.ID] = row.Name
		}
	}
	return names
}

func joinModNames(ids []mod.ID, names map[mod.ID]string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if n := names[id]; n != "" {
			parts = append(parts, n)
			continue
		}
		parts = append(parts, id.Local())
	}
	return strings.Join(parts, ", ")
}

func (c *cmd) printLoadOrder(rows []loadorder.Row) {
	names := loadOrderNames(rows)
	t := make([][]string, 0, len(rows))
	for _, row := range rows {
		notes := make([]string, 0, 2)
		if row.Cycle {
			notes = append(notes, "dependency cycle")
		}
		if len(row.MissingRequired) > 0 {
			notes = append(notes, "missing "+joinModNames(row.MissingRequired, names))
		}
		t = append(t, []string{
			strconv.Itoa(row.Position),
			row.Name,
			row.ID.Local(),
			joinModNames(row.Required, names),
			joinModNames(row.Optional, names),
			joinModNames(row.Dependents, names),
			strings.Join(notes, "; "),
		})
	}
	c.table("#\tNAME\tMOD ID\tREQUIRED\tOPTIONAL\tDEPENDENTS\tNOTES", t)
}
