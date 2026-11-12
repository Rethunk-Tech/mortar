package lan

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestFetchEntryThrottlesFileProgress(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	receiverStore, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	const files = 40
	for i := range files {
		if err := os.WriteFile(filepath.Join(source, fmt.Sprintf("f%02d.txt", i)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	server, key := sealedStoreServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = writeTar(r.Context(), w, source)
	}))
	var emitted int
	svc := NewService(Deps{Store: receiverStore, Emit: func(string, any) { emitted++ }})
	incoming := incomingTransfer{Peer: strings.TrimPrefix(server.URL, "http://"), Game: "stardew", Token: "t", Key: key}
	if _, err := svc.fetchEntry(t.Context(), incoming, transferItem{Key: store.NexusKey(1, 1), Hash: "local-forged"}, 1, 1, 1, 0, time.Now()); err == nil {
		t.Fatal("an entry that does not match the vouched hash was installed")
	}
	hash, err := store.HashDir(source)
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Unix(1_700_000_000, 0)
	svc.now = func() time.Time { return clock }
	fetch := func() {
		t.Helper()
		if _, err := svc.fetchEntry(t.Context(), incoming, transferItem{Key: store.NexusKey(1, 1), Hash: hash}, 1, 1, 1, 0, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	// A clock that does not move is the throttle's worst case: however many reads the unpack takes, and however
	// long a loaded runner stretches that out, one entry inside one window sends one event.
	emitted = 0
	fetch()
	if emitted != 1 {
		t.Fatalf("%d files inside %v emitted %d progress events, want 1", files, progressEvery, emitted)
	}
	// Moving the clock past the window is what opens the next one, not the wall's passing.
	reads, emitted := 0, 0
	svc.now = func() time.Time {
		if reads++; reads == 2 {
			clock = clock.Add(progressEvery)
		}
		return clock
	}
	fetch()
	if emitted != 2 {
		t.Fatalf("%d files inside %v emitted %d progress events across two windows, want 2", files, progressEvery, emitted)
	}
}

func TestA404ForAVouchedEntryFailsTheTransfer(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	receiverStore, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	server, key := sealedStoreServer(t, http.HandlerFunc(http.NotFound))
	svc := NewService(Deps{Store: receiverStore})
	incoming := incomingTransfer{Peer: strings.TrimPrefix(server.URL, "http://"), Game: "stardew", Token: "t", Key: key}
	if _, err := svc.fetchEntry(t.Context(), incoming, transferItem{Key: store.NexusKey(1, 1)}, 1, 1, 1, 0, time.Now()); !errors.Is(err, errRemoteStoreEntryMissing) {
		t.Fatalf("unvouched entry answering 404 = %v, want errRemoteStoreEntryMissing", err)
	}
	if _, err := svc.fetchEntry(t.Context(), incoming, transferItem{Key: store.NexusKey(1, 2), Hash: "vouched"}, 1, 1, 1, 0, time.Now()); err == nil || errors.Is(err, errRemoteStoreEntryMissing) {
		t.Fatalf("vouched entry answering 404 = %v, want a transfer error", err)
	}
}
