package lan

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// FuzzHandshake posts one body to each endpoint a peer on the network can reach before pairing: a profile share and
// both pairing steps, with a pairing code open.
func FuzzHandshake(f *testing.F) {
	var buf bytes.Buffer
	farm := profile.Profile{Name: "Farm friends", Entries: []profile.Entry{{Key: "one", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2}}}}
	if _, err := share.Write(&buf, "stardew", farm, f.TempDir()); err != nil {
		f.Fatal(err)
	}
	payload := base64.RawStdEncoding.EncodeToString(buf.Bytes())
	for _, v := range []any{
		shareRequest{Sender: "Alex", Game: "stardew", Payload: payload, Version: protocolVersion, SenderID: "id", SenderPort: 9, Nonce: "n", Proof: "p", Digests: map[string]entryDigest{"k": {Hash: "h", MAC: "m"}}},
		shareRequest{Sender: " Alex", Game: "lethal-company", Payload: payload + " ", Version: protocolVersion},
		pairBeginRequest{ID: "joiner", Name: "Laptop", A: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
		pairFinishRequest{Session: "s", Proof: "p"},
	} {
		b, err := json.Marshal(v)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add([]byte(`{"sender":"a"} {"x":1}`))
	service := NewService(Deps{Dir: f.TempDir()})
	service.mu.Lock()
	service.enabled = true
	service.mu.Unlock()
	if _, err := service.PairCode(); err != nil {
		f.Fatal(err)
	}
	handler := service.handler()
	f.Fuzz(func(t *testing.T, body []byte) {
		for _, endpoint := range []string{"/share", "/pair/begin", "/pair/finish"} {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodPost, endpoint, bytes.NewReader(body)))
			if endpoint != "/share" || w.Code != http.StatusOK {
				continue
			}
			request, err := decodeRequest(body)
			if err != nil {
				t.Fatalf("accepted a share that does not decode: %v", err)
			}
			if _, err := validateRequest(request); err != nil {
				t.Fatalf("accepted an invalid share: %v", err)
			}
		}
		service.Inbox()
	})
}

// FuzzExtractTar feeds a store transfer of two entries with fuzzed names, kinds and link targets: nothing may land
// outside the receiving folder, and nothing but folders and regular files inside it.
func FuzzExtractTar(f *testing.F) {
	for _, s := range []struct {
		a, b, link string
		kind       byte
	}{
		{"nexus-1-2/mod/a.dll", "nexus-1-2/mod/b.json", "", tar.TypeReg},
		{"../evil", "x", "", tar.TypeReg},
		{"link", "link/x", "/tmp", tar.TypeSymlink},
		{"hard", "y", "../sentinel/z", tar.TypeLink},
		{"d", "d/../../x", "", tar.TypeDir},
		{`a\..\..\x`, "/abs", "", tar.TypeReg},
		{"fifo", "z", "", tar.TypeFifo},
	} {
		f.Add(s.a, s.b, s.link, s.kind, []byte("data"))
	}
	f.Fuzz(func(t *testing.T, a, b, link string, kind byte, data []byte) {
		var buf bytes.Buffer
		w := tar.NewWriter(&buf)
		for i, name := range []string{a, b} {
			h := &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(data))}
			if i == 0 {
				h.Typeflag, h.Linkname = kind, link
				if kind != tar.TypeReg {
					h.Size = 0
				}
			}
			if w.WriteHeader(h) != nil {
				return
			}
			_, _ = w.Write(data[:h.Size])
		}
		if w.Close() != nil {
			return
		}
		parent := t.TempDir()
		root, sentinel := filepath.Join(parent, "store"), filepath.Join(parent, "sentinel")
		for _, d := range []string{root, sentinel} {
			if err := os.Mkdir(d, 0o700); err != nil {
				t.Fatal(err)
			}
		}
		var total int64
		_ = extractTar(t.Context(), &buf, root, &total, func(int64) {})
		if top, _ := os.ReadDir(parent); len(top) != 2 {
			t.Fatalf("extraction wrote beside its folder: %v", top)
		}
		if side, _ := os.ReadDir(sentinel); len(side) != 0 {
			t.Fatalf("extraction wrote into a sibling: %v", side)
		}
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && !d.Type().IsRegular() {
				t.Fatalf("extracted a %v at %s", d.Type(), p)
			}
			return nil
		})
	})
}
