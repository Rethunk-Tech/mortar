package lan

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	_, joinerMsg, err := pairConfig().NewJoiner("AAAAAAAA", "joiner", "Laptop")
	if err != nil {
		f.Fatal(err)
	}
	for _, v := range []any{
		shareRequest{Sender: "Alex", Game: "stardew", Payload: payload, Version: protocolVersion, SenderID: "id", SenderPort: 9, Digests: map[string]entryDigest{"k": {Hash: "h", MAC: "m"}}},
		shareRequest{Sender: " Alex", Game: "lethal-company", Payload: payload + " ", Version: protocolVersion},
		pairBeginRequest{Msg: base64.RawStdEncoding.EncodeToString(joinerMsg)},
		pairFinishRequest{Session: "s", Confirm: "AAAA"},
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
	if _, err := service.PairCode(f.Context()); err != nil {
		f.Fatal(err)
	}
	handler := service.handler()
	f.Fuzz(func(t *testing.T, body []byte) {
		for _, endpoint := range []string{"/share", pairBeginPath, pairFinishPath} {
			w := httptest.NewRecorder()
			post := httptest.NewRequestWithContext(t.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
			post.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(w, post)
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
