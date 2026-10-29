package meta

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
)

const compatSample = `[{"name": "Content Patcher", "id": "Pathoschild.ContentPatcher", "nexus": 1915, "compatibility": {"status": "ok"}}]`

func TestCompatListFetchesAgainACacheWrittenWithoutThisBuildsStamp(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		_, _ = w.Write([]byte(`[{"name":"Bus","id":"hootless.BusLocations","unofficialUpdate":{"version":"1.2.2-unofficial.1-Xytronix","url":"https://example.com/b"}}]`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), CompatURL: srv.URL}
	// An index cached before the unofficial version was kept: a day's TTL left, no build stamp, no version.
	diskcache.PutEntry(c.Cache(), CompatCacheFile, diskcache.Entry[smapiweb.CompatIndex]{Fetched: time.Now(), Value: smapiweb.CompatIndex{
		ByID: map[string]smapiweb.CompatEntry{"hootless.buslocations": {Status: smapiweb.StatusUnofficial, UnofficialURL: "https://example.com/b"}},
	}})

	idx, err := c.CompatList(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := idx.Lookup("hootless.BusLocations", 0); !ok || e.UnofficialVersion != "1.2.2-unofficial.1-Xytronix" || asked.Load() != 1 {
		t.Fatalf("entry = %+v, asked %d times; want the old-shape cache fetched again", e, asked.Load())
	}
	if _, err := c.CompatList(t.Context()); err != nil || asked.Load() != 1 {
		t.Fatalf("this build's cache answers the next call: asked %d times, %v", asked.Load(), err)
	}
}

func TestCompatListMemoIgnoresAFileStampedInTheFuture(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		_, _ = w.Write([]byte(compatSample))
	}))
	t.Cleanup(srv.Close)
	now := time.Now()
	c := &Client{HTTP: srv.Client(), CacheDir: t.TempDir(), CompatURL: srv.URL, Now: func() time.Time { return now }}
	if _, err := c.CompatList(t.Context()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(-30 * 24 * time.Hour) // clock set back a month: the file is now dated in the future
	if _, err := c.CompatList(t.Context()); err != nil || asked.Load() != 2 {
		t.Fatalf("asked %d times, %v; want the future-stamped memo refused", asked.Load(), err)
	}
}
