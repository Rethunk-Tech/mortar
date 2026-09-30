package archive

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// zentry is one zip entry; a name ending in "/" is a directory, a zero mode a regular file.
type zentry struct {
	name string
	body string
	mode os.FileMode
}

func buildZip(t *testing.T, entries ...zentry) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Store}
		if e.mode != 0 {
			h.SetMode(e.mode)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return writeTemp(t, "a.zip", buf.Bytes())
}

func writeTemp(t *testing.T, name string, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := fsx.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func extract(t *testing.T, archive string, opts Options) (string, error) {
	t.Helper()
	dest := t.TempDir()
	return dest, Extract(archive, dest, opts)
}

func wantReason(t *testing.T, err, reason error, entry string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || !errors.Is(err, reason) {
		t.Fatalf("got %v, want *Error with reason %v", err, reason)
	}
	if e.Entry != entry {
		t.Fatalf("entry = %q, want %q", e.Entry, entry)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := fsx.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestValidExtractsAndNormalisesModes(t *testing.T) {
	zipPath := buildZip(t,
		zentry{name: "mod/", mode: os.ModeDir | 0o700},
		zentry{name: "mod/run.sh", body: "x", mode: 0o4755},
		zentry{name: "mod/sub/data.json", body: "{}"},
	)
	cases := map[string]string{
		"zip":  zipPath,
		"7z":   "testdata/valid.7z",
		"rar":  "testdata/real.rar",
		"rar5": buildRar(t, rarFile{name: "sub/a.txt", data: "hello"}),
	}
	wantFile := map[string]string{"zip": "mod/sub/data.json", "7z": "sub/b.txt", "rar": "asd.go", "rar5": "sub/a.txt"}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			dest, err := extract(t, p, Options{})
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dest, filepath.FromSlash(wantFile[name]))
			info, err := os.Stat(target)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o600 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
				t.Fatalf("file mode = %v", info.Mode())
			}
			dir, err := os.Stat(filepath.Dir(target))
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Dir(target) != dest && dir.Mode().Perm() != 0o750 {
				t.Fatalf("dir mode = %v", dir.Mode())
			}
		})
	}
	dest, _ := extract(t, "testdata/valid.7z", Options{})
	if got := readFile(t, filepath.Join(dest, "a.txt")); got != "hello mortar\n" {
		t.Fatalf("7z content = %q", got)
	}
}

func TestUnsafeEntriesRejected(t *testing.T) {
	cases := []struct {
		name   string
		entry  string
		reason error
	}{
		{"traversal", "../evil", ErrTraversal},
		{"nested traversal", "a/../../evil", ErrTraversal},
		{"backslash traversal", `..\evil`, ErrTraversal},
		{"absolute", "/etc/evil", ErrTraversal},
		{"reserved", "mod/CON", ErrUnsafeName},
		{"reserved with extension", "nul.txt", ErrUnsafeName},
		{"reserved dir", "com1/x", ErrUnsafeName},
		{"lpt9", "LPT9.dll", ErrUnsafeName},
		{"colon", "a:b", ErrUnsafeName},
		{"drive letter", "C:/x", ErrUnsafeName},
	}
	for _, c := range cases {
		t.Run("zip "+c.name, func(t *testing.T) {
			_, err := extract(t, buildZip(t, zentry{name: c.entry, body: "x"}), Options{})
			wantReason(t, err, c.reason, c.entry)
		})
		t.Run("rar "+c.name, func(t *testing.T) {
			_, err := extract(t, buildRar(t, rarFile{name: c.entry, data: "x"}), Options{})
			wantReason(t, err, c.reason, c.entry)
		})
	}
	for _, p := range []string{"testdata/traversal.7z", "testdata/absolute.7z"} {
		_, err := extract(t, p, Options{})
		var e *Error
		if !errors.As(err, &e) || !errors.Is(err, ErrTraversal) {
			t.Fatalf("%s: got %v", p, err)
		}
	}
}

func TestNothingWrittenOutsideDest(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "dest")
	if err := os.Mkdir(dest, 0o750); err != nil {
		t.Fatal(err)
	}
	err := Extract(buildZip(t, zentry{name: "../evil", body: "x"}), dest, Options{})
	if err == nil {
		t.Fatal("want error")
	}
	if _, err := os.Stat(filepath.Join(parent, "evil")); !os.IsNotExist(err) {
		t.Fatalf("evil written outside dest: %v", err)
	}
}

func TestLinksAndSpecialFilesRejected(t *testing.T) {
	_, err := extract(t, buildZip(t, zentry{name: "l", body: "target", mode: os.ModeSymlink | 0o777}), Options{})
	wantReason(t, err, ErrLink, "l")
	_, err = extract(t, buildZip(t, zentry{name: "d", mode: os.ModeDevice | 0o666}), Options{})
	wantReason(t, err, ErrSpecialFile, "d")
	_, err = extract(t, buildZip(t, zentry{name: "p", mode: os.ModeNamedPipe | 0o666}), Options{})
	wantReason(t, err, ErrSpecialFile, "p")
	_, err = extract(t, "testdata/symlink.7z", Options{})
	wantReason(t, err, ErrLink, "link")
	_, err = extract(t, buildRar(t, rarFile{name: "l", symlinkTo: "a"}), Options{})
	wantReason(t, err, ErrLink, "l")
}

func TestCaseCollision(t *testing.T) {
	_, err := extract(t, buildZip(t, zentry{name: "Mod/a.txt", body: "1"}, zentry{name: "mod/b.txt", body: "2"}), Options{})
	wantReason(t, err, ErrCaseCollision, "mod/b.txt")
	_, err = extract(t, buildZip(t, zentry{name: "a.txt", body: "1"}, zentry{name: "a.txt", body: "2"}), Options{})
	wantReason(t, err, ErrCaseCollision, "a.txt")
	_, err = extract(t, buildRar(t, rarFile{name: "A", data: "1"}, rarFile{name: "a", data: "2"}), Options{})
	wantReason(t, err, ErrCaseCollision, "a")
}

func TestCaps(t *testing.T) {
	big := string(make([]byte, 100))
	_, err := extract(t, buildZip(t, zentry{name: "big", body: big}), Options{MaxEntryBytes: 99})
	wantReason(t, err, ErrEntryTooLarge, "big")
	if _, err := extract(t, buildZip(t, zentry{name: "big", body: big}), Options{MaxEntryBytes: 100}); err != nil {
		t.Fatalf("entry at the cap: %v", err)
	}
	_, err = extract(t, buildZip(t, zentry{name: "a", body: big}, zentry{name: "b", body: big}), Options{MaxTotalBytes: 150})
	wantReason(t, err, ErrArchiveTooLarge, "b")
	_, err = extract(t, buildZip(t, zentry{name: "a", body: "1"}, zentry{name: "b", body: "1"}, zentry{name: "c", body: "1"}), Options{MaxEntries: 2})
	wantReason(t, err, ErrTooManyEntries, "")
	_, err = extract(t, buildRar(t, rarFile{name: "big", data: big}), Options{MaxEntryBytes: 50})
	wantReason(t, err, ErrEntryTooLarge, "big")
	_, err = extract(t, "testdata/valid.7z", Options{MaxEntryBytes: 5})
	wantReason(t, err, ErrEntryTooLarge, "a.txt")
}

func TestCapCountsActualBytesNotDeclared(t *testing.T) {
	// Declare 1 byte in the local header and central directory, store 100.
	p := buildZip(t, zentry{name: "big", body: string(make([]byte, 100))})
	b, err := fsx.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range [][]byte{[]byte("PK\x03\x04"), []byte("PK\x01\x02")} {
		i := bytes.Index(b, sig)
		off := i + 22 // uncompressed size in both header kinds
		if sig[2] == 1 {
			off = i + 24
		}
		binary.LittleEndian.PutUint32(b[off:], 1)
	}
	_, err = extract(t, writeTemp(t, "lie.zip", b), Options{MaxEntryBytes: 10})
	if _, ok := errors.AsType[*Error](err); !ok {
		t.Fatalf("got %v", err)
	}
}

func TestChecksumMismatch(t *testing.T) {
	zb, err := os.ReadFile(buildZip(t, zentry{name: "f", body: "payload"}))
	if err != nil {
		t.Fatal(err)
	}
	zb[bytes.Index(zb, []byte("payload"))] ^= 0xff
	_, err = extract(t, writeTemp(t, "bad.zip", zb), Options{})
	wantReason(t, err, ErrChecksum, "f")

	sb, err := os.ReadFile("testdata/valid.7z")
	if err != nil {
		t.Fatal(err)
	}
	sb[bytes.Index(sb, []byte("hello mortar"))] ^= 0xff
	_, err = extract(t, writeTemp(t, "bad.7z", sb), Options{})
	wantReason(t, err, ErrChecksum, "a.txt")

	rb, err := os.ReadFile(buildRar(t, rarFile{name: "f", data: "payload"}))
	if err != nil {
		t.Fatal(err)
	}
	rb[bytes.Index(rb, []byte("payload"))] ^= 0xff
	_, err = extract(t, writeTemp(t, "bad.rar", rb), Options{})
	wantReason(t, err, ErrChecksum, "f")
}

func TestEncryptedRejected(t *testing.T) {
	for _, p := range []string{"testdata/encrypted.7z", "testdata/encrypted-headers.7z"} {
		_, err := extract(t, p, Options{})
		if !errors.Is(err, ErrEncrypted) {
			t.Fatalf("%s: got %v", p, err)
		}
	}
	_, err := extract(t, buildRar(t, rarFile{name: "f", data: "x", encrypted: true}), Options{})
	wantReason(t, err, ErrEncrypted, "f")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "f", Method: zip.Store, Flags: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("x"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = extract(t, writeTemp(t, "enc.zip", buf.Bytes()), Options{})
	wantReason(t, err, ErrEncrypted, "f")
}

func TestFormatDetectionIgnoresExtension(t *testing.T) {
	b, err := os.ReadFile("testdata/valid.7z")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := extract(t, writeTemp(t, "mod.zip", b), Options{}); err != nil {
		t.Fatal(err)
	}
	_, err = extract(t, writeTemp(t, "mod.zip", []byte("not an archive")), Options{})
	wantReason(t, err, ErrUnsupportedFormat, "")
}

func TestDiskFullIsDistinguishable(t *testing.T) {
	err := wrap("f", &os.PathError{Op: "write", Path: "f", Err: syscall.ENOSPC})
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatal("ENOSPC lost in the error chain")
	}
}

// rarFile is one entry of a hand-built RAR5 archive (stored, no compression).
type rarFile struct {
	name      string
	data      string
	symlinkTo string
	encrypted bool
}

func vint(n uint64) []byte {
	var out []byte
	for n >= 0x80 {
		out = append(out, byte(n)|0x80)
		n >>= 7
	}
	return append(out, byte(n))
}

// rarBlock frames a RAR5 header: CRC32 over the size field and the content.
func rarBlock(content []byte, pad ...byte) []byte {
	content = append(content, pad...) // the reader wants at least 4 header bytes; a padded zero vint is valid
	body := append(vint(uint64(len(content))), content...)
	crc := binary.LittleEndian.AppendUint32(nil, crc32.ChecksumIEEE(body))
	return append(crc, body...)
}

// buildRar hand-writes a RAR5 archive because no RAR-creating tool is
// installed; testdata/real.rar covers a real tool's output.
func buildRar(t *testing.T, files ...rarFile) string {
	t.Helper()
	out := []byte("Rar!\x1a\x07\x01\x00")
	out = append(out, rarBlock(append(vint(1), vint(0)...), 0x80, 0)...) // main header
	for _, f := range files {
		var extra []byte
		if f.symlinkTo != "" {
			rec := append(vint(5), vint(1)...) // redirection, unix symlink
			rec = append(rec, vint(0)...)
			rec = append(rec, vint(uint64(len(f.symlinkTo)))...)
			rec = append(rec, f.symlinkTo...)
			extra = append(vint(uint64(len(rec))), rec...)
		}
		if f.encrypted {
			rec := append(vint(1), vint(0)...) // encryption, version 0
			rec = append(rec, vint(0)...)
			rec = append(rec, 15)
			rec = append(rec, make([]byte, 32)...)
			extra = append(vint(uint64(len(rec))), rec...)
		}
		flags := uint64(0x02) // data area present
		if extra != nil {
			flags |= 0x01
		}
		c := append(vint(2), vint(flags)...)
		if extra != nil {
			c = append(c, vint(uint64(len(extra)))...)
		}
		c = append(c, vint(uint64(len(f.data)))...) // packed size
		c = append(c, vint(0x04)...)                // crc32 present
		c = append(c, vint(uint64(len(f.data)))...) // unpacked size
		c = append(c, vint(0o644)...)               // attributes
		crc := make([]byte, 4)
		binary.LittleEndian.PutUint32(crc, crc32.ChecksumIEEE([]byte(f.data)))
		c = append(c, crc...)
		c = append(c, vint(0)...) // stored
		c = append(c, vint(1)...) // host OS: unix
		c = append(c, vint(uint64(len(f.name)))...)
		c = append(c, f.name...)
		c = append(c, extra...)
		out = append(out, rarBlock(c)...)
		out = append(out, f.data...)
	}
	out = append(out, rarBlock(append(vint(5), vint(0)...), 0x80, 0)...) // end of archive
	return writeTemp(t, "a.rar", out)
}
