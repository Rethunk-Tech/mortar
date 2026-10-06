package sharesvc

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func TestImportDataMakesTheSharedProfileWithItsSwitchedOffMods(t *testing.T) {
	s, rec := newService(t, true)
	off := mod.SMAPI("B.B")
	p := profile.Profile{Name: "From the desk", Entries: []profile.Entry{
		{Key: "ka", Source: profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1}, Mods: []profile.Component{{ID: mod.SMAPI("A.A"), Folder: "."}}},
		{Key: "kb", Source: profile.Source{Kind: profile.KindNexus, ModID: 600, FileID: 6}, Mods: []profile.Component{{ID: off, Folder: "."}}, Disabled: []mod.ID{off}},
	}}
	var payload bytes.Buffer
	if _, err := share.Write(&payload, "stardew", p, t.TempDir(), share.OwnInclude()); err != nil {
		t.Fatal(err)
	}
	res, err := s.ImportData(context.Background(), "stardew", base64.RawStdEncoding.EncodeToString(payload.Bytes()))
	if err != nil || res.Profile.Name != "From the desk" {
		t.Fatalf("import = %+v, %v", res, err)
	}
	queued := map[int]bool{}
	for _, r := range rec.reqs {
		queued[r.ModID] = true
	}
	if !queued[100] || !queued[600] {
		t.Fatalf("queued mods = %v, want the switched-off one too", queued)
	}
	if _, err := s.ImportData(context.Background(), "stardew", "not a payload"); err == nil {
		t.Fatal("a bad payload must fail")
	}
}
