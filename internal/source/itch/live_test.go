package itch

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// liveGame is a free Sims 4 mod page (roburky's Emotional Inertia Classic) whose upload needs no purchase.
const liveGame = "134565"

// TestLive runs the driver against itch.io with the key in the OS keyring, so it runs only when
// MORTAR_LIVE_SOURCES is set and a key is stored. The key and the signed download address are never logged.
func TestLive(t *testing.T) {
	if os.Getenv("MORTAR_LIVE_SOURCES") == "" {
		t.Skip("set MORTAR_LIVE_SOURCES=1 to query itch.io")
	}
	key, err := secret.Get(keyName)
	if err != nil {
		t.Skipf("no itch.io key in the keyring: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	d := Driver{}

	name, err := d.Validate(ctx, key)
	if err != nil || name == "" {
		t.Fatalf("validate: %q, %v", name, err)
	}
	t.Logf("signed in as %s", name)

	page, err := d.Search(ctx, source.Query{Key: "sims", Page: source.FirstPage})
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("search: %d items, %v", len(page.Items), err)
	}
	t.Logf("search: %d items, first %q", len(page.Items), page.Items[0].Name)

	got, err := d.Resolve(ctx, liveGame, "")
	if err != nil || got.URL == "" || got.FileName == "" {
		t.Fatalf("resolve: file %q, %v", got.FileName, err)
	}
	t.Logf("resolved upload %s: %s, %d bytes", got.UploadID, got.FileName, got.Size)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, got.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK || n != got.Size {
		t.Fatalf("download: status %d, %d of %d bytes, %v", resp.StatusCode, n, got.Size, err)
	}
	t.Logf("downloaded %d bytes", n)
}
