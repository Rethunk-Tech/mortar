package cli

import (
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/loadorder"
)

func loadOrderNames(rows []loadorder.Row) map[string]string {
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.UniqueID != "" && row.Name != "" {
			names[row.UniqueID] = row.Name
		}
	}
	return names
}

func joinModNames(ids []string, names map[string]string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if n := names[id]; n != "" {
			parts = append(parts, n)
			continue
		}
		parts = append(parts, id)
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
			row.UniqueID,
			joinModNames(row.Required, names),
			joinModNames(row.Optional, names),
			joinModNames(row.Dependents, names),
			strings.Join(notes, "; "),
		})
	}
	c.table("#\tNAME\tMOD ID\tREQUIRED\tOPTIONAL\tDEPENDENTS\tNOTES", t)
}
