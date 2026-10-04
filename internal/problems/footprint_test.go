package problems

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func change(t *testing.T, raw string) cpChange {
	t.Helper()
	var ch cpChange
	if err := json.Unmarshal([]byte(raw), &ch); err != nil {
		t.Fatal(err)
	}
	return ch
}

func edit(t *testing.T, raw string, image bool) cpPatch {
	t.Helper()
	ch := change(t, raw)
	return cpPatch{kind: "edit", shapes: editShapes(t.TempDir(), ch, image), spouse: spouseOf(ch.When), places: placesOf(ch.When)}
}

func TestEditsClash(t *testing.T) {
	cases := []struct {
		name  string
		a, b  string
		image bool
		want  bool
	}{
		{"disjoint image areas", `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16}}`, `{"ToArea":{"X":16,"Y":0,"Width":16,"Height":16}}`, true, false},
		{"overlapping image areas", `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16}}`, `{"ToArea":{"X":8,"Y":8,"Width":16,"Height":16}}`, true, true},
		{"FromArea sizes a patch at the top-left", `{"FromArea":{"X":64,"Y":64,"Width":16,"Height":16}}`, `{"ToArea":{"X":32,"Y":0,"Width":16,"Height":16}}`, true, false},
		{"tokenized area is the whole asset", `{"ToArea":{"X":"{{x}}","Y":0,"Width":16,"Height":16}}`, `{"ToArea":{"X":200,"Y":200,"Width":1,"Height":1}}`, true, true},
		{"map tiles on different spots", `{"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Back","SetIndex":5}]}`, `{"MapTiles":[{"Position":{"X":2,"Y":1},"Layer":"Back","SetIndex":5}]}`, false, false},
		{"same tile, other layer", `{"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Back","SetIndex":5}]}`, `{"MapTiles":[{"Position":{"X":1,"Y":1},"Layer":"Front","SetIndex":5}]}`, false, false},
		{"tile inside a map patch area", `{"FromFile":"p.tmx","ToArea":{"X":0,"Y":0,"Width":10,"Height":10}}`, `{"MapTiles":[{"Position":{"X":3,"Y":3},"Layer":"Buildings","SetIndex":1}]}`, false, true},
		{"property-only tile edits merge", `{"MapTiles":[{"Position":{"X":"{{Random: 1, 2}}","Y":1},"Layer":"Back","SetProperties":{"Light":"1"}}]}`, `{"FromFile":"p.tmx"}`, false, false},
		{"different map properties", `{"MapProperties":{"Music":"x"}}`, `{"MapProperties":{"Light":"y"}}`, false, false},
		{"same map property, different values", `{"MapProperties":{"Music":"x"}}`, `{"MapProperties":{"music":"y"}}`, false, true},
		{"same map property, same value", `{"MapProperties":{"AllowGiantCrops":"T"}}`, `{"MapProperties":{"allowgiantcrops":"t"}}`, false, false},
		{"different spouses never apply together", `{"When":{"Query: '{{Spouse}}' = 'Sterling'":true}}`, `{"When":{"Relationship:Jasper":"Married"}}`, true, false},
		{"same spouse still clashes", `{"When":{"Relationship:Jasper":"Married"}}`, `{"When":{"Query: '{{Spouse}}' = 'Jasper'":true}}`, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, b := edit(t, c.a, c.image), edit(t, c.b, c.image)
			if got, _ := editsClash([]cpPatch{a}, []cpPatch{b}); got != c.want {
				t.Fatalf("clash = %v, shapes %v / %v", got, a.shapes, b.shapes)
			}
		})
	}
}

func TestAdditiveEditHasNoShape(t *testing.T) {
	if s := editShapes(t.TempDir(), change(t, `{"AddWarps":["1 2 Town 3 4"],"TextOperations":[{"Operation":"Append"}]}`), false); len(s) != 0 {
		t.Fatalf("shapes %v", s)
	}
}

func TestOpaqueImageShapeMatchesBruteForceCells(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 48, 40))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if (x*7+y*11)%13 < 5 {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, A: uint8((x+y)%255 + 1)})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, "patch.png"), buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	from := change(t, `{"FromArea":{"X":3,"Y":5,"Width":37,"Height":29}}`).FromArea
	got, ok := opaqueImageShape(dir, "patch.png", from, 7, 9)
	if !ok {
		t.Fatal("opaque image shape failed")
	}
	wantSet := map[string]bool{}
	for py := 5; py < 34; py++ {
		for px := 3; px < 40; px++ {
			_, _, _, alpha := img.At(px, py).RGBA()
			if alpha > 0 {
				wantSet[strconv.Itoa((7+px-3)/16)+","+strconv.Itoa((9+py-5)/16)] = true
			}
		}
	}
	want := make([]string, 0, len(wantSet))
	for y := range 4 {
		for x := range 3 {
			cell := strconv.Itoa(x) + "," + strconv.Itoa(y)
			if wantSet[cell] {
				want = append(want, cell)
			}
		}
	}
	wantCells := ";" + strings.Join(want, ";")
	if got.cells != wantCells {
		t.Fatalf("cells = %v, want %v", got.cells, wantCells)
	}
	seen := map[string]bool{}
	for cell := range strings.SplitSeq(got.cells, ";") {
		if cell == "" {
			continue
		}
		if seen[cell] {
			t.Fatalf("duplicate cell %q in %v", cell, got.cells)
		}
		seen[cell] = true
	}
	full, ok := opaqueImageShape(dir, "patch.png", nil, 7, 9)
	if !ok || full.cells == got.cells {
		t.Fatalf("full-area cache key did not preserve the requested area: %q / %q", full.cells, got.cells)
	}
	cacheEntries := 0
	cacheValuesAreCells := true
	prefix := filepath.Join(dir, "patch.png") + "\x00"
	pngShapeCache.Range(func(key, value any) bool {
		keyString, ok := key.(string)
		if ok && strings.HasPrefix(keyString, prefix) {
			cacheEntries++
			_, valueIsCells := value.(string)
			cacheValuesAreCells = cacheValuesAreCells && valueIsCells
		}
		return true
	})
	if cacheEntries != 2 || !cacheValuesAreCells {
		t.Fatalf("PNG cache entries = %d, cells-only values = %v", cacheEntries, cacheValuesAreCells)
	}
}

func TestImageAlphaFallbackKeepsLowOpacity(t *testing.T) {
	img := lowOpacityImage{}
	if imageAlpha(img, 0, 0) == 0 {
		t.Fatal("low nonzero alpha was treated as transparent")
	}
}

func TestSkipImageOverlapUsesRectangles(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(0, 0, color.NRGBA{A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, "patch.png"), buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	ch := change(t, `{"FromFile":"patch.png","PatchMode":"Overlay"}`)
	full := imagePatchShapes(dir, ch, 0, 0)
	if len(full) != 1 || full[0].cells == "" {
		t.Fatalf("full scan shapes = %#v", full)
	}
	skipImageOverlap = true
	t.Cleanup(func() { skipImageOverlap = false })
	skipped := imagePatchShapes(dir, ch, 0, 0)
	if len(skipped) != 1 || skipped[0].cells != "" || skipped[0].w != 32 || skipped[0].h != 16 {
		t.Fatalf("skip scan shapes = %#v", skipped)
	}
}

func TestOpaqueImageShapeConcurrent(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	img.SetNRGBA(8, 8, color.NRGBA{A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, "patch.png"), buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan string, 16)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			got, ok := opaqueImageShape(dir, "patch.png", nil, 0, 0)
			if !ok || !strings.Contains(got.cells, "0,0") {
				errCh <- got.cells
			}
		})
	}
	wg.Wait()
	close(errCh)
	for msg := range errCh {
		t.Fatalf("concurrent opaque shape = %q", msg)
	}
}

type lowOpacityImage struct{}

func (lowOpacityImage) ColorModel() color.Model { return color.RGBA64Model }
func (lowOpacityImage) Bounds() image.Rectangle { return image.Rect(0, 0, 1, 1) }
func (lowOpacityImage) At(int, int) color.Color { return color.RGBA64{A: 1} }

func TestPlacesMakeEditsExclusive(t *testing.T) {
	a := edit(t, `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16},"When":{"LocationName":"EastScarp_Village"}}`, true)
	b := edit(t, `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16},"When":{"LocationName |contains=Custom_Umuwi":true}}`, true)
	c := edit(t, `{"ToArea":{"X":0,"Y":0,"Width":16,"Height":16},"When":{"LocationName":"Custom_Umuwi, EastScarp_Village"}}`, true)
	if clash, _ := editsClash([]cpPatch{a}, []cpPatch{b}); clash {
		t.Fatal("different locations clashed")
	}
	if clash, _ := editsClash([]cpPatch{a}, []cpPatch{c}); !clash {
		t.Fatal("shared location did not clash")
	}
}

func TestMapPatchWithoutToAreaUsesTheSourceMapSize(t *testing.T) {
	dir := t.TempDir()
	tmx := `<?xml version="1.0"?><map version="1.0" orientation="orthogonal" width="10" height="8" tilewidth="16"></map>`
	if err := os.WriteFile(filepath.Join(dir, "p.tmx"), []byte(tmx), 0o600); err != nil {
		t.Fatal(err)
	}
	s := editShapes(dir, change(t, `{"FromFile":"p.tmx"}`), false)
	if len(s) != 1 || s[0].kind != 'r' || s[0].w != 10 || s[0].h != 8 {
		t.Fatalf("shapes %v", s)
	}
	if s := editShapes(dir, change(t, `{"FromFile":"p.tbin"}`), false); len(s) != 1 || s[0].kind != 'w' {
		t.Fatalf("tbin should be whole: %v", s)
	}
}

func TestDecodeMapCacheUsesFileStamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "map.tmj")
	write := func(width int) {
		t.Helper()
		raw := `{"width":` + strconv.Itoa(width) + `,"height":1,"layers":[]}`
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(1)
	mapCache = sync.Map{}
	first, ok := decodeMap(dir, "map.tmj")
	if !ok || first.width != 1 {
		t.Fatalf("first decode = %#v, %v", first, ok)
	}
	if _, ok := decodeMap(dir, "map.tmj"); !ok {
		t.Fatal("second decode failed")
	}
	changed := time.Now().Add(time.Second)
	write(2)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatal(err)
	}
	second, ok := decodeMap(dir, "map.tmj")
	if !ok || second.width != 2 {
		t.Fatalf("changed decode = %#v, %v", second, ok)
	}
	entries := 0
	prefix := path + "\x00"
	mapCache.Range(func(key, _ any) bool {
		keyString, ok := key.(string)
		if ok && strings.HasPrefix(keyString, prefix) {
			entries++
		}
		return true
	})
	if entries != 2 {
		t.Fatalf("map cache entries = %d, want 2", entries)
	}
}

func TestHasModWithEmptyInput(t *testing.T) {
	w := parseWhen(map[string]json.RawMessage{"HasMod: |contains=Other.Mod": json.RawMessage(`false`)}, map[string]bool{}, nil)
	if len(w.noneOf) != 1 || w.noneOf[0] != "other.mod" {
		t.Fatalf("when %+v", w)
	}
}

func TestHarmlessOverlaps(t *testing.T) {
	area := `"ToArea":{"X":0,"Y":0,"Width":16,"Height":16}`
	mapEdit := func(when string) cpPatch {
		return edit(t, `{"FromFile":"p.tmx",`+area+when+`}`, false)
	}
	img := func(when string) cpPatch {
		p := edit(t, `{`+area+when+`}`, true)
		p.image = true
		return p
	}
	tiny := edit(t, `{"FromFile":"p.tmx","ToArea":{"X":"{{x}}","Y":"{{y}}","Width":1,"Height":1}}`, false)
	cases := []struct {
		name  string
		a, b  cpPatch
		minor bool
	}{
		{"two image edits are cosmetic", img(""), img(""), true},
		{"two map edits matter", mapEdit(""), mapEdit(""), false},
		{"a map edit in one location only", mapEdit(`,"When":{"LocationName":"Cave"}`), mapEdit(""), true},
		{"an image edit in one weather only", img(`,"When":{"Weather":"Storm"}`), mapEdit(""), true},
		{"one tile at a computed spot", tiny, mapEdit(""), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clash, minor := editsClash([]cpPatch{c.a}, []cpPatch{c.b})
			if !clash || minor != c.minor {
				t.Fatalf("clash=%v minor=%v", clash, minor)
			}
		})
	}
}
