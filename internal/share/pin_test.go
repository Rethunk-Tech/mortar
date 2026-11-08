package share

import (
	"archive/zip"
	"bytes"
	"io"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func pinPayload(t *testing.T, files map[string]string) []byte {
	t.Helper()
	e := profile.Entry{Key: "k", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2}, Mods: []profile.Component{{ID: "smapi:A.One", Folder: "A"}}}
	var buf bytes.Buffer
	if _, err := Write(&buf, "stardew", profile.Profile{Name: "Main", Entries: []profile.Entry{e}}, "", OwnInclude()); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		if err := putFile(zw, f.Name, data); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range files {
		if err := putFile(zw, name, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func zipFiles(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		out[f.Name] = string(data)
	}
	return out
}

// A pinned mod is the same mod under a case variant of its SMAPI id, so src's off state crosses to dst.
func TestPinDisabledMatchesSmapiIDsByCase(t *testing.T) {
	dst := []Ref{{ModID: 1, FileID: 2}}
	src := []Ref{{ModID: 1, FileID: 2, Disabled: []mod.ID{"smapi:A.Off"}}}
	got := pinDisabled(dst, src, []mod.ID{"smapi:A.OFF"})
	if len(got) != 1 || !slices.Equal(got[0].Disabled, []mod.ID{"smapi:A.OFF"}) {
		t.Fatalf("pinDisabled = %+v", got)
	}
}

func TestPinTakesNamedFilesFromSrcAndDropsWhatSrcLacks(t *testing.T) {
	dst := pinPayload(t, map[string]string{"loader/BepInEx/config/a.cfg": "synced", "loader/BepInEx/config/b.cfg": "keep"})
	src := pinPayload(t, map[string]string{"loader/BepInEx/config/a.cfg": "mine", "configs/smapi/A.One/config.json": "{}"})
	got, err := Pin(dst, src, nil, []string{"a.cfg", "A.One/config.json", "b.cfg"})
	if err != nil {
		t.Fatal(err)
	}
	files := zipFiles(t, got)
	if files["loader/BepInEx/config/a.cfg"] != "mine" || files["configs/smapi/A.One/config.json"] != "{}" {
		t.Fatalf("files = %v", files)
	}
	if _, ok := files["loader/BepInEx/config/b.cfg"]; ok {
		t.Fatal("a pinned file src lacks must go")
	}
}
