package share

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/andybalholm/brotli"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func nexus(key string, im, file int, disabled ...mod.ID) profile.Entry {
	return profile.Entry{
		Key:      key,
		Source:   profile.Source{Kind: profile.KindNexus, ModID: im, FileID: file},
		Mods:     []profile.Component{{ID: mod.SMAPI("A." + key), Folder: "."}},
		Disabled: disabled,
	}
}

func sample() profile.Profile {
	gh := profile.Entry{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1.2.3", Asset: "mod-1.2.3.zip"}}
	return profile.Profile{Name: "Farm 🌾", Notes: "n", Description: "co-op Fridays", Entries: []profile.Entry{
		{Key: "smapi-4.1", Source: profile.Source{Kind: profile.SourceSMAPI}},
		{Key: "bridge-1", Source: profile.Source{Kind: profile.SourceMortar}},
		nexus("one", 541, 1000),
		nexus("off", 7, 8, "smapi:A.off"),
		{Key: "loc.zip", Source: profile.Source{Kind: profile.KindLocal, Name: "loc.zip"}},
		gh,
		nexus("two", 2, 3),
	}}
}

func TestRoundTripAndLinks(t *testing.T) {
	t.Parallel()
	res, err := Encode("stardew", sample(), profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Ref{{ModID: 541, FileID: 1000}, {GitHub: "owner/repo@v1.2.3/mod-1.2.3.zip"}, {ModID: 2, FileID: 3}}
	if len(res.LeftOut) != 1 || res.LeftOut[0] != (LeftOut{Key: "loc.zip", Reason: "local archive"}) {
		t.Fatalf("left out = %+v", res.LeftOut)
	}
	for _, in := range []string{res.Web, res.App, res.App + "/", "  " + res.Payload + "\n"} {
		got, err := Parse(in)
		if err != nil || got.Name != "Farm 🌾" || fmt.Sprint(got.Entries) != fmt.Sprint(want) ||
			got.Game != "stardew" || got.SourceKeys["nexus"] != "stardewvalley" {
			t.Fatalf("Parse(%.40q) = %+v, %v", in, got, err)
		}
	}
	lc, err := Encode("lethal-company", sample(), profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{lc.Web, lc.App} {
		if got, err := Parse(in); err != nil || got.Game != "lethal-company" {
			t.Fatalf("Parse(%.40q) game = %q, %v", in, got.Game, err)
		}
	}
}

func TestDetailsRoundTrip(t *testing.T) {
	t.Parallel()
	choices := map[string]map[string][]string{"Options": {"Pack": {"Optional"}}}
	p := profile.Profile{Name: "Choices", Entries: []profile.Entry{{
		Key: "n", Source: profile.Source{Kind: profile.KindNexus, ModID: 4, FileID: 5},
		Mods: []profile.Component{{ID: "smapi:A.On", Folder: "."}, {ID: "smapi:A.Off", Folder: "."}}, Disabled: []mod.ID{"smapi:A.Off"}, Fomod: choices,
	}}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(res.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || !slices.Equal(got.Entries[0].Disabled, []mod.ID{"smapi:A.Off"}) ||
		fmt.Sprint(got.Entries[0].Fomod) != fmt.Sprint(choices) {
		t.Fatalf("link details = %+v", got.Entries)
	}

	dir := modsDirWith(t, map[string]string{"n/config.json": "{}"})
	var buf bytes.Buffer
	if _, err := Write(&buf, "stardew", p, dir); err != nil {
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
	if pv.Game != "stardew" || pv.SourceKeys["nexus"] != "stardewvalley" {
		t.Fatalf("file origin = %q %v", pv.Game, pv.SourceKeys)
	}
	if len(pv.Entries) != 1 || !slices.Equal(pv.Entries[0].Disabled, []mod.ID{"smapi:A.Off"}) ||
		fmt.Sprint(pv.Entries[0].Fomod) != fmt.Sprint(choices) {
		t.Fatalf("file details = %+v", pv.Entries)
	}
}

func TestV3WireShape(t *testing.T) {
	t.Parallel()
	doc := `[3,"mixed","stardew",{"nexus":"stardewvalley"},[` +
		`{"s":"nexus","mod":4,"file":5,"disabled":["A.Off"],"note":"n","tags":["t"]},` +
		`{"s":"github","repo":"owner/repo","tag":"v1","asset":"a.zip","note":"g"}]]`
	got, err := Parse(pack(t, doc))
	if err != nil {
		t.Fatal(err)
	}
	if got.Game != "stardew" || got.SourceKeys["nexus"] != "stardewvalley" || len(got.Entries) != 2 ||
		got.Entries[0].ModID != 4 || got.Entries[0].FileID != 5 || got.Entries[0].Note != "n" ||
		got.Entries[1].GitHub != "owner/repo@v1/a.zip" || got.Entries[1].Note != "g" {
		t.Fatalf("v3 = %+v", got)
	}
	out, err := json.Marshal(got.Entries)
	if err != nil || !strings.Contains(string(out), `{"s":"github","repo":"owner/repo","tag":"v1","asset":"a.zip","note":"g"}`) ||
		!strings.Contains(string(out), `{"s":"nexus","mod":4,"file":5,`) {
		t.Fatalf("wire = %s, %v", out, err)
	}
	if _, err := Parse(pack(t, `[3,"x","stardew",{"nexus":"stardewvalley"},[]]`)); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("mortar://lethal-company/p/" + pack(t, doc)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("link naming another game: %v", err)
	}
}

func TestLinkDropsDetailsBeforeRefs(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "large"}
	for i := range 250 {
		sum := sha256.Sum256(fmt.Appendf(nil, "choice-%d", i))
		p.Entries = append(p.Entries, profile.Entry{
			Key: fmt.Sprint(i), Source: profile.Source{Kind: profile.KindNexus, ModID: i + 1, FileID: i + 2},
			Mods:  []profile.Component{{ID: mod.SMAPI(fmt.Sprintf("A.%d", i))}},
			Fomod: map[string]map[string][]string{"Step": {fmt.Sprintf("Group-%d", i): {fmt.Sprintf("%x", sum)}}},
		})
	}
	res, err := Encode("stardew", p, profile.ShareFacts{})
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
	t.Parallel()
	p, err := Encode("stardew", sample(), profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{
		"", "https://evil.example/stardew/p#" + p.Payload, "https://mortar.rethunk.tech/Lethal/p#" + p.Payload,
		"https://mortar.rethunk.tech/stardew/x#" + p.Payload, "http://mortar.rethunk.tech/stardew/p#" + p.Payload,
		"mortar://lethal company/p/" + p.Payload, "mortar://stardew/q/" + p.Payload, "javascript:alert(1)",
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
	t.Parallel()
	for _, doc := range []string{`[4,"x","stardew",{},[]]`, `[9,{"weird":true}]`} {
		if _, err := Parse(pack(t, doc)); !errors.Is(err, ErrNewerVersion) {
			t.Errorf("%s: err = %v", doc, err)
		}
	}
}

func TestHostileShapes(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 61)
	many := "[" + strings.Repeat("[1,2],", MaxEntries) + "[1,2]]"
	for _, doc := range []string{
		``, `{}`, `[]`, `[1]`, `[3,"x"]`, `[3,"x","stardew",{"nexus":"k"},[],4]`, `[0,"x",[]]`, `[-1,"x",[]]`, `["1","x",[]]`, `[1.5,"x",[]]`,
		`[1,5,[]]`, `[2,"x",[]]`, `[2,"x","stardew",{},[]]`, `[3,"x","stardew",{},[[1,2]]]`, `[3,"x","stardew",{},["o/r@t/a"]]`, `[3,"x","stardew",{},[{"s":"nexus","mod":1,"file":2}]]`, `[3,"x","Bad Game",{"nexus":"k"},[]]`, `[3,"x","stardew",{"nexus":"a b"},[]]`, `[3,"x","stardew",{"nexus":"k"},[{"s":"other","mod":1,"file":2}]]`, `[3,"x","stardew",{"nexus":"k"},[{"s":"github","repo":"o/r","tag":"t"}]]`, `[3,"x","stardew",{"nexus":"k"},[{"s":"nexus","mod":1,"file":2,"repo":"o/r"}]]`, `[3,"",[]]`, `[3," x",[]]`, `[3,"` + long + `",[]]`, "[1,\"a\\u0000b\",[]]",
		`[3,"x","stardew",{"nexus":"k"},{}]`, `[3,"x","stardew",{"nexus":"k"},[[1]]]`, `[3,"x","stardew",{"nexus":"k"},[[1,2,3]]]`, `[3,"x","stardew",{"nexus":"k"},[[0,2]]]`, `[3,"x","stardew",{"nexus":"k"},[[1,-2]]]`, `[3,"x","stardew",{"nexus":"k"},[[1,2147483648]]]`,
		`[3,"x","stardew",{"nexus":"k"},[["1","2"]]]`, `[3,"x","stardew",{"nexus":"k"},[[1.5,2]]]`, `[3,"x","stardew",{"nexus":"k"},[null]]`, `[3,"x","stardew",{"nexus":"k"},[7]]`, `[3,"x","stardew",{"nexus":"k"},[""]]`,
		`[3,"x","stardew",{"nexus":"k"},["../etc/passwd"]]`, `[3,"x","stardew",{"nexus":"k"},["o/r@t/a b"]]`, `[3,"x","stardew",{"nexus":"k"},["o/r@t/../a"]]`, `[3,"x","stardew",{"nexus":"k"},["o/r@t/a/b"]]`,
		`[3,"x","stardew",{"nexus":"k"},["o/r@/a"]]`, `[3,"x","stardew",{"nexus":"k"},["o/..@t/a"]]`, `[3,"x","stardew",{"nexus":"k"},["o/.@t/a"]]`, `[3,"x","stardew",{"nexus":"k"},["o/r@..` + `/a"]]`, `[3,"x","stardew",{"nexus":"k"},["o/r@t/.."]]`, `[3,"x","stardew",{"nexus":"k"},["/r@t/a"]]`, `[3,"x","stardew",{"nexus":"k"},[[1,2]]] trailing`, `[3,"x","stardew",{"nexus":"k"},` + many + `]`,
	} {
		_, err := Parse(pack(t, doc))
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%.60q: err = %v, want ErrMalformed", doc, err)
		}
	}
}

func TestCaps(t *testing.T) {
	t.Parallel()
	if _, err := Parse(strings.Repeat("A", MaxEncoded+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized payload: %v", err)
	}
	if _, err := Parse("mortar://stardew/p/" + strings.Repeat("A", MaxEncoded+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized link: %v", err)
	}
	// A bomb: tiny once compressed, enormous inflated. The decompressed cap must trip without inflating it all.
	bomb := pack(t, `[3,"x","stardew",{},[`+strings.Repeat(" ", 4<<20)+`]]`)
	if len(bomb) > MaxEncoded {
		t.Fatalf("bomb is %d chars, want it under the encoded cap", len(bomb))
	}
	if _, err := Parse(bomb); !errors.Is(err, ErrTooLarge) {
		t.Errorf("bomb: %v", err)
	}
	// Exactly at the decompressed cap parses on to the shape check instead of tripping it.
	const head = `[3,"x","stardew",{},[`
	edge := head + strings.Repeat(" ", MaxDecoded-len(head+`]]`)) + `]]`
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
	t.Parallel()
	p := profile.Profile{Name: "big"}
	for i := range 1000 {
		p.Entries = append(p.Entries, nexus(fmt.Sprint(i), pseudo(1, i, 0, 2_000_000_000), pseudo(1, i, 1, 2_000_000_000)))
	}
	if _, err := Encode("stardew", p, profile.ShareFacts{}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("err = %v", err)
	}
	if _, err := Encode("stardew", profile.Profile{Name: strings.Repeat("a", 61)}, profile.ShareFacts{}); !errors.Is(err, ErrMalformed) {
		t.Errorf("long name: %v", err)
	}
}

// Deterministic ids in the ranges of real mods must keep the link sizes in this neighbourhood, so a codec
// regression shows up here.
// pseudo returns a deterministic id in 1..limit, spread like real ones.
func pseudo(seed, i, field, limit int) int {
	sum := sha256.Sum256(fmt.Appendf(nil, "%d/%d/%d", seed, i, field))
	return int(binary.BigEndian.Uint32(sum[:4]))%limit + 1
}

func TestLinkSizes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ mods, min, max int }{{50, 540, 660}, {100, 900, 1100}, {200, 1600, 1950}} {
		p := profile.Profile{Name: "Sample profile"}
		for i := range tc.mods {
			p.Entries = append(p.Entries, nexus(fmt.Sprint(i), pseudo(tc.mods, i, 0, 42000), pseudo(tc.mods, i, 1, 180000)))
		}
		res, err := Encode("stardew", p, profile.ShareFacts{})
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
	t.Parallel()
	p := profile.Profile{Name: "x", Entries: []profile.Entry{
		{Key: "smapi", Source: profile.Source{Kind: profile.SourceSMAPI}},
		{Key: "bridge", Source: profile.Source{Kind: profile.SourceMortar}},
		nexus("gone", 1, 2, "smapi:A.gone"),
	}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
	if err != nil || len(res.LeftOut) != 0 || len(res.Shared.Entries) != 0 {
		t.Fatalf("%+v, %v", res, err)
	}
}

func TestLeftOutReasons(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "x", Entries: []profile.Entry{
		{Key: "a", Source: profile.Source{Kind: profile.KindNexus}},
		{Key: "b", Source: profile.Source{Kind: profile.KindGitHub, Repo: "bad name"}},
		{Key: "c", Source: profile.Source{Kind: "weird"}},
	}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
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
	t.Parallel()
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
	p.Entries[5].Mods = []profile.Component{{ID: "smapi:A.gh", Folder: "."}}
	var buf bytes.Buffer
	skipped, err := Write(&buf, "stardew", p, dir)
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
		got[c.ID.Local()+"/"+c.Path] = string(c.Data)
	}
	want := map[string]string{"A.one/config.json": `{"a":1}`, "A.one/data/deep.JSON": `{"b":2}`, "A.two/config.json": `{"c":3}`, "A.gh/config.json": `{"g":1}`}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("configs = %v, want %v", got, want)
	}
}

func TestEntryNotesRoundTrip(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "Notes", Entries: []profile.Entry{
		nexus("one", 541, 1000),
		{
			Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1", Asset: "a.zip"},
			Mods: []profile.Component{{ID: "smapi:G", Folder: "."}}, Note: "gh note", Tags: []string{"git"},
		},
	}}
	p.Entries[0].Note = "farm tweak\n\tsecond line"
	p.Entries[0].Tags = []string{"QoL", "UI"}
	res, err := Encode("stardew", p, profile.ShareFacts{})
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
	if _, err := Write(&buf, "stardew", p, t.TempDir()); err != nil {
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

func TestGroupsFileRoundTrip(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "G", Entries: []profile.Entry{
		nexus("one", 541, 1000),
		nexus("two", 2, 3),
		{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"}},
	}, Groups: []profile.Group{{Name: "Core", Keys: []string{"one", "two", "gh"}}}}
	var buf bytes.Buffer
	if _, err := Write(&buf, "stardew", p, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Groups) != 1 || pv.Groups[0].Name != "Core" || len(pv.Groups[0].Refs) != 3 || pv.Groups[0].Refs[2].GitHub != "o/r@v1/a.zip" {
		t.Fatalf("groups = %+v", pv.Groups)
	}
	imported := profile.Profile{Entries: []profile.Entry{
		nexus("alpha", 541, 1000),
		nexus("beta", 2, 3),
		{Key: "gamma", Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"}},
	}}
	keys := ResolveGroupKeys(imported, pv.Groups[0])
	if !slices.Equal(keys, []string{"alpha", "beta", "gamma"}) {
		t.Fatalf("resolved = %v", keys)
	}
	var omit bytes.Buffer
	if _, err := Write(&omit, "stardew", p, t.TempDir(), Include{Notes: false, FomodChoices: true}); err != nil {
		t.Fatal(err)
	}
	off, err := ReadBytes(omit.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(off.Groups) != 0 {
		t.Fatalf("notes off still carried groups: %+v", off.Groups)
	}
}

func TestCollectOmitsEntryNotesWhenDisabled(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "x", Entries: []profile.Entry{
		{
			Key: "n", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2},
			Mods: []profile.Component{{ID: "smapi:A", Folder: "."}}, Note: "secret", Tags: []string{"t"},
		},
	}}
	s, _, _ := Collect(p, Include{Notes: false, FomodChoices: true})
	if len(s.Entries) != 1 || s.Entries[0].Note != "" || len(s.Entries[0].Tags) != 0 {
		t.Fatalf("collect = %+v", s.Entries[0])
	}
}

func TestImportEntryNotesTruncates(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	with := sample()
	without := sample()
	without.Description = ""
	a, err := Encode("stardew", with, profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encode("stardew", without, profile.ShareFacts{})
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
	t.Parallel()
	head := [2]string{"profile.json", `{"version":3,"game":"stardew","sourceKeys":{"nexus":"stardewvalley"},"name":"x","notes":"","entries":[{"s":"nexus","mod":1,"file":2}],"ids":["smapi:A.one"]}`}
	for name, files := range map[string][][2]string{
		"no profile":         {{"configs/smapi/A.one/c.json", "{}"}},
		"traversal":          {head, {"configs/smapi/A.one/../../x.json", "{}"}},
		"traversal mid":      {head, {"configs/smapi/A.one/a/../../x.json", "{}"}},
		"absolute":           {head, {"configs/smapi/A.one//etc/x.json", "{}"}},
		"backslash":          {head, {"configs/smapi/A.one/a\\b.json", "{}"}},
		"drive":              {head, {"configs/smapi/A.one/C:x.json", "{}"}},
		"non-json":           {head, {"configs/smapi/A.one/run.exe", "MZ"}},
		"json in name only":  {head, {"configs/smapi/A.one/x.json.dll", "MZ"}},
		"dotfile segment":    {head, {"configs/smapi/A.one/.hidden/x.json", "{}"}},
		"reserved":           {head, {"configs/smapi/A.one/NUL.json", "{}"}},
		"unknown uniqueid":   {head, {"configs/smapi/Evil.Mod/c.json", "{}"}},
		"traversal uniqueid": {head, {"configs/../c.json", "{}"}},
		"outside layout":     {head, {"mods/x.json", "{}"}},
		"duplicate":          {head, {"configs/smapi/A.one/c.json", "{}"}, {"configs/smapi/a.ONE/C.json", "{}"}},
		"no uniqueid folder": {head, {"configs/c.json", "{}"}},
		"bad id in list":     {{"profile.json", `{"version":3,"game":"stardew","sourceKeys":{"nexus":"stardewvalley"},"name":"x","entries":[],"ids":["smapi:../x"]}`}},
		"newer version":      {{"profile.json", `{"version":4,"name":"x","entries":[]}`}},
		"bad entry":          {{"profile.json", `{"version":3,"game":"stardew","sourceKeys":{"nexus":"stardewvalley"},"name":"x","entries":[{"s":"nexus","mod":0,"file":1}]}`}},
		"version 2":          {{"profile.json", `{"version":2,"name":"x","entries":[]}`}},
		"bad name":           {{"profile.json", `{"version":3,"game":"stardew","sourceKeys":{"nexus":"stardewvalley"},"name":"","entries":[]}`}},
		"not json":           {{"profile.json", `nope`}},
		"oversized config":   {head, {"configs/smapi/A.one/c.json", strings.Repeat(" ", MaxConfigBytes+1)}},
		"oversized profile":  {{"profile.json", `{"version":3,"game":"stardew","sourceKeys":{"nexus":"stardewvalley"},"name":"x","notes":"` + strings.Repeat("a", maxProfileBytes) + `","entries":[]}`}},
	} {
		if _, err := Read(zipOf(t, files...)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := Read(zipOf(t, head, [2]string{"configs/smapi/A.one/ok.json", "{}"})); err != nil {
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
	t.Parallel()
	files := [][2]string{{"profile.json", `{"version":3,"game":"stardew","sourceKeys":{"nexus":"stardewvalley"},"name":"x","entries":[],"ids":["smapi:A.one"]}`}}
	for i := range MaxConfigFiles + 1 {
		files = append(files, [2]string{fmt.Sprintf("configs/smapi/A.one/%d.json", i), "{}"})
	}
	if _, err := Read(zipOf(t, files...)); err == nil {
		t.Error("too many entries accepted")
	}
}

func TestOverlayPlacementTravels(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "P", Entries: []profile.Entry{
		nexus("main", 7, 1),
		{Key: "opt", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 2}, OverlayOf: "main", OverlayFrom: "[CP] X", OverlayTo: "[CP] X/assets"},
		{Key: "alt", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 3}, OverlayOf: "main", OverlayOff: true},
	}}
	res, err := Encode("stardew", p, profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(res.Web)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Overlay != nil || len(got.Entries) != 2 {
		t.Fatalf("entries = %+v", got.Entries)
	}
	if o := got.Entries[1].Overlay; o == nil || *o != (Overlay{From: "[CP] X", To: "[CP] X/assets"}) {
		t.Fatalf("overlay = %+v", got.Entries[1].Overlay)
	}
	res, err = Encode("stardew", p, profile.ShareFacts{}, Include{DisabledMods: true})
	if err != nil {
		t.Fatal(err)
	}
	if got, err = Parse(res.Web); err != nil || got.Entries[2].Overlay == nil || !got.Entries[2].Overlay.Off {
		t.Fatalf("off overlay = %+v, %v", got.Entries, err)
	}
}

func TestOverlayPlacementIsChecked(t *testing.T) {
	t.Parallel()
	for _, r := range []Ref{
		{ModID: 1, FileID: 2, Overlay: &Overlay{To: "../x"}},
		{ModID: 1, FileID: 2, Overlay: &Overlay{From: "/abs"}},
		{ModID: 1, FileID: 2, Overlay: &Overlay{To: `a\b`}},
		{GitHub: "o/r@v1/a.zip", Overlay: &Overlay{}},
	} {
		if checkShared(Shared{Name: "P", Entries: []Ref{r}}) == nil {
			t.Errorf("%+v passed", r)
		}
	}
}

func TestThunderstoreEntriesTravelInLinks(t *testing.T) {
	t.Parallel()
	p := profile.Profile{Name: "Friends", Entries: []profile.Entry{
		{Key: "a", Source: profile.Source{Kind: profile.KindThunderstore, Name: "Alice-MoreCompany", Version: "1.2.3"}},
	}}
	res, err := Encode("lethal-company", p, profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(res.App)
	if err != nil || got.SourceKeys["thunderstore"] != "lethal-company" || len(got.Entries) != 1 ||
		got.Entries[0].Package != "Alice-MoreCompany" || got.Entries[0].Version != "1.2.3" {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
	if !got.Entries[0].MatchesEntry(p.Entries[0]) {
		t.Error("the ref does not match its entry")
	}
	out, err := json.Marshal(got.Entries)
	if err != nil || string(out) != `[{"s":"thunderstore","ns":"Alice","name":"MoreCompany","version":"1.2.3"}]` {
		t.Errorf("wire = %s, %v", out, err)
	}
	// stardew has no Thunderstore community, so a package cannot name its game's source.
	if _, err := Encode("stardew", p, profile.ShareFacts{}); err == nil {
		t.Error("a Thunderstore entry was encoded for a game without a community")
	}
	if _, err := Parse(pack(t, `[3,"x","lethal-company",{"thunderstore":"lethal-company"},[{"s":"thunderstore","ns":"A","name":"B","version":"bad"}]]`)); err == nil {
		t.Error("a malformed version was accepted")
	}
}

func TestPageFactsRoundTrip(t *testing.T) {
	t.Parallel()
	s := Shared{
		Game: "stardew", SourceKeys: map[string]string{"nexus": "stardewvalley"}, Name: "Main", GameVersion: "1.6.15",
		Entries: []Ref{{ModID: 1, FileID: 2, SizeKB: roundKB(12_345), MinGame: "1.6.9"}, {ModID: 3, FileID: 4}},
	}
	payload, err := s.payload()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(payload)
	if err != nil || got.GameVersion != "1.6.15" || got.Entries[0].SizeKB != 12_000 || got.Entries[0].MinGame != "1.6.9" || got.Entries[1].SizeKB != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
	s.Entries[0].MinGame = "soon"
	if _, err := s.payload(); err == nil {
		t.Error("a minimum game version that is not a version was accepted")
	}
	for in, want := range map[int64]int64{0: 0, 7: 7, 99: 99, 155: 160, 12_345: 12_000, 995_000: 1_000_000} {
		if got := roundKB(in); got != want {
			t.Errorf("roundKB(%d) = %d, want %d", in, got, want)
		}
	}
}

// The share page's decoder reads the same file (site/share/share.test.ts), so the app and the page agree on links.
func TestSiteReadsTheAppsLinks(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../site/share/testdata/go-link.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(strings.TrimSpace(string(b)))
	if err != nil || got.Name != "Lobby" || got.Game != "lethal-company" || len(got.Entries) != 2 ||
		got.Entries[0].Package != "Alice-MoreCompany" || got.Entries[1].GitHub != "owner/repo@v1/mod.zip" {
		t.Fatalf("parse = %+v, %v", got, err)
	}
	p := profile.Profile{Name: "Lobby", Entries: []profile.Entry{
		{Key: "a", Source: profile.Source{Kind: profile.KindThunderstore, Name: "Alice-MoreCompany", Version: "1.2.3"}},
		{Key: "b", Source: profile.Source{Kind: profile.KindGitHub, Repo: "owner/repo", Tag: "v1", Asset: "mod.zip"}},
	}}
	res, err := Encode("lethal-company", p, profile.ShareFacts{})
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(res.Web)
	if err != nil || fmt.Sprint(again) != fmt.Sprint(got) {
		t.Fatalf("today's encoder makes %+v, the stored link reads %+v", again, got)
	}
}
