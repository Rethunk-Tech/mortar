package share

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"testing"
)

func TestRewritesDropChangedCopiesReadingSkips(t *testing.T) {
	t.Parallel()
	dir, p := changedProfile(t)
	var buf bytes.Buffer
	if _, err := Write(&buf, "sims4", p, filepath.Join(dir, "mods"), OwnInclude()); err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		w, _ := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate})
		r, _ := f.Open()
		_, _ = io.Copy(w, io.LimitReader(r, 1<<20))
		_ = r.Close()
	}
	zeros := make([]byte, 5<<20)
	for i := range 8 {
		w, _ := zw.Create(fmt.Sprintf("%sMods/mc/pic%d.png", changedPrefix, i))
		_, _ = w.Write(zeros)
	}
	_ = zw.Close()
	pv, err := ReadBytes(out.Bytes())
	if err != nil {
		t.Fatalf("ReadBytes refused: %v", err)
	}
	kept, err := WithRefs(out.Bytes(), pv.Entries, nil)
	if err != nil {
		t.Fatal(err)
	}
	zk, _ := zip.NewReader(bytes.NewReader(kept), int64(len(kept)))
	var inflated uint64
	for _, f := range zk.File {
		inflated += f.UncompressedSize64
	}
	if inflated > MaxConfigTotal {
		t.Fatal("entries ReadBytes skipped were inflated past every cap")
	}
}
