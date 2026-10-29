package contentpatcher

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"testing"
)

func zlibBase64(t *testing.T, size int) string {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecodeTMXValuesBoundsInflation(t *testing.T) {
	t.Parallel()
	bomb := zlibBase64(t, 1<<20)
	if _, ok := decodeTMXValues(bomb, "base64", "zlib", 2, 2); ok {
		t.Fatal("zlib bomb inflated past its declared layer size")
	}
	if _, ok := decodeTMXValues(bomb, "base64", "zlib", 1<<20, 1<<20); ok {
		t.Fatal("absurd declared layer size accepted")
	}
	if v, ok := decodeTMXValues(zlibBase64(t, 16), "base64", "zlib", 2, 2); !ok || len(v) != 4 {
		t.Fatalf("honest layer refused: %v %v", v, ok)
	}
}

// An infinite map may hold any number of chunks, each within the per-chunk bound, so the bound covers the whole map
// before any chunk inflates.
func TestTMXTileCapCoversTheWholeMap(t *testing.T) {
	t.Parallel()
	chunk := tmxChunk{Width: 1000, Height: 1000}
	within := []tmxLayer{{Data: tmxData{Chunks: []tmxChunk{chunk, chunk, chunk, chunk}}}}
	if !tmxWithinTileCap(within) {
		t.Fatal("four 1000x1000 chunks refused")
	}
	over := []tmxLayer{within[0], {Data: tmxData{Chunks: []tmxChunk{chunk}}}}
	if tmxWithinTileCap(over) {
		t.Fatal("five 1000x1000 chunks across two layers accepted")
	}
	if tmxWithinTileCap([]tmxLayer{{Width: -1, Height: 10}}) {
		t.Fatal("a negative width accepted")
	}
}
