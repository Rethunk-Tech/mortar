package share

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/andybalholm/brotli"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// checkParseErr fails on an error that is none of Parse's sentinels, which the import dialog maps to its messages.
func checkParseErr(t *testing.T, err error) {
	t.Helper()
	for _, want := range []error{ErrNotLink, ErrMalformed, ErrTooLarge, ErrNewerVersion} {
		if errors.Is(err, want) {
			return
		}
	}
	t.Fatalf("untyped error: %v", err)
}

// checkCanonical re-encodes an accepted share and parses it back: the second encoding must equal the first, so
// whatever a link carries survives being shared on.
func checkCanonical(t *testing.T, s Shared) {
	t.Helper()
	first, err := s.payload()
	if errors.Is(err, ErrTooLarge) {
		return
	}
	if err != nil {
		t.Fatalf("accepted share does not re-encode: %v (%+v)", err, s)
	}
	back, err := Parse(first)
	if err != nil {
		t.Fatalf("re-encoded share does not parse: %v", err)
	}
	second, err := back.payload()
	if err != nil || second != first {
		t.Fatalf("round trip changed the share: %v\n%+v\n%+v", err, s, back)
	}
}

func packDoc(doc []byte) string {
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, brotli.BestSpeed)
	_, _ = w.Write(doc)
	_ = w.Close()
	return base64.RawURLEncoding.EncodeToString(buf.Bytes())
}

func fuzzLinks(f *testing.F) Result {
	res, err := Encode("stardew", sample(), profile.ShareFacts{})
	if err != nil {
		f.Fatal(err)
	}
	return res
}

// FuzzParse feeds pasted text: web and app links, bare payloads and near misses.
func FuzzParse(f *testing.F) {
	res := fuzzLinks(f)
	for _, s := range []string{res.Web, res.App, res.App + "/", res.Payload, "mortar://stardew/p/!!", "https://mortar.rethunk.tech/Bad/p#x", "mortar://other/p/" + res.Payload, ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		s, err := Parse(text)
		if err != nil {
			checkParseErr(t, err)
			return
		}
		checkCanonical(t, s)
	})
}

// FuzzParseDoc mutates the decompressed document, which byte mutation of the compressed payload rarely reaches.
func FuzzParseDoc(f *testing.F) {
	packed, err := base64.RawURLEncoding.DecodeString(fuzzLinks(f).Payload)
	if err != nil {
		f.Fatal(err)
	}
	doc, err := io.ReadAll(brotli.NewReader(bytes.NewReader(packed)))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(doc)
	for _, s := range []string{
		`[5,"x","stardew",{"nexus":"stardewvalley"},[[1,2],"o/r@t/a.zip"],"1.6.15"]`,
		`[5,"x","stardew",{"nexus":"k"},[{"s":"nexus","mod":1,"file":2,"off":["smapi:A"],"fomod":{"s":{"g":["p"]}}}]]`,
		`[5,"x","stardew",{"nexus":"k"},[["../etc/passwd"]]]`,
		`[9,{"weird":true}]`,
		`[5,"x","stardew",{},[]] trailing`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, doc []byte) {
		s, err := Parse(packDoc(doc))
		if err != nil {
			checkParseErr(t, err)
			return
		}
		checkCanonical(t, s)
	})
}

// checkPreview fails when an accepted .mortar file names a config path Apply or writeLoaderConfigs would refuse,
// or carries a share that does not survive re-encoding.
func checkPreview(t *testing.T, pv Preview) {
	t.Helper()
	for _, c := range pv.Configs {
		if path.Clean(c.Path) != c.Path || !validConfigPath(c.Path) || !validID(c.ID) || len(c.Data) > MaxConfigBytes {
			t.Fatalf("accepted config %q for %q (%d bytes)", c.Path, c.ID, len(c.Data))
		}
	}
	for _, c := range pv.LoaderConfigs {
		if !validLoaderConfigPath(c.Path) || len(c.Data) > MaxConfigBytes {
			t.Fatalf("accepted loader config %q", c.Path)
		}
	}
	checkCanonical(t, pv.Shared)
}

func fuzzFile(f *testing.F) []byte {
	p := sample()
	p.Groups = []profile.Group{{Name: "Core", Keys: []string{"one", "gh"}}}
	dir := f.TempDir()
	for name, body := range map[string]string{"one/config.json": `{"a":1}`, "two/data/deep.json": `{"b":2}`} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			f.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			f.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if _, err := Write(&buf, "stardew", p, dir); err != nil {
		f.Fatal(err)
	}
	return buf.Bytes()
}

// FuzzReadBytes feeds whole .mortar files, as a LAN peer or a double-clicked file delivers them.
func FuzzReadBytes(f *testing.F) {
	f.Add(fuzzFile(f))
	f.Add([]byte("PK\x05\x06" + string(make([]byte, 18))))
	f.Fuzz(func(t *testing.T, raw []byte) {
		pv, err := ReadBytes(raw)
		if err != nil {
			if !errors.Is(err, ErrBadFile) && !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrNewerVersion) && !errors.Is(err, ErrTooLarge) {
				t.Fatalf("untyped error: %v", err)
			}
			return
		}
		checkPreview(t, pv)
	})
}

// FuzzReadFileEntries writes well-formed zips from a fuzzed profile.json and one fuzzed entry, so the layout and path
// checks see far more names than mutated zip bytes reach.
func FuzzReadFileEntries(f *testing.F) {
	file := fuzzFile(f)
	zr, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	if err != nil {
		f.Fatal(err)
	}
	var doc []byte
	for _, zf := range zr.File {
		if zf.Name == profileFile {
			rc, _ := zf.Open()
			doc, _ = io.ReadAll(rc)
			_ = rc.Close()
		}
	}
	for _, name := range []string{
		"configs/smapi/A.one/config.json", "configs/smapi/A.one/../../x.json", "configs/smapi/a.ONE/Config.JSON",
		"configs/smapi/A.one/CON.json", "loader/BepInEx/config/a.cfg", "loader/../x.cfg", "loader/a/b.dll", "configs/smapi/Nope/c.json", "dir/",
	} {
		f.Add(doc, name, []byte(`{"a":1}`))
	}
	f.Fuzz(func(t *testing.T, doc []byte, name string, data []byte) {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for _, e := range []struct {
			name string
			data []byte
		}{{profileFile, doc}, {name, data}} {
			w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Store})
			if err != nil {
				return
			}
			_, _ = w.Write(e.data)
		}
		if zw.Close() != nil {
			return
		}
		pv, err := ReadBytes(buf.Bytes())
		if err == nil {
			checkPreview(t, pv)
		}
	})
}
