package lan

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = writeTar(r.Context(), w, source)
	}))
	defer server.Close()
	var emitted int
	svc := NewService(Deps{Store: receiverStore, Emit: func(string, any) { emitted++ }})
	incoming := incomingTransfer{Peer: strings.TrimPrefix(server.URL, "http://"), Game: "stardew", Token: "t"}
	if _, err := svc.fetchEntry(t.Context(), incoming, transferItem{Key: store.NexusKey(1, 1), Hash: "local-forged"}, 1, 1, 1, 0, time.Now()); err == nil {
		t.Fatal("an entry that does not match the vouched hash was installed")
	}
	hash, err := store.HashDir(source)
	if err != nil {
		t.Fatal(err)
	}
	emitted = 0
	if _, err := svc.fetchEntry(t.Context(), incoming, transferItem{Key: store.NexusKey(1, 1), Hash: hash}, 1, 1, 1, 0, time.Now()); err != nil {
		t.Fatal(err)
	}
	if emitted == 0 || emitted > 2 {
		t.Fatalf("%d files inside %v emitted %d progress events, want 1", files, progressEvery, emitted)
	}
}
