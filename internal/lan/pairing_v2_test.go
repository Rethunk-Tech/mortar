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
