package lan

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setSelf gives a service a chosen pairing id, as a second computer copied from or posing as another would have.
func setSelf(s *Service, id string) {
	s.book.mu.Lock()
	s.book.file.Self, s.book.loaded = id, true
	s.book.mu.Unlock()
}

func pairedIDs(t *testing.T, s *Service) []string {
	t.Helper()
	peers, err := s.Paired()
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(peers))
	for _, p := range peers {
		ids = append(ids, p.ID)
	}
	return ids
}

// The joiner checks the host's proof before it answers, so a wrong code is caught on the joiner's side and neither
// computer stores anything.
func TestAWrongCodeIsCaughtByTheJoinerAndPairsNeither(t *testing.T) {
	t.Parallel()
	host, hostAddr := pairedService(t, nil, nil)
	joiner, _ := pairedService(t, nil, nil)
	code, err := host.PairCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	wrong := "AAAA-AAAA"
	if code == wrong {
		wrong = "BBBB-BBBB"
	}
	err = joiner.Pair(t.Context(), hostAddr, wrong)
	if err == nil || !strings.Contains(err.Error(), "did not prove it knew the code") {
		t.Fatalf("wrong code: %v", err)
	}
	if len(pairedIDs(t, host)) != 0 || len(pairedIDs(t, joiner)) != 0 {
		t.Fatal("a wrong code left a pairing behind")
	}
}

// A host that refuses the joiner's last step leaves the joiner unpaired, so no pairing is ever one-sided.
func TestAHostThatDropsFinishLeavesTheJoinerUnpaired(t *testing.T) {
	t.Parallel()
	host, _ := pairedService(t, nil, nil)
	joiner, _ := pairedService(t, nil, nil)
	inner := host.handler()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/finish") {
			http.Error(w, "gone", http.StatusInternalServerError)
			return
		}
		inner.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	code, err := host.PairCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := joiner.Pair(t.Context(), strings.TrimPrefix(server.URL, "http://"), code); err == nil {
		t.Fatal("pairing succeeded without the host's finish")
	}
	if len(pairedIDs(t, joiner)) != 0 {
		t.Fatal("the joiner paired with a host that refused the last step")
	}
}

// Pairing changed protocol: each side tells an older Mortar to update instead of failing with a bare 404.
func TestAnOlderMortarIsToldToUpdateBeforePairing(t *testing.T) {
	t.Parallel()
	old := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(old.Close)
	joiner, _ := pairedService(t, nil, nil)
	if err := joiner.Pair(t.Context(), strings.TrimPrefix(old.URL, "http://"), "AAAA-AAAA"); err == nil || !strings.Contains(err.Error(), "update Mortar") {
		t.Fatalf("pairing with an older host: %v", err)
	}

	host, hostAddr := pairedService(t, nil, nil)
	if _, err := host.PairCode(t.Context()); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"id": "old", "name": "Old", "a": strings.Repeat("A", 43)})
	for _, path := range []string{"/pair/begin", "/pair/finish"} {
		resp, err := postJSON(t, "http://"+hostAddr+path, body)
		if err != nil {
			t.Fatal(err)
		}
		msg, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound || !bytes.Contains(msg, []byte("update Mortar")) {
			t.Fatalf("%s from an older joiner = %d %q", path, resp.StatusCode, msg)
		}
	}
}

// A pairing never replaces one the computer already holds, nor pairs it with its own id: either would let a computer
// that learned the code take over a paired computer's identity.
func TestPairingRefusesAnIDAlreadyPairedOrOwn(t *testing.T) {
	t.Parallel()
	a, aAddr := pairedService(t, nil, nil)
	b, _ := pairedService(t, nil, nil)
	pair(t, a, b, aAddr)
	aID, _ := a.book.self()
	bID, _ := b.book.self()
	aKey, bKey := a.book.key(bID), b.book.key(aID)

	// The host refuses a joiner claiming a paired id.
	posingAsB, _ := pairedService(t, nil, nil)
	setSelf(posingAsB, bID)
	code, err := a.PairCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := posingAsB.Pair(t.Context(), aAddr, code); err == nil {
		t.Error("the host replaced a paired computer's key")
	}
	if !bytes.Equal(a.book.key(bID), aKey) || len(pairedIDs(t, posingAsB)) != 0 {
		t.Error("a refused pairing changed a book")
	}

	// The joiner refuses a host claiming a paired id.
	posingAsA, posingAddr := pairedService(t, nil, nil)
	setSelf(posingAsA, aID)
	if code, err = posingAsA.PairCode(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := b.Pair(t.Context(), posingAddr, code); err == nil {
		t.Error("the joiner replaced a paired computer's key")
	}
	if !bytes.Equal(b.book.key(aID), bKey) || len(pairedIDs(t, posingAsA)) != 0 {
		t.Error("a refused pairing changed a book")
	}

	// Neither side pairs with its own id.
	twin, _ := pairedService(t, nil, nil)
	setSelf(twin, aID)
	if code, err = a.PairCode(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := twin.Pair(t.Context(), aAddr, code); err == nil {
		t.Error("a computer paired with its own id")
	}
	if ids := pairedIDs(t, a); len(ids) != 1 {
		t.Errorf("host book = %v, want only the first pairing", ids)
	}
}
