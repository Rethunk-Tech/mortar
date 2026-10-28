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
	bomb := zlibBase64(t, 64<<20)
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
