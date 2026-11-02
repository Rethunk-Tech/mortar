package modio

import (
	"crypto/md5" //nolint:gosec // G501: MD5 is the only hash mod.io publishes for a file
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// TestLive runs the driver against the real API. MORTAR_MODIO_LIVE=1 turns it on; the token comes from
// MORTAR_MODIO_TOKEN, else the keyring.
func TestLive(t *testing.T) {
	if os.Getenv("MORTAR_MODIO_LIVE") == "" {
		t.Skip("MORTAR_MODIO_LIVE is not set")
	}
	const timberborn = "3659"
	d := Driver{}
	if token := os.Getenv("MORTAR_MODIO_TOKEN"); token != "" {
		d.Token = func() (string, error) { return token, nil }
	}
	if why := d.Unavailable(); why != "" {
		t.Fatalf("unavailable: %s", why)
	}
	ctx := t.Context()
	page, err := d.Search(ctx, source.Query{Game: "timberborn", Key: timberborn, Sort: source.SortDownloads, Size: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		t.Fatal("no mods")
	}
	t.Logf("search: %d hits, first %q", len(page.Items), page.Items[0].Name)
	cats, err := d.Categories(ctx, timberborn)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("categories: %d", len(cats))
	for _, item := range page.Items {
		id := item.ID
		if _, err := d.Details(ctx, timberborn, id, ""); err != nil {
			t.Fatal(err)
		}
		f, err := d.Resolve(ctx, timberborn, id, "")
		if err != nil {
			t.Fatal(err)
		}
		if f.Size > 32<<20 {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		sum := md5.New() //nolint:gosec // G401: MD5 is the only hash mod.io publishes for a file
		n, err := io.Copy(sum, res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, f.MD5) || n != f.Size {
			t.Fatalf("%s: md5 %s size %d, want %s size %d", f.FileName, got, n, f.MD5, f.Size)
		}
		t.Logf("downloaded %s (%d bytes) and its MD5 matches", f.FileName, n)
		return
	}
	t.Fatal("no file small enough to download")
}
