// Package modfixture generates a Stardew Valley profile shaped like a large real one (about 800 mods, a third of them
// Content Patcher packs), for tests that hold the checks to a time budget. The same seed always writes the same files.
package modfixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// Shape is the profile the fixture copies. The defaults are measured on a real 811-mod Stardew profile: 671 entries,
// 371 enabled Content Patcher packs, about 48,000 files of which 63% are PNGs, and about 45,000 patches over 14,000
// targets whose popularity falls off like a Zipf curve.
type Shape struct {
	Entries int
	// PackShare is the share of mods that are Content Patcher packs.
	PackShare float64
	// FilesMedian and FilesSpread draw each mod's file count from a log-normal: median and sigma of its log.
	FilesMedian, FilesSpread float64
	// PatchesMedian and PatchesSpread draw each pack's patch count the same way.
	PatchesMedian, PatchesSpread float64
	// Targets is how many shared assets the patches' Zipf curve spreads over.
	Targets int
	// EntryBytes is about how much text one EditData entry carries.
	EntryBytes int
}

// Real is the shape of the measured profile.
var Real = Shape{
	Entries: 671, PackShare: 0.456,
	FilesMedian: 15, FilesSpread: 1.67,
	PatchesMedian: 28, PatchesSpread: 1.82,
	Targets: 4000, EntryBytes: 150,
}

const game = "stardew"

type gen struct {
	t      testing.TB
	noise  *rand.ChaCha8
	shape  Shape
	target zipf
	art    zipf
	pngs   [][]byte
	// ids is every enabled mod written so far, which later mods may depend on.
	ids     []string
	entries []profile.Entry
}

// Profile writes the fixture's mods into a new profile of the store and returns the profile's id. It lays the files
// out and records the entries directly: installing 671 entries through the store copies and syncs every file twice,
// which takes minutes.
func Profile(t testing.TB, profiles *profile.Store, shape Shape) string {
	t.Helper()
	p, err := profiles.Create(game, "Fixture")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := profiles.ProfileDir(game, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	// A fixed seed, so every run writes the same fixture.
	noise := rand.NewChaCha8([32]byte{})
	g := &gen{t: t, noise: noise, shape: shape, target: newZipf(1.1, 2, shape.Targets), art: newZipf(1, 5, 800)}
	for _, size := range [][2]int{{16, 16}, {32, 32}, {64, 64}, {96, 32}, {128, 128}} {
		g.pngs = append(g.pngs, g.png(size[0], size[1]))
	}
	mods := filepath.Join(dir, "mods")
	g.contentPatcher(mods)
	// The real profile's entries hold one mod 580 times in 671, the rest up to eight, like a framework with its packs.
	perEntry := []int{580, 65, 14, 6, 2, 1, 2, 1}
	for e := 1; e < shape.Entries; e++ {
		n, x := 1, g.intN(671)
		for n < len(perEntry) && x >= perEntry[n-1] {
			x -= perEntry[n-1]
			n++
		}
		// About 3% of the entries are switched off.
		g.entry(mods, e, n, e%37 == 0)
	}
	// Mortar does not trust a file stamped in the last few seconds to stay as it is, so files as new as these keep
	// the caches from settling for several checks; a real profile's files are months old.
	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	root, err := os.OpenRoot(mods)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := fs.WalkDir(root.FS(), ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return root.Chtimes(path, old, old)
	}); err != nil {
		t.Fatal(err)
	}
	p.Entries = g.entries
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "profile.json", string(raw))
	if err := profiles.RecordModsSnapshot(game, p.ID); err != nil {
		t.Fatal(err)
	}
	return p.ID
}

// float, intN and norm draw from the seeded stream: uniform in [0, 1), uniform in [0, n), and standard normal.
func (g *gen) float() float64 { return float64(g.noise.Uint64()>>11) / (1 << 53) }

func (g *gen) intN(n int) int { return int(g.float() * float64(n)) }

func (g *gen) norm() float64 {
	return math.Sqrt(-2*math.Log(1-g.float())) * math.Cos(2*math.Pi*g.float())
}

func (g *gen) logNormal(median, sigma float64, limit int) int {
	return max(1, min(limit, int(math.Round(median*math.Exp(g.norm()*sigma)))))
}

func (g *gen) png(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	_, _ = g.noise.Read(img.Pix)
	// A quarter of the pixels are transparent, the rest opaque, so overlays have a shape to compare.
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] < 64 {
			img.Pix[i] = 0
		} else {
			img.Pix[i] = 255
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		g.t.Fatal(err)
	}
	return b.Bytes()
}

func (g *gen) write(path string, body []byte) {
	g.t.Helper()
	testfs.WriteFile(g.t, filepath.Dir(path), filepath.Base(path), string(body))
}

func (g *gen) json(path string, v any) {
	g.t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		g.t.Fatal(err)
	}
	g.write(path, b)
}

const cpID = "Pathoschild.ContentPatcher"

// filesPerDir spreads a mod's files over folders of five, as real mods average 4.4 files a folder.
const filesPerDir = 5

func (g *gen) contentPatcher(mods string) {
	key := "nexus-1915-100000"
	folder := filepath.Join(mods, key, "ContentPatcher")
	g.json(filepath.Join(folder, "manifest.json"), map[string]any{
		"Name": "Content Patcher", "Author": "Pathoschild", "Version": "2.7.0", "UniqueID": cpID, "EntryDll": "ContentPatcher.dll",
		"UpdateKeys": []string{"Nexus:1915"},
	})
	g.write(filepath.Join(folder, "ContentPatcher.dll"), bytes.Repeat([]byte{0x4d}, 4096))
	g.ids = append(g.ids, cpID)
	g.entries = append(g.entries, profile.Entry{
		Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: 1915, FileID: 100000, Version: "2.7.0", Name: "Content Patcher.zip"},
		Mods: []profile.Component{{ID: mod.SMAPI(cpID), Version: "2.7.0", Name: "Content Patcher", Author: "Pathoschild", Folder: "ContentPatcher"}},
	})
}

// entry writes one Nexus download holding n mods, each in its own folder, and records it.
func (g *gen) entry(mods string, e, n int, off bool) {
	key := fmt.Sprintf("nexus-%d-%d", 10000+e, 200000+e)
	en := profile.Entry{Key: key, Source: profile.Source{Kind: profile.KindNexus, ModID: 10000 + e, FileID: 200000 + e, Version: "1.0.0", Name: fmt.Sprintf("Fixture %d.zip", e)}}
	var ids []string
	for m := range n {
		name := fmt.Sprintf("Fixture Mod %d-%d", e, m)
		folder := name
		if off {
			folder = "." + name
		}
		id := g.mod(filepath.Join(mods, key, folder), name)
		en.Mods = append(en.Mods, profile.Component{ID: mod.SMAPI(id), Version: "1.0.0", Name: name, Author: "Fixture", Folder: name})
		if off {
			en.Disabled = append(en.Disabled, mod.SMAPI(id))
		} else {
			ids = append(ids, id)
		}
	}
	g.ids = append(g.ids, ids...)
	g.entries = append(g.entries, en)
}

func (g *gen) mod(folder, name string) string {
	id := "Fixture." + strings.ReplaceAll(name, " ", "")
	pack := g.float() < g.shape.PackShare
	files := g.logNormal(g.shape.FilesMedian, g.shape.FilesSpread, 2400)
	man := map[string]any{
		"Name": name, "Author": "Fixture", "Version": "1.0.0",
		"Description": "A generated mod for the performance budgets.", "UniqueID": id,
		"UpdateKeys": []string{"Nexus:" + strconv.Itoa(10000+len(g.entries))}, "MinimumApiVersion": "4.0.0",
		"Dependencies": g.deps(),
	}
	if pack {
		man["ContentPackFor"] = map[string]string{"UniqueID": cpID}
		files -= g.pack(folder, id, files)
	} else {
		man["EntryDll"] = "Mod.dll"
		g.write(filepath.Join(folder, "Mod.dll"), bytes.Repeat([]byte{0x4d}, 2048+len(g.ids)))
		g.json(filepath.Join(folder, "i18n", "default.json"), map[string]string{"name": name, "desc": "Generated"})
		files -= 2
	}
	g.json(filepath.Join(folder, "manifest.json"), man)
	// The rest are the assets a mod ships: mostly PNGs, then data and text.
	for i := range files - 1 {
		if i%8 < 5 {
			g.write(filepath.Join(folder, "assets", strconv.Itoa(i/filesPerDir), fmt.Sprintf("sprite%d.png", i)), g.pngs[i%len(g.pngs)])
		} else {
			g.write(filepath.Join(folder, "assets", strconv.Itoa(i/filesPerDir), fmt.Sprintf("data%d.json", i)), []byte(`{"value":`+strconv.Itoa(i)+`}`))
		}
	}
	return id
}

// deps draws the real profile's fan-out: half the mods need nothing, a quarter one mod, and a few need dozens. Every
// dependency is an enabled mod written earlier, so the requirements check finds them all installed.
func (g *gen) deps() []map[string]any {
	var n int
	switch x := g.float(); {
	case x < 0.52:
		return nil
	case x < 0.77:
		n = 1
	case x < 0.84:
		n = 2
	case x < 0.90:
		n = 3 + g.intN(2)
	case x < 0.97:
		n = 5 + g.intN(5)
	case x < 0.995:
		n = 10 + g.intN(13)
	default:
		n = 60
	}
	out := make([]map[string]any, 0, n)
	for range min(n, len(g.ids)) {
		out = append(out, map[string]any{"UniqueID": g.ids[g.intN(len(g.ids))], "MinimumVersion": "1.0.0"})
	}
	return out
}

var dataKinds = []string{"Data/Objects", "Data/Machines", "Data/NPCGiftTastes", "Data/Shops", "Data/Locations", "Data/Characters", "Data/TriggerActions", "Data/Mail"}

// targetName draws an asset. A tenth of the data patches and half the image and map patches edit an asset only their
// pack touches; the rest follow a Zipf curve, steep for data, whose top ranks every pack edits, and flat for images and
// maps, where a popular sheet has a handful of editors.
func (g *gen) targetName(prefix, id string, p int) string {
	own, shared := 10, g.target
	if prefix != "Data" {
		own, shared = 2, g.art
	}
	if g.intN(own) == 0 {
		return prefix + "/" + id + "_" + strconv.Itoa(p%5)
	}
	rank := shared.rank(g.float())
	if prefix == "Data" && rank < len(dataKinds) {
		return dataKinds[rank]
	}
	return prefix + "/Fixture" + strconv.Itoa(rank)
}

// pack writes content.json, its includes and the files its patches load, and returns how many files it wrote.
func (g *gen) pack(folder, id string, budget int) int {
	patches := g.logNormal(g.shape.PatchesMedian, g.shape.PatchesSpread, 4600)
	written := 1
	var images, maps []string
	for i := range max(1, budget*5/8) {
		name := fmt.Sprintf("images/%d/img%d.png", i/filesPerDir, i)
		g.write(filepath.Join(folder, filepath.FromSlash(name)), g.pngs[g.intN(len(g.pngs))])
		images = append(images, name)
	}
	written += len(images)
	for i := range max(1, budget/16) {
		name := fmt.Sprintf("maps/map%d.tmx", i)
		g.write(filepath.Join(folder, filepath.FromSlash(name)), tmx(16+i%16, 16))
		maps = append(maps, name)
	}
	written += len(maps)
	changes := make([]map[string]any, 0, patches)
	for p := range patches {
		changes = append(changes, g.patch(id, p, images, maps))
	}
	// Most packs keep a few patches in content.json and include the rest from files of ten; one in twenty of those
	// under 800 patches keeps up to 400 there, like the real profile's biggest content.json (618 patches, 900 KB).
	keep := min(len(changes), 10)
	if g.intN(20) == 0 && len(changes) < 800 {
		keep = min(len(changes), 250)
	}
	const perFile = 10
	if keep < len(changes) {
		var includes []string
		for i := keep; i < len(changes); i += perFile {
			name := fmt.Sprintf("data/part%d.json", len(includes))
			g.json(filepath.Join(folder, filepath.FromSlash(name)), map[string]any{"Changes": changes[i:min(len(changes), i+perFile)]})
			includes = append(includes, name)
		}
		changes = append(changes[:keep:keep], map[string]any{"Action": "Include", "FromFile": strings.Join(includes, ", ")})
		written += len(includes)
	}
	content := map[string]any{"Format": "2.7.0", "Changes": changes}
	if g.intN(3) == 0 {
		content["ConfigSchema"] = map[string]any{
			"Enabled": map[string]any{"AllowValues": "true, false", "Default": "true"},
			"Style":   map[string]any{"AllowValues": "A, B, C", "Default": "A"},
		}
	}
	g.json(filepath.Join(folder, "content.json"), content)
	return written
}

func (g *gen) patch(id string, p int, images, maps []string) map[string]any {
	var c map[string]any
	switch x := g.float(); {
	case x < 0.62:
		entries := map[string]any{}
		for e := range 1 + g.intN(6) {
			key := id + "_" + strconv.Itoa(p) + "_" + strconv.Itoa(e)
			if g.intN(50) == 0 {
				key = "Shared_" + strconv.Itoa(g.intN(200))
			}
			entries[key] = strings.Repeat("generated text ", max(1, g.shape.EntryBytes/15))
		}
		c = map[string]any{"Action": "EditData", "Target": g.targetName("Data", id, p), "Entries": entries}
	case x < 0.77:
		if g.intN(4) == 0 {
			c = map[string]any{"Action": "Load", "Target": "Maps/" + id + "_" + strconv.Itoa(p), "FromFile": maps[g.intN(len(maps))]}
		} else {
			c = map[string]any{"Action": "Load", "Target": "Characters/" + id + "_" + strconv.Itoa(p), "FromFile": images[g.intN(len(images))]}
		}
	case x < 0.89:
		c = map[string]any{
			"Action": "EditImage", "Target": g.targetName("TileSheets", id, p), "FromFile": images[g.intN(len(images))], "PatchMode": "Overlay",
			"ToArea": map[string]int{"X": 16 * g.intN(48), "Y": 16 * g.intN(48), "Width": 16, "Height": 16},
		}
	default:
		c = map[string]any{
			"Action": "EditMap", "Target": g.targetName("Maps", id, p), "FromFile": maps[g.intN(len(maps))],
			"ToArea": map[string]int{"X": g.intN(200), "Y": g.intN(200), "Width": 8, "Height": 8},
		}
	}
	if g.intN(4) == 0 {
		c["When"] = map[string]string{"Season": []string{"spring", "summer", "fall", "winter"}[g.intN(4)]}
	}
	return c
}

func tmx(w, h int) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>
<map version="1.0" orientation="orthogonal" width="%d" height="%d" tilewidth="16" tileheight="16">
 <tileset firstgid="1" name="outdoors" tilewidth="16" tileheight="16"><image source="spring_outdoorsTileSheet" width="400" height="1264"/></tileset>
 <layer name="Back" width="%d" height="%d"><data encoding="csv">`, w, h, w, h)
	for i := range w * h {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(1 + i%50))
	}
	b.WriteString("</data></layer>\n</map>\n")
	return []byte(b.String())
}

// zipf draws ranks 0 to n-1 with weights (v+rank)^-s, by binary search over the cumulative weights.
type zipf []float64

func newZipf(s, v float64, n int) zipf {
	z := make(zipf, n)
	sum := 0.0
	for k := range n {
		sum += math.Pow(v+float64(k), -s)
		z[k] = sum
	}
	return z
}

// rank maps u, uniform in [0, 1), to a rank.
func (z zipf) rank(u float64) int {
	return sort.SearchFloat64s(z, u*z[len(z)-1])
}
