package lan

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// relayed is what a host in the middle sees of the exchanges it forwards.
type relayed struct {
	mu   sync.Mutex
	reqs []relayedExchange
}

type relayedExchange struct {
	path         string
	header       http.Header
	request, rsp []byte
	status       int
}

// relay forwards every request to target keeping the Host it was dialed with, as a host posing as the target would.
func relay(t *testing.T, target string, seen *relayed) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		out, _ := http.NewRequestWithContext(r.Context(), r.Method, "http://"+target+r.URL.RequestURI(), bytes.NewReader(body))
		out.Header = r.Header.Clone()
		out.Host = r.Host
		resp, err := http.DefaultClient.Do(out)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		rb, _ := io.ReadAll(resp.Body)
		seen.mu.Lock()
		seen.reqs = append(seen.reqs, relayedExchange{path: r.URL.Path, header: r.Header.Clone(), request: body, rsp: rb, status: resp.StatusCode})
		seen.mu.Unlock()
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.Header().Del("Content-Length")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(rb)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

type pairRig struct {
	sender, receiver         *Service
	senderAddr, receiverAddr string
	listed, unlisted         string
}

// newPairRig has a sender holding two store entries and a receiver paired with it.
func newPairRig(t *testing.T) pairRig {
	t.Helper()
	senderStore := store.OpenAt(t.TempDir())
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "mod.dll"), []byte("SECRET archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	rig := pairRig{listed: store.NexusKey(7, 2), unlisted: store.NexusKey(99, 1)}
	for _, k := range []string{rig.listed, rig.unlisted} {
		if err := senderStore.AddDir(t.Context(), "stardew", k, src); err != nil {
			t.Fatal(err)
		}
	}
	rig.receiver, rig.receiverAddr = pairedService(t, store.OpenAt(t.TempDir()), nil)
	rig.sender, rig.senderAddr = pairedService(t, senderStore, nil)
	pair(t, rig.receiver, rig.sender, rig.receiverAddr)
	return rig
}

func (r pairRig) payload(t *testing.T) []byte {
	t.Helper()
	var p bytes.Buffer
	if _, err := share.Write(&p, "stardew", profile.Profile{Name: "x", Entries: []profile.Entry{
		{Key: r.listed, Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 2}},
	}}, t.TempDir(), share.Include{}); err != nil {
		t.Fatal(err)
	}
	return p.Bytes()
}

func (r pairRig) send(t *testing.T, to string) (paired bool, err error) {
	t.Helper()
	err = r.sender.sendPayload(t.Context(), to, "stardew", "", func(p bool) ([]byte, error) {
		paired = p
		return r.payload(t), nil
	})
	return paired, err
}

func TestARelayedHandshakeIsNotTreatedAsPaired(t *testing.T) {
	t.Parallel()
	rig := newPairRig(t)
	var seen relayed
	// This relay keeps the Host the sender dialed, so only the rest of the handshake can tell it from the receiver.
	paired, err := rig.send(t, relay(t, rig.receiverAddr, &seen))
	if err != nil {
		t.Fatal(err)
	}
	if !paired {
		t.Skip("the relay was refused at the handshake")
	}
	seen.mu.Lock()
	defer seen.mu.Unlock()
	var token string
	for _, ex := range seen.reqs {
		if ex.path != "/share" {
			continue
		}
		if bytes.Contains(ex.request, []byte("Mods")) || bytes.Contains(ex.request, []byte(base64.RawStdEncoding.EncodeToString(rig.payload(t))[:40])) {
			t.Error("the share crossed the relay in the clear")
		}
		var rsp shareResponse
		if json.Unmarshal(ex.rsp, &rsp) == nil {
			token = rsp.TransferToken
		}
	}
	if token != "" {
		t.Fatalf("the transfer token crossed the relay in the clear: %q", token)
	}
}

func TestARelayThatChangesTheDialedHostFailsTheHandshake(t *testing.T) {
	t.Parallel()
	rig := newPairRig(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := http.NewRequestWithContext(r.Context(), r.Method, "http://"+rig.receiverAddr+r.URL.RequestURI(), r.Body)
		out.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(out)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	t.Cleanup(srv.Close)
	paired, err := rig.send(t, strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	if paired {
		t.Fatal("a relay was proven paired")
	}
}

func TestAStorePullNeedsTheSealedTransferKey(t *testing.T) {
	t.Parallel()
	rig := newPairRig(t)
	key := bytes.Repeat([]byte{3}, 32)
	rig.sender.rememberGrant("tok", "stardew", []string{rig.listed}, key)
	get := func(set func(*http.Request)) int {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+rig.senderAddr+"/store/stardew/"+rig.listed, nil)
		set(req)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if got := get(func(r *http.Request) { r.Header.Set("Authorization", "Bearer tok") }); got != http.StatusForbidden {
		t.Errorf("a bare token = %d, want 403", got)
	}
	if got := get(func(r *http.Request) { r.Header.Set(handleHeader, tokenHandle("tok")) }); got != http.StatusForbidden {
		t.Errorf("a handle without a seal = %d, want 403", got)
	}
	resp, err := sealedDo(t.Context(), http.MethodGet, "http://"+rig.senderAddr+"/store/stardew/"+rig.listed, bytes.Repeat([]byte{4}, 32), tokenHandle("tok"), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("a seal under the wrong key = %d, want 403", resp.StatusCode)
	}
}

func TestASealedRequestIsServedOnce(t *testing.T) {
	t.Parallel()
	rig := newPairRig(t)
	key := bytes.Repeat([]byte{3}, 32)
	rig.sender.rememberGrant("tok", "stardew", []string{rig.listed}, key)
	var seen relayed
	to := relay(t, rig.senderAddr, &seen)
	resp, err := sealedDo(t.Context(), http.MethodGet, "http://"+to+"/store/stardew/"+rig.listed, key, tokenHandle("tok"), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first pull = %d", resp.StatusCode)
	}
	replay, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+rig.senderAddr+"/store/stardew/"+rig.listed, nil)
	replay.Header = seen.reqs[0].header
	again, err := http.DefaultClient.Do(replay)
	if err != nil {
		t.Fatal(err)
	}
	_ = again.Body.Close()
	if again.StatusCode != http.StatusForbidden {
		t.Fatalf("replayed pull = %d, want 403", again.StatusCode)
	}
}

// A paired receiver that lists more files than the share named is granted only the listed ones.
func TestAGrantCoversOnlyTheFilesTheShareListed(t *testing.T) {
	t.Parallel()
	rig := newPairRig(t)
	greedy := rig.receiver.handler()
	front := http.NewServeMux()
	front.HandleFunc("POST /share", rig.receiver.sealed(rig.receiver.resolveShare, true, func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		rig.receiver.handleShare(rec, r)
		var rsp shareResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &rsp); err != nil {
			t.Error(err)
		}
		rsp.EntryKeys = append(rsp.EntryKeys, rig.unlisted)
		_ = json.NewEncoder(w).Encode(rsp)
	}))
	front.Handle("/", greedy)
	srv := httptest.NewServer(front)
	t.Cleanup(srv.Close)
	if _, err := rig.send(t, strings.TrimPrefix(srv.URL, "http://")); err != nil {
		t.Fatal(err)
	}
	rig.sender.mu.RLock()
	defer rig.sender.mu.RUnlock()
	if len(rig.sender.grants) != 1 {
		t.Fatalf("grants = %d", len(rig.sender.grants))
	}
	for _, g := range rig.sender.grants {
		if !g.Keys[rig.listed] || g.Keys[rig.unlisted] || len(g.Keys) != 1 {
			t.Fatalf("granted keys = %v, want only %s", g.Keys, rig.listed)
		}
	}
}

func TestHelloDoesNotSayWhichIdsArePaired(t *testing.T) {
	t.Parallel()
	rig := newPairRig(t)
	selfID, _ := rig.sender.book.self()
	proofFor := func(id string) string {
		resp, err := http.Get("http://" + rig.receiverAddr + "/hello?" + url.Values{"id": {id}, "challenge": {"c"}}.Encode())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var h helloResponse
		if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
			t.Fatal(err)
		}
		return h.Proof
	}
	if proofFor(selfID) == "" || proofFor("someone-else") == "" {
		t.Fatal("an answer without a proof tells the caller its id is not paired")
	}
}

func TestOneHostCannotUseUpTheHandshakes(t *testing.T) {
	t.Parallel()
	_, addr := pairedService(t, nil, nil)
	codes := map[int]int{}
	for range maxNoncesPerPeer + 4 {
		resp, err := http.Get("http://" + addr + "/hello")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		codes[resp.StatusCode]++
	}
	if codes[http.StatusOK] != maxNoncesPerPeer {
		t.Fatalf("answers = %v, want %d handshakes from one host", codes, maxNoncesPerPeer)
	}
}

func TestOneHostCannotHoldEveryPairingSession(t *testing.T) {
	t.Parallel()
	host, addr := pairedService(t, nil, nil)
	if _, err := host.PairCode(t.Context()); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(pairBeginRequest{ID: "x", Name: "x", A: base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))})
	codes := map[int]int{}
	for range maxSessionsPerPeer + 3 {
		resp, err := http.Post("http://"+addr+"/pair/begin", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		codes[resp.StatusCode]++
	}
	if codes[http.StatusOK] != maxSessionsPerPeer {
		t.Fatalf("answers = %v, want %d sessions from one host", codes, maxSessionsPerPeer)
	}
}

func TestGuessesAreCountedBeforeTheyAreChecked(t *testing.T) {
	t.Parallel()
	host, _ := pairedService(t, nil, nil)
	if _, err := host.PairCode(t.Context()); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	host.pairing.mu.Lock()
	defer host.pairing.mu.Unlock()
	for range maxFailures {
		host.pairChargeLocked("192.0.2.9", now)
	}
	if err := host.pairOpen("192.0.2.9", now); err == nil {
		t.Fatal("a sixth guess was allowed")
	}
}
