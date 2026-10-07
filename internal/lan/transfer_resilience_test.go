package lan

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// entryDir makes a folder holding one file of size bytes and returns it with its transfer hash.
func entryDir(t *testing.T, size int) (string, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.dll"), bytes.Repeat([]byte("x"), size), 0o600); err != nil {
		t.Fatal(err)
	}
	hash, err := store.HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, hash
}

func TestAReaderThatPausesMidEntryStillGetsTheWholeEntry(t *testing.T) {
	t.Parallel()
	senderStore := store.OpenAt(t.TempDir())
	const size = 64 << 20
	dir, _ := entryDir(t, size)
	key := store.NexusKey(7, 1)
	if err := senderStore.AddDir(t.Context(), "stardew", key, dir); err != nil {
		t.Fatal(err)
	}
	sender := NewService(Deps{Store: senderStore})
	sender.rememberGrant("tok", "stardew", []string{key})
	server := httptest.NewUnstartedServer(sender.handler())
	server.Config = newHTTPServer(sender.handler())
	server.Start()
	defer server.Close()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/store/stardew/"+key, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer tok")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	head := make([]byte, 1<<20)
	if _, err := io.ReadFull(response.Body, head); err != nil {
		t.Fatal(err)
	}
	// Longer than the request deadline, which must not cap the whole stream.
	time.Sleep(httpTimeout + time.Second)
	rest, err := io.Copy(io.Discard, response.Body)
	if err != nil {
		t.Fatalf("read after the pause: %v", err)
	}
	if got := int64(len(head)) + rest; got < size {
		t.Fatalf("received %d bytes, want at least %d", got, size)
	}
}

func TestAGrantSlidesWithActivityAndExpiresWhenIdle(t *testing.T) {
	t.Parallel()
	s := NewService(Deps{})
	s.rememberGrant("tok", "stardew", []string{"k"})
	s.mu.Lock()
	grant := s.grants["tok"]
	grant.Expires = time.Now().Add(time.Second)
	s.grants["tok"] = grant
	s.mu.Unlock()
	if !s.grantAllows("tok", "stardew", "k") {
		t.Fatal("a live grant was refused")
	}
	s.mu.Lock()
	extended := s.grants["tok"].Expires
	s.mu.Unlock()
	if time.Until(extended) < transferTTL-time.Second {
		t.Fatalf("activity left %s on the grant, want about %s", time.Until(extended), transferTTL)
	}
	s.mu.Lock()
	grant.Expires = time.Now().Add(-time.Second)
	s.grants["tok"] = grant
	s.mu.Unlock()
	if s.grantAllows("tok", "stardew", "k") {
		t.Fatal("an idle grant still allowed a pull")
	}
}

func TestAFinishedPullEndsItsGrant(t *testing.T) {
	r := sentRig(t)
	token := r.receiver.incoming[r.arrival.ID].Token
	if !r.sender.grantAllows(token, "stardew", store.NexusKey(7, 2)) {
		t.Fatal("the grant was not live before the pull")
	}
	if err := r.receiver.Transfer(t.Context(), r.arrival.ID); err != nil {
		t.Fatal(err)
	}
	if r.sender.grantAllows(token, "stardew", store.NexusKey(7, 2)) {
		t.Fatal("the grant outlived the pull")
	}
	// Running the pull again has nothing left to fetch.
	if err := r.receiver.Transfer(t.Context(), r.arrival.ID); err != nil {
		t.Fatalf("second pull: %v", err)
	}
}

// flakyStore serves keys' entries over /store/, dropping the connection mid-stream for the first drops[key] requests
// and answering 403 for the key named by refuse.
type flakyStore struct {
	entries map[string]string
	drops   map[string]int
	refuse  atomic.Value
	hits    map[string]*atomic.Int32
}

func (f *flakyStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	hit := f.hits[key]
	n := int(hit.Add(1))
	if refused, _ := f.refuse.Load().(string); refused == key {
		http.Error(w, "no", http.StatusForbidden)
		return
	}
	if n <= f.drops[key] {
		w.Header().Set("Content-Length", "100000")
		_, _ = w.Write([]byte("partial"))
		return
	}
	_ = writeTar(r.Context(), w, f.entries[key])
}

func fetchRig(t *testing.T, keys []string, drops map[string]int) (*Service, incomingTransfer, *flakyStore) {
	t.Helper()
	flaky := &flakyStore{entries: map[string]string{}, drops: drops, hits: map[string]*atomic.Int32{}}
	items := make([]transferItem, len(keys))
	for i, key := range keys {
		dir, hash := entryDir(t, 100+i)
		flaky.entries[key] = dir
		flaky.hits[key] = new(atomic.Int32)
		items[i] = transferItem{Key: key, Hash: hash, Package: "Mod" + key}
	}
	server := httptest.NewServer(flaky)
	t.Cleanup(server.Close)
	receiver := NewService(Deps{Store: store.OpenAt(t.TempDir())})
	incoming := incomingTransfer{
		Peer: strings.TrimPrefix(server.URL, "http://"), Sender: "Desk", Game: "stardew", Token: "t",
		Items: items, Expires: time.Now().Add(time.Hour),
	}
	receiver.incoming[1] = incoming
	return receiver, incoming, flaky
}

func TestAnEntryThatDropsTwiceIsRetried(t *testing.T) {
	t.Parallel()
	key := store.NexusKey(7, 1)
	receiver, incoming, flaky := fetchRig(t, []string{key}, map[string]int{key: 2})
	if _, err := receiver.fetchEntry(t.Context(), incoming, incoming.Items[0], 1, 0, 1, 0, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := flaky.hits[key].Load(); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
	if has, _ := receiver.storeHas("stardew", key); !has {
		t.Fatal("entry was not installed")
	}
}

func TestAnEntryThatKeepsDroppingFailsAfterThreeTries(t *testing.T) {
	t.Parallel()
	key := store.NexusKey(7, 1)
	receiver, incoming, flaky := fetchRig(t, []string{key}, map[string]int{key: 99})
	_, err := receiver.fetchEntry(t.Context(), incoming, incoming.Items[0], 1, 0, 1, 0, time.Now())
	if usererr.KindOf(err) != usererr.Network || flaky.hits[key].Load() != fetchAttempts {
		t.Fatalf("err = %v, requests = %d", err, flaky.hits[key].Load())
	}
}

func TestARefusedGrantIsNotRetriedAndARerunSkipsInstalledEntries(t *testing.T) {
	t.Parallel()
	a, b := store.NexusKey(7, 1), store.NexusKey(7, 2)
	receiver, _, flaky := fetchRig(t, []string{a, b}, nil)
	flaky.refuse.Store(b)
	err := receiver.Transfer(t.Context(), 1)
	var refused refusedError
	if !errors.As(err, &refused) || refused.status != http.StatusForbidden {
		t.Fatalf("first pull = %v, want a refusal", err)
	}
	if got := flaky.hits[b].Load(); got != 1 {
		t.Fatalf("refused entry was asked for %d times, want 1", got)
	}
	flaky.refuse.Store("")
	if err := receiver.Transfer(t.Context(), 1); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if got := flaky.hits[a].Load(); got != 1 {
		t.Fatalf("installed entry was fetched %d times, want 1", got)
	}
}

func TestReceiveFailuresNameTheModAndTheCause(t *testing.T) {
	t.Parallel()
	item := transferItem{Package: "Cool Mod"}
	for _, tc := range []struct {
		err  error
		kind usererr.Kind
	}{
		{syscall.ENOSPC, usererr.DiskFull},
		{os.ErrPermission, usererr.Permission},
		{errors.New("boom"), usererr.Damaged},
	} {
		err := installError("install", item, "Desk", tc.err)
		if usererr.KindOf(err) != tc.kind || !strings.Contains(err.Error(), "could not install Cool Mod from Desk") {
			t.Errorf("%v: err = %v, kind %s", tc.err, err, usererr.KindOf(err))
		}
	}
}
