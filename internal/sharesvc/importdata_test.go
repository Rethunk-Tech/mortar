package sharesvc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

func TestImportDataPlacesAPairedComputersLocalArchiveFromTheStore(t *testing.T) {
	s, rec := newService(t, true)
	items, _ := testenv.Stores(t)
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"Name":"Mine","Author":"a","Version":"1.0.0","UniqueID":"Me.Mine","EntryDll":"m.dll"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := store.HashDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir(t.Context(), "stardew", key, src); err != nil {
		t.Fatal(err)
	}
	s.d.Stored = func(game, k string) bool { _, err := items.Path(game, k); return err == nil }
	missing := "local-" + strings.Repeat("0", 64)
	p := profile.Profile{Name: "Desk", Entries: []profile.Entry{
		{Key: key, Source: profile.Source{Kind: profile.KindLocal, Name: "Mine.zip"}, Mods: []profile.Component{{ID: mod.SMAPI("Me.Mine"), Folder: "."}}},
		{Key: missing, Source: profile.Source{Kind: profile.KindLocal, Name: "Gone.zip"}},
	}}
	inc := share.OwnInclude()
	inc.LocalFiles = true
	var payload bytes.Buffer
	if _, err := share.Write(&payload, "stardew", p, t.TempDir(), inc); err != nil {
		t.Fatal(err)
	}
	res, err := s.ImportData(context.Background(), "stardew", base64.RawStdEncoding.EncodeToString(payload.Bytes()))
	if err != nil || len(rec.reqs) != 0 {
		t.Fatalf("import = %+v, %v; queued %+v", res, err, rec.reqs)
	}
	if len(res.Profile.Entries) == 0 || !slices.ContainsFunc(res.Profile.Entries, func(e profile.Entry) bool {
		return e.Key == key && e.Source.Kind == profile.KindLocal && e.Source.Name == "Mine.zip"
	}) {
		t.Fatalf("entries = %+v", res.Profile.Entries)
	}
	if !strings.Contains(res.Profile.Notes, "Gone.zip") {
		t.Fatalf("the archive that did not arrive is not noted: %q", res.Profile.Notes)
	}
}

func TestASharedFileSaysWhyItCannotBePreviewed(t *testing.T) {
	s, _ := newService(t, true)
	var other bytes.Buffer
	if _, err := share.Write(&other, "lethal-company", profile.Profile{Name: "Crew"}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(other.Bytes())
	if _, err := s.PreviewData(context.Background(), "stardew", encoded, ""); usererr.KindOf(err) != usererr.OtherGame {
		t.Fatalf("another game's file: %v", err)
	}
	if _, err := s.PreviewData(context.Background(), "stardew", "not a payload", ""); usererr.KindOf(err) != usererr.Damaged {
		t.Fatalf("a damaged payload: %v", err)
	}
	var newer bytes.Buffer
	zw := zip.NewWriter(&newer)
	w, err := zw.Create("profile.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(w, `{"version":%d,"name":"Later","game":"stardew"}`, share.FormatVersion+1); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	encoded = base64.RawStdEncoding.EncodeToString(newer.Bytes())
	if _, err := s.PreviewData(context.Background(), "stardew", encoded, ""); usererr.KindOf(err) != usererr.Outdated {
		t.Fatalf("a newer Mortar's file: %v", err)
	}
}
