package share

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/andybalholm/brotli"

	"github.com/Rethunk-AI/mortar/internal/profile"
)

func nexus(key string, mod, file int, disabled ...string) profile.Entry {
	return profile.Entry{
		Key:      key,
		Source:   profile.Source{Kind: profile.KindNexus, ModID: mod, FileID: file},
		Mods:     []profile.EntryMod{{UniqueID: "A." + key, Folder: "."}},
		Disabled: disabled,
	}
}

func sample() profile.Profile {
	gh := profile.Entry{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1.2.3", Asset: "mod-1.2.3.zip"}}
	return profile.Profile{Name: "Farm 🌾", Notes: "n", Description: "co-op Fridays", Entries: []profile.Entry{
		{Key: "smapi-4.1", Source: profile.Source{Kind: profile.SourceSMAPI}},
		{Key: "bridge-1", Source: profile.Source{Kind: profile.SourceMortar}},
		nexus("one", 541, 1000),
		nexus("off", 7, 8, "A.off"),
		{Key: "loc.zip", Source: profile.Source{Kind: profile.KindLocal, Name: "loc.zip"}},
		gh,
		nexus("two", 2, 3),
	}}
}

func TestRoundTripAndLinks(t *testing.T) {
	res, err := Encode(sample())
	if err != nil {
		t.Fatal(err)
	}
	want := []Ref{{ModID: 541, FileID: 1000}, {GitHub: "owner/repo@v1.2.3/mod-1.2.3.zip"}, {ModID: 2, FileID: 3}}
	if len(res.LeftOut) != 1 || res.LeftOut[0] != (LeftOut{Key: "loc.zip", Reason: "local archive"}) {
		t.Fatalf("left out = %+v", res.LeftOut)
	}
	for _, in := range []string{res.Web, res.App, res.App + "/", "  " + res.Payload + "\n"} {
		got, err := Parse(in)
		if err != nil || got.Name != "Farm 🌾" || fmt.Sprint(got.Entries) != fmt.Sprint(want) {
			t.Fatalf("Parse(%.40q) = %+v, %v", in, got, err)
		}
	}
}

func TestDetailsRoundTrip(t *testing.T) {
	choices := map[string]map[string][]string{"Options": {"Pack": {"Optional"}}}
	p := profile.Profile{Name: "Choices", Entries: []profile.Entry{{
		Key: "n", Source: profile.Source{Kind: profile.KindNexus, ModID: 4, FileID: 5},
		Mods: []profile.EntryMod{{UniqueID: "A.On", Folder: "."}, {UniqueID: "A.Off", Folder: "."}}, Disabled: []string{"A.Off"}, Fomod: choices,
	}}}
	res, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(res.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || !slices.Equal(got.Entries[0].Disabled, []string{"A.Off"}) ||
		fmt.Sprint(got.Entries[0].Fomod) != fmt.Sprint(choices) {
		t.Fatalf("link details = %+v", got.Entries)
	}

	dir := modsDirWith(t, map[string]string{"n/config.json": "{}"})
	var buf bytes.Buffer
	if _, err := Write(&buf, p, dir); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "details.mortar")
	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	pv, err := Read(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Entries) != 1 || !slices.Equal(pv.Entries[0].Disabled, []string{"A.Off"}) ||
		fmt.Sprint(pv.Entries[0].Fomod) != fmt.Sprint(choices) {
		t.Fatalf("file details = %+v", pv.Entries)
	}
}

func TestOldLinksRemainWithoutDetails(t *testing.T) {
	got, err := Parse(pack(t, `[1,"old",[[4,5],"owner/repo@v1/a.zip"]]`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "old" || len(got.Entries) != 2 || got.Entries[0].Disabled != nil || got.Entries[0].Fomod != nil ||
		got.Entries[1].Disabled != nil || got.Entries[1].Fomod != nil {
		t.Fatalf("old link = %+v", got)
	}
}

func TestLinkDropsDetailsBeforeRefs(t *testing.T) {
	p := profile.Profile{Name: "large"}
	for i := range 250 {
		sum := sha256.Sum256(fmt.Appendf(nil, "choice-%d", i))
		p.Entries = append(p.Entries, profile.Entry{
			Key: fmt.Sprint(i), Source: profile.Source{Kind: profile.KindNexus, ModID: i + 1, FileID: i + 2},
			Mods:  []profile.EntryMod{{UniqueID: fmt.Sprintf("A.%d", i)}},
			Fomod: map[string]map[string][]string{"Step": {fmt.Sprintf("Group-%d", i): {fmt.Sprintf("%x", sum)}}},
		})
	}
	res, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Payload) >= MaxEncoded {
		t.Fatalf("fallback payload is %d characters", len(res.Payload))
	}
	for _, ref := range res.Shared.Entries {
		if len(ref.Disabled) != 0 || len(ref.Fomod) != 0 || ref.Note != "" || len(ref.Tags) != 0 {
			t.Fatalf("details survived fallback: %+v", ref)
		}
	}
}

func TestWrongForms(t *testing.T) {
	p, err := Encode(sample())
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{
		"", "https://evil.example/stardew/p#" + p.Payload, "https://mortar.rethunk.tech/lethal/p#" + p.Payload,
		"https://mortar.rethunk.tech/stardew/x#" + p.Payload, "http://mortar.rethunk.tech/stardew/p#" + p.Payload,
		"mortar://lethal/p/" + p.Payload, "mortar://stardew/q/" + p.Payload, "javascript:alert(1)",
		"https://mortar.rethunk.tech/stardew/p#", p.Payload + "=", "not base64 !!",
	} {
		if _, err := Parse(in); !errors.Is(err, ErrNotLink) {
			t.Errorf("Parse(%.50q) err = %v, want ErrNotLink", in, err)
		}
	}
}

func pack(t *testing.T, doc string) string {
	t.Helper()
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	if _, err := w.Write([]byte(doc)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes())
}

func TestNewerVersionRefused(t *testing.T) {
	for _, doc := range []string{`[3,"x",[[1,2]]]`, `[9,{"weird":true}]`} {
		if _, err := Parse(pack(t, doc)); !errors.Is(err, ErrNewerVersion) {
			t.Errorf("%s: err = %v", doc, err)
		}
	}
}

func TestHostileShapes(t *testing.T) {
	long := strings.Repeat("a", 61)
	many := "[" + strings.Repeat("[1,2],", MaxEntries) + "[1,2]]"
	for _, doc := range []string{
		``, `{}`, `[]`, `[1]`, `[1,"x"]`, `[1,"x",[],4]`, `[0,"x",[]]`, `[-1,"x",[]]`, `["1","x",[]]`, `[1.5,"x",[]]`,
		`[1,5,[]]`, `[1,"",[]]`, `[1," x",[]]`, `[1,"` + long + `",[]]`, "[1,\"a\\u0000b\",[]]",
		`[1,"x",{}]`, `[1,"x",[[1]]]`, `[1,"x",[[1,2,3]]]`, `[1,"x",[[0,2]]]`, `[1,"x",[[1,-2]]]`, `[1,"x",[[1,2147483648]]]`,
		`[1,"x",[["1","2"]]]`, `[1,"x",[[1.5,2]]]`, `[1,"x",[null]]`, `[1,"x",[7]]`, `[1,"x",[""]]`,
		`[1,"x",["../etc/passwd"]]`, `[1,"x",["o/r@t/a b"]]`, `[1,"x",["o/r@t/../a"]]`, `[1,"x",["o/r@t/a/b"]]`,
		`[1,"x",["o/r@/a"]]`, `[1,"x",["o/..@t/a"]]`, `[1,"x",["o/.@t/a"]]`, `[1,"x",["o/r@..` + `/a"]]`, `[1,"x",["o/r@t/.."]]`, `[1,"x",["/r@t/a"]]`, `[1,"x",[[1,2]]] trailing`, `[1,"x",` + many + `]`,
	} {
		_, err := Parse(pack(t, doc))
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%.60q: err = %v, want ErrMalformed", doc, err)
		}
	}
}

func TestCaps(t *testing.T) {
	if _, err := Parse(strings.Repeat("A", MaxEncoded+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized payload: %v", err)
	}
	if _, err := Parse("mortar://stardew/p/" + strings.Repeat("A", MaxEncoded+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized link: %v", err)
	}
	// A bomb: tiny once compressed, enormous inflated. The decompressed cap must trip without inflating it all.
	bomb := pack(t, `[1,"x",[`+strings.Repeat(" ", 4<<20)+`]]`)
	if len(bomb) > MaxEncoded {
		t.Fatalf("bomb is %d chars, want it under the encoded cap", len(bomb))
	}
	if _, err := Parse(bomb); !errors.Is(err, ErrTooLarge) {
		t.Errorf("bomb: %v", err)
	}
	// Exactly at the decompressed cap parses on to the shape check instead of tripping it.
	edge := `[1,"x",[` + strings.Repeat(" ", MaxDecoded-len(`[1,"x",[]]`)) + `]]`
	if len(edge) != MaxDecoded {
		t.Fatalf("edge is %d bytes", len(edge))
	}
	if got, err := Parse(pack(t, edge)); err != nil || len(got.Entries) != 0 {
		t.Errorf("edge: %+v, %v", got, err)
	}
	if _, err := Parse("!!!!"); err == nil {
		t.Error("garbage accepted")
	}
	if _, err := Parse(base64.RawURLEncoding.EncodeToString([]byte("not brotli at all, just text"))); err == nil {
		t.Error("non-brotli accepted")
	}
}

func TestEncodeRefusesTooLarge(t *testing.T) {
	p := profile.Profile{Name: "big"}
	for i := range 1000 {
		p.Entries = append(p.Entries, nexus(fmt.Sprint(i), pseudo(1, i, 0, 2_000_000_000), pseudo(1, i, 1, 2_000_000_000)))
	}
	if _, err := Encode(p); !errors.Is(err, ErrTooLarge) {
		t.Errorf("err = %v", err)
	}
	if _, err := Encode(profile.Profile{Name: strings.Repeat("a", 61)}); !errors.Is(err, ErrMalformed) {
		t.Errorf("long name: %v", err)
	}
}

// design.md § Sharing measured 498, 879 and 1,630 characters for 50, 100 and 200 real mods. Deterministic ids
// in the same ranges must stay in that neighbourhood, so a codec regression shows up here.
// pseudo returns a deterministic id in 1..limit, spread like real ones.
func pseudo(seed, i, field, limit int) int {
	sum := sha256.Sum256(fmt.Appendf(nil, "%d/%d/%d", seed, i, field))
	return int(binary.BigEndian.Uint32(sum[:4]))%limit + 1
}

func TestLinkSizes(t *testing.T) {
	for _, tc := range []struct{ mods, min, max int }{{50, 450, 550}, {100, 790, 970}, {200, 1470, 1800}} {
		p := profile.Profile{Name: "Sample profile"}
		for i := range tc.mods {
			p.Entries = append(p.Entries, nexus(fmt.Sprint(i), pseudo(tc.mods, i, 0, 42000), pseudo(tc.mods, i, 1, 180000)))
		}
		res, err := Encode(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%d mods: %d characters", tc.mods, len(res.Web))
		if n := len(res.Web); n < tc.min || n > tc.max {
			t.Errorf("%d mods: link is %d characters, want %d to %d", tc.mods, n, tc.min, tc.max)
		}
	}
}

func TestBundledNeverListed(t *testing.T) {
	p := profile.Profile{Name: "x", Entries: []profile.Entry{
		{Key: "smapi", Source: profile.Source{Kind: profile.SourceSMAPI}},
		{Key: "bridge", Source: profile.Source{Kind: profile.SourceMortar}},
		nexus("gone", 1, 2, "A.gone"),
	}}
	res, err := Encode(p)
	if err != nil || len(res.LeftOut) != 0 || len(res.Shared.Entries) != 0 {
		t.Fatalf("%+v, %v", res, err)
	}
}

func TestLeftOutReasons(t *testing.T) {
	p := profile.Profile{Name: "x", Entries: []profile.Entry{
		{Key: "a", Source: profile.Source{Kind: profile.KindNexus}},
		{Key: "b", Source: profile.Source{Kind: profile.KindGitHub, Repo: "bad name"}},
		{Key: "c", Source: profile.Source{Kind: "weird"}},
	}}
	res, err := Encode(p)
	if err != nil || len(res.LeftOut) != 3 {
		t.Fatalf("%+v, %v", res.LeftOut, err)
	}
}

func modsDirWith(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestMortarFileRoundTrip(t *testing.T) {
	p := sample()
	p.Entries[2].Mods[0].Folder = "inner"
	dir := modsDirWith(t, map[string]string{
		"one/inner/config.json":    `{"a":1}`,
		"one/inner/data/deep.JSON": `{"b":2}`,
		"one/inner/manifest.json":  `{"UniqueID":"A.one"}`,
		"one/inner/readme.txt":     "x",
		"one/inner/bad:name.json":  "{}",
		"two/config.json":          `{"c":3}`,
		"off/config.json":          `{"never":1}`,
		"smapi-4.1/config.json":    `{"never":1}`,
		"loc.zip/config.json":      `{"local":1}`,
		"gh/config.json":           `{"g":1}`,
		"one/inner/huge.json":      strings.Repeat(" ", MaxConfigBytes+1),
	})
	p.Entries[5].Mods = []profile.EntryMod{{UniqueID: "A.gh", Folder: "."}}
	var buf bytes.Buffer
	skipped, err := Write(&buf, p, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 2 {
		t.Errorf("skipped = %v", skipped)
	}
	file := filepath.Join(t.TempDir(), "x.mortar")
	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	pv, err := Read(file)
	if err != nil {
		t.Fatal(err)
	}
	if pv.Name != "Farm 🌾" || pv.Notes != "n" || pv.Description != "co-op Fridays" || len(pv.Entries) != 3 {
		t.Errorf("preview = %+v", pv)
	}
	got := map[string]string{}
	for _, c := range pv.Configs {
		got[c.UniqueID+"/"+c.Path] = string(c.Data)
	}
	want := map[string]string{"A.one/config.json": `{"a":1}`, "A.one/data/deep.JSON": `{"b":2}`, "A.two/config.json": `{"c":3}`, "A.gh/config.json": `{"g":1}`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("configs = %v, want %v", got, want)
	}
}

func TestEntryNotesRoundTrip(t *testing.T) {
	p := profile.Profile{Name: "Notes", Entries: []profile.Entry{
		nexus("one", 541, 1000),
		{
			Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1", Asset: "a.zip"},
			Mods: []profile.EntryMod{{UniqueID: "G", Folder: "."}}, Note: "gh note", Tags: []string{"git"},
		},
	}}
	p.Entries[0].Note = "farm tweak\n\tsecond line"
	p.Entries[0].Tags = []string{"QoL", "UI"}
	res, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(res.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 || got.Entries[0].Note != p.Entries[0].Note || !slices.Equal(got.Entries[0].Tags, []string{"QoL", "UI"}) ||
		got.Entries[1].Note != "gh note" || !slices.Equal(got.Entries[1].Tags, []string{"git"}) {
		t.Fatalf("link notes = %+v", got.Entries)
	}
	var buf bytes.Buffer
	if _, err := Write(&buf, p, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if pv.Entries[0].Note != p.Entries[0].Note || !slices.Equal(pv.Entries[0].Tags, []string{"QoL", "UI"}) {
		t.Fatalf("file notes = %+v", pv.Entries[0])
	}
}

func TestCollectOmitsEntryNotesWhenDisabled(t *testing.T) {
	p := profile.Profile{Name: "x", Entries: []profile.Entry{
		{
			Key: "n", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2},
			Mods: []profile.EntryMod{{UniqueID: "A", Folder: "."}}, Note: "secret", Tags: []string{"t"},
		},
	}}
	s, _, _ := Collect(p, Include{Notes: false, FomodChoices: true})
	if len(s.Entries) != 1 || s.Entries[0].Note != "" || len(s.Entries[0].Tags) != 0 {
		t.Fatalf("collect = %+v", s.Entries[0])
	}
}

func TestImportEntryNotesTruncates(t *testing.T) {
	long := strings.Repeat("n", profile.MaxEntryNote+10)
	ref := Ref{ModID: 1, FileID: 2, Note: long, Tags: []string{
		strings.Repeat("t", profile.MaxEntryTag+1), "ok", "", "ok", "dup", "DUP",
		"a", "b", "c", "d", "e", "f", "g", "h", "i",
	}}
	var e profile.Entry
	ImportEntryNotes(&e, ref)
	if utf8.RuneCountInString(e.Note) != profile.MaxEntryNote {
		t.Fatalf("note len = %d", utf8.RuneCountInString(e.Note))
	}
	if !slices.Equal(e.Tags, []string{"ok", "dup", "a", "b", "c", "d", "e", "f"}) {
		t.Fatalf("tags = %#v", e.Tags)
	}
}

func TestEncodeOmitsDescription(t *testing.T) {
	with := sample()
	without := sample()
	without.Description = ""
	a, err := Encode(with)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encode(without)
	if err != nil {
		t.Fatal(err)
	}
	if a.Payload != b.Payload {
		t.Fatal("description changed the share link payload")
	}
}

func zipOf(t *testing.T, files ...[2]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(f[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "x.mortar")
	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestReadRejects(t *testing.T) {
	head := [2]string{"profile.json", `{"version":1,"name":"x","notes":"","entries":[[1,2]],"uniqueIds":["A.one"]}`}
	for name, files := range map[string][][2]string{
		"no profile":         {{"configs/A.one/c.json", "{}"}},
		"traversal":          {head, {"configs/A.one/../../x.json", "{}"}},
		"traversal mid":      {head, {"configs/A.one/a/../../x.json", "{}"}},
		"absolute":           {head, {"configs/A.one//etc/x.json", "{}"}},
		"backslash":          {head, {"configs/A.one/a\\b.json", "{}"}},
		"drive":              {head, {"configs/A.one/C:x.json", "{}"}},
		"non-json":           {head, {"configs/A.one/run.exe", "MZ"}},
		"json in name only":  {head, {"configs/A.one/x.json.dll", "MZ"}},
		"dotfile segment":    {head, {"configs/A.one/.hidden/x.json", "{}"}},
		"reserved":           {head, {"configs/A.one/NUL.json", "{}"}},
		"unknown uniqueid":   {head, {"configs/Evil.Mod/c.json", "{}"}},
		"traversal uniqueid": {head, {"configs/../c.json", "{}"}},
		"outside layout":     {head, {"mods/x.json", "{}"}},
		"duplicate":          {head, {"configs/A.one/c.json", "{}"}, {"configs/a.ONE/C.json", "{}"}},
		"no uniqueid folder": {head, {"configs/c.json", "{}"}},
		"bad id in list":     {{"profile.json", `{"version":1,"name":"x","entries":[],"uniqueIds":["../x"]}`}},
		"newer version":      {{"profile.json", `{"version":3,"name":"x","entries":[]}`}},
		"bad entry":          {{"profile.json", `{"version":1,"name":"x","entries":[[0,1]]}`}},
		"bad name":           {{"profile.json", `{"version":1,"name":"","entries":[]}`}},
		"not json":           {{"profile.json", `nope`}},
		"oversized config":   {head, {"configs/A.one/c.json", strings.Repeat(" ", MaxConfigBytes+1)}},
		"oversized profile":  {{"profile.json", `{"version":1,"name":"x","notes":"` + strings.Repeat("a", maxProfileBytes) + `","entries":[]}`}},
	} {
		if _, err := Read(zipOf(t, files...)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := Read(zipOf(t, head, [2]string{"configs/A.one/ok.json", "{}"})); err != nil {
		t.Errorf("valid file refused: %v", err)
	}
	if _, err := Read(filepath.Join(t.TempDir(), "missing.mortar")); err == nil {
		t.Error("missing file accepted")
	}
	junk := filepath.Join(t.TempDir(), "j.mortar")
	if err := os.WriteFile(junk, []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(junk); !errors.Is(err, ErrBadFile) {
		t.Errorf("junk: %v", err)
	}
}

func TestReadEntryCap(t *testing.T) {
	files := [][2]string{{"profile.json", `{"version":1,"name":"x","entries":[],"uniqueIds":["A.one"]}`}}
	for i := range MaxConfigFiles + 1 {
		files = append(files, [2]string{fmt.Sprintf("configs/A.one/%d.json", i), "{}"})
	}
	if _, err := Read(zipOf(t, files...)); err == nil {
		t.Error("too many entries accepted")
	}
}
