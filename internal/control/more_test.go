package control

import (
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

type listHandler struct {
	owners    map[string]nxm.Owner
	ownerErrs map[string]error
}

func (h listHandler) Owner(scheme string) (nxm.Owner, error) {
	if err, ok := h.ownerErrs[scheme]; ok {
		return nxm.Owner{}, err
	}
	return h.owners[scheme], nil
}
func (listHandler) Register() error                           { return nil }
func (listHandler) RegisterSchemes([]string) error            { return nil }
func (listHandler) Restore(map[string]string) error           { return nil }
func (listHandler) Release([]string, map[string]string) error { return nil }
func (listHandler) ForwardOther(string, string) error         { return nil }
func (listHandler) RegisterLinks() error                      { return nil }

func TestLinksListsOwnersPerScheme(t *testing.T) {
	testfs.DataHome(t)
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := &Services{Nxm: nxmsvc.NewService(store, listHandler{
		owners: map[string]nxm.Owner{
			"nxm":    {Name: "Mortar", ID: "mortar", Mine: true},
			"ror2mm": {Name: "r2modman", ID: "r2"},
		},
		ownerErrs: map[string]error{"curseforge": errors.New("no handler")},
	})}
	res, err := svc.Handle(t.Context(), "links", Params{})
	if err != nil {
		t.Fatalf("links: %v", err)
	}
	rows, ok := res.([]nxmsvc.LinkOwner)
	if !ok || len(rows) == 0 {
		t.Fatalf("rows %T %+v", res, res)
	}
	byScheme := map[string]nxmsvc.LinkOwner{}
	for _, row := range rows {
		byScheme[row.Scheme] = row
	}
	if row := byScheme["nxm"]; !row.Mine || row.Name != "Mortar" {
		t.Fatalf("nxm: %+v", row)
	}
	if row := byScheme["ror2mm"]; row.Mine || row.Name != "r2modman" {
		t.Fatalf("ror2mm: %+v", row)
	}
	if row := byScheme["curseforge"]; row.Name != "no handler" {
		t.Fatalf("curseforge error: %+v", row)
	}
}
