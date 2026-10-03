package problems

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestConflictEvidenceEditImageOverlap(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	a := imageConflictPack(t, "Pack.ImageA", 0, 0, 16, 16)
	b := imageConflictPack(t, "Pack.ImageB", 8, 8, 16, 16)
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{a, b})
	if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "edit" || got.AssetConflicts[0].Target != "tilesheets/crops" {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	ev := got.AssetConflicts[0].Evidence
	if len(ev) != 2 {
		t.Fatalf("evidence = %+v", ev)
	}
	byID := map[string]ConflictEvidence{}
	for _, e := range ev {
		byID[e.PackID] = e
		if e.Action != kindEditImage || e.Source != "content.json" || e.Index != 0 || e.Target != "tilesheets/crops" {
			t.Fatalf("evidence %+v", e)
		}
	}
	left, right := byID["Pack.ImageA"], byID["Pack.ImageB"]
	if left.PackName != "Pack.ImageA" || !strings.Contains(left.ToArea, `"X":0`) || left.Priority != "Late" {
		t.Fatalf("left %+v", left)
	}
	if left.CropW != 8 || left.CropH != 8 || left.CropX != 8 || left.CropY != 8 {
		t.Fatalf("left crop %+v", left)
	}
	if right.CropW != 8 || right.CropH != 8 || right.CropX != 0 || right.CropY != 0 {
		t.Fatalf("right crop %+v", right)
	}
}

func TestConflictEvidenceEditData(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	a := dataConflictPack(t, "Pack.DataA", "alpha")
	b := dataConflictPack(t, "Pack.DataB", "beta")
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{a, b})
	if len(got.AssetConflicts) != 1 || got.AssetConflicts[0].Kind != "edit" || got.AssetConflicts[0].Target != "data/objects" {
		t.Fatalf("got %+v", got.AssetConflicts)
	}
	ev := got.AssetConflicts[0].Evidence
	if len(ev) != 2 {
		t.Fatalf("evidence = %+v", ev)
	}
	for _, e := range ev {
		if e.Action != kindEditData || e.Source != "content.json" || e.Index != 0 || e.When == "" || e.Priority != "Late" {
			t.Fatalf("evidence %+v", e)
		}
		if !strings.Contains(e.When, "season=spring") {
			t.Fatalf("when %q", e.When)
		}
	}
}

func TestCropImageClampsToBounds(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 10), G: uint8(y * 10), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()

	url, err := cropImageDataURL(raw, -4, -4, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeDataURLPNG(t, url)
	if got.Bounds().Dx() != 8 || got.Bounds().Dy() != 8 {
		t.Fatalf("clamped size %v", got.Bounds())
	}

	url, err = cropImageDataURL(raw, 6, 6, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	got = decodeDataURLPNG(t, url)
	if got.Bounds().Dx() != 2 || got.Bounds().Dy() != 2 {
		t.Fatalf("edge size %v", got.Bounds())
	}
	r, g, _, a := got.At(got.Bounds().Min.X, got.Bounds().Min.Y).RGBA()
	if r>>8 != 60 || g>>8 != 60 || a>>8 != 255 {
		t.Fatalf("edge pixel r=%d g=%d a=%d", r>>8, g>>8, a>>8)
	}
}

func imageConflictPack(t *testing.T, id string, x, y, w, h int) Installed {
	t.Helper()
	root := t.TempDir()
	writeManifest(t, root, id)
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	img.SetNRGBA(0, 0, color.NRGBA{A: 255, R: id[len(id)-1]})
	writePNG(t, filepath.Join(root, "patch.png"), img)
	content := `{"Changes":[{"Action":"EditImage","Target":"TileSheets/crops","FromFile":"patch.png","Priority":"Late","ToArea":{"X":` +
		strconv.Itoa(x) + `,"Y":` + strconv.Itoa(y) + `,"Width":` + strconv.Itoa(w) + `,"Height":` + strconv.Itoa(h) + `}}]}`
	if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id}
}

func dataConflictPack(t *testing.T, id, value string) Installed {
	t.Helper()
	root := t.TempDir()
	writeManifest(t, root, id)
	content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","Priority":"Late","When":{"Season":"Spring"},"Entries":{"123":"` + value + `"}}]}`
	if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id}
}

func writeManifest(t *testing.T, root, id string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"Name":"` + id + `","UniqueID":"` + id + `","Version":"1","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
	if err := fsx.WriteFile(filepath.Join(root, "manifest.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func decodeDataURLPNG(t *testing.T, url string) image.Image {
	t.Helper()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(url, prefix) {
		t.Fatalf("url %q", url)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(url, prefix))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestTokenDataKeysDoNotClashAcrossPacks(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id, value string) Installed {
		root := t.TempDir()
		writeManifest(t, root, id)
		content := `{"Changes":[{"Action":"EditData","Target":"Data/TriggerActions","Entries":{"{{ModId}}_MigrateIds":"` + value + `"}}]}`
		if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id}
	}
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{pack("Mizu.Quail", "a"), pack("Mizu.Turkey", "b")})
	if len(got.AssetConflicts) != 0 {
		t.Fatalf("{{ModId}} keys are per pack, got %+v", got.AssetConflicts)
	}
}

func TestTargetFieldScopesDataKeys(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id, item string) Installed {
		root := t.TempDir()
		writeManifest(t, root, id)
		content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","TargetField":["` + item + `"],"Entries":{"Price":"` + id + `"}}]}`
		if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id}
	}
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{pack("A.One", "301"), pack("B.Two", "302")})
	if len(got.AssetConflicts) != 0 {
		t.Fatalf("Price on different items is no conflict, got %+v", got.AssetConflicts)
	}
	same := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{pack("C.One", "301"), pack("D.Two", "301")})
	if len(same.AssetConflicts) != 1 {
		t.Fatalf("Price on the same item still clashes, got %+v", same.AssetConflicts)
	}
}

func TestListAppendsDoNotClash(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	resetContentPackCaches()
	t.Cleanup(resetContentPackCaches)

	pack := func(id string) Installed {
		root := t.TempDir()
		writeManifest(t, root, id)
		content := `{"Changes":[{"Action":"EditData","Target":"Data/Objects","TargetField":["16","ContextTags"],"Entries":{"#-1":"` + id + `_tag"}}]}`
		if err := fsx.WriteFile(filepath.Join(root, "content.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return Installed{Key: id, Enabled: true, Folder: root, Name: id, UniqueID: id}
	}
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{pack("A.Tags"), pack("B.Tags")})
	if len(got.AssetConflicts) != 0 {
		t.Fatalf("list appends never clash, got %+v", got.AssetConflicts)
	}
}
