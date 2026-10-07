package thunderstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const benchPackages = 50_000

// syntheticChunk is a gzipped listing chunk of n packages shaped like the site's: a few versions each, each with
// a description and a dependency list.
func syntheticChunk(n int) []byte {
	var raw bytes.Buffer
	raw.WriteByte('[')
	for i := range n {
		if i > 0 {
			raw.WriteByte(',')
		}
		fmt.Fprintf(&raw, `{"name":"Pkg%d","owner":"Owner%d","package_url":"https://thunderstore.io/c/x/p/Owner%d/Pkg%d/",`+
			`"date_created":"2024-01-01T00:00:00Z","date_updated":"2025-01-01T00:00:00Z","rating_score":%d,"is_deprecated":false,`+
			`"has_nsfw_content":false,"categories":["Mods","Tools"],"versions":[`, i, i%700, i%700, i, i%50)
		for v := range 2 {
			if v > 0 {
				raw.WriteByte(',')
			}
			fmt.Fprintf(&raw, `{"description":"A package that does thing number %d.","icon":"https://gcdn.thunderstore.io/icons/Owner-Pkg%d-1.0.%d.png",`+
				`"version_number":"1.0.%d","dependencies":["BepInEx-BepInExPack-5.4.2100","Owner%d-Dep%d-1.2.3"],"downloads":%d,`+
				`"website_url":"https://github.com/Owner/Pkg%d","file_size":%d}`,
				i, i, v, v, i%700, i%300, i+v, i, 1000+i)
		}
		raw.WriteString(`]}`)
	}
	raw.WriteByte(']')
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	_, _ = zw.Write(raw.Bytes())
	_ = zw.Close()
	return out.Bytes()
}

// BenchmarkBuildBytesPerPackage reports what building a 50k-package listing allocates and how much of it is live at
// once, per package.
func BenchmarkBuildBytesPerPackage(b *testing.B) {
	body := syntheticChunk(benchPackages)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	d := Driver{URL: srv.URL, HTTP: srv.Client()}
	dir := b.TempDir()
	b.ReportAllocs()
	var alloc, peak uint64
	for i := range b.N {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		stop := trackPeak(&peak)
		if err := d.build(context.Background(), []string{srv.URL + "/chunk"}, filepath.Join(dir, fmt.Sprintf("p%d.json", i)), "test"); err != nil {
			b.Fatal(err)
		}
		stop()
		runtime.ReadMemStats(&after)
		alloc += after.TotalAlloc - before.TotalAlloc
	}
	b.ReportMetric(float64(alloc)/float64(b.N)/benchPackages, "alloc-B/pkg")
	b.ReportMetric(float64(peak)/benchPackages, "peak-heap-B/pkg")
}

// trackPeak samples the heap while a build runs and records the highest in-use size seen.
func trackPeak(peak *uint64) (stop func()) {
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		var m runtime.MemStats
		for {
			runtime.ReadMemStats(&m)
			if m.HeapAlloc > *peak {
				*peak = m.HeapAlloc
			}
			select {
			case <-done:
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	}()
	return func() { close(done); <-finished }
}

func TestConcurrentCallersShareOneBuild(t *testing.T) {
	var chunkHits atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/package-listing-index/") {
			_, _ = fmt.Fprintf(w, `["%s/chunk"]`, srv.URL)
			return
		}
		chunkHits.Add(1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write(syntheticChunk(10))
	}))
	defer srv.Close()
	d := Driver{URL: srv.URL, HTTP: srv.Client(), CacheDir: t.TempDir()}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if _, err := d.packages(context.Background(), "x", "test"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := chunkHits.Load(); n != 1 {
		t.Fatalf("chunk fetched %d times, want 1", n)
	}
}
