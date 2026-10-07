package archive

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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

func extract(t *testing.T, archive string, opts options) (string, error) {
	t.Helper()
	dest := t.TempDir()
	return dest, extractWith(archive, dest, opts)
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
			dest, err := extract(t, p, options{})
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
	dest, _ := extract(t, "testdata/valid.7z", options{})
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
		{"trailing dot", "mod/foo.", ErrUnsafeName},
		{"trailing space", "mod/foo ", ErrUnsafeName},
		{"invalid character", "mod/a?b", ErrUnsafeName},
		{"console device", "CONOUT$", ErrUnsafeName},
		{"drive letter", "C:/x", ErrUnsafeName},
	}
	for _, c := range cases {
		t.Run("zip "+c.name, func(t *testing.T) {
			_, err := extract(t, buildZip(t, zentry{name: c.entry, body: "x"}), options{})
			wantReason(t, err, c.reason, c.entry)
		})
		t.Run("rar "+c.name, func(t *testing.T) {
			_, err := extract(t, buildRar(t, rarFile{name: c.entry, data: "x"}), options{})
			wantReason(t, err, c.reason, c.entry)
		})
	}
	for _, p := range []string{"testdata/traversal.7z", "testdata/absolute.7z"} {
		_, err := extract(t, p, options{})
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
	err := Extract(buildZip(t, zentry{name: "../evil", body: "x"}), dest)
	if err == nil {
		t.Fatal("want error")
	}
	if _, err := os.Stat(filepath.Join(parent, "evil")); !os.IsNotExist(err) {
		t.Fatalf("evil written outside dest: %v", err)
	}
}

func TestLinksAndSpecialFilesRejected(t *testing.T) {
	_, err := extract(t, buildZip(t, zentry{name: "l", body: "target", mode: os.ModeSymlink | 0o777}), options{})
	wantReason(t, err, ErrLink, "l")
	_, err = extract(t, buildZip(t, zentry{name: "d", mode: os.ModeDevice | 0o666}), options{})
	wantReason(t, err, ErrSpecialFile, "d")
	_, err = extract(t, buildZip(t, zentry{name: "p", mode: os.ModeNamedPipe | 0o666}), options{})
	wantReason(t, err, ErrSpecialFile, "p")
	_, err = extract(t, "testdata/symlink.7z", options{})
	wantReason(t, err, ErrLink, "link")
	_, err = extract(t, buildRar(t, rarFile{name: "l", symlinkTo: "a"}), options{})
	wantReason(t, err, ErrLink, "l")
}

func TestCaseCollision(t *testing.T) {
	_, err := extract(t, buildZip(t, zentry{name: "Mod/a.txt", body: "1"}, zentry{name: "mod/b.txt", body: "2"}), options{})
	wantReason(t, err, ErrCaseCollision, "mod/b.txt")
	_, err = extract(t, buildZip(t, zentry{name: "a.txt", body: "1"}, zentry{name: "a.txt", body: "2"}), options{})
	wantReason(t, err, ErrCaseCollision, "a.txt")
	_, err = extract(t, buildRar(t, rarFile{name: "A", data: "1"}, rarFile{name: "a", data: "2"}), options{})
	wantReason(t, err, ErrCaseCollision, "a")
}

func TestCaps(t *testing.T) {
	big := string(make([]byte, 100))
	_, err := extract(t, buildZip(t, zentry{name: "big", body: big}), options{MaxEntryBytes: 99})
	wantReason(t, err, ErrEntryTooLarge, "big")
	if _, err := extract(t, buildZip(t, zentry{name: "big", body: big}), options{MaxEntryBytes: 100}); err != nil {
		t.Fatalf("entry at the cap: %v", err)
	}
	_, err = extract(t, buildZip(t, zentry{name: "a", body: big}, zentry{name: "b", body: big}), options{MaxTotalBytes: 150})
	wantReason(t, err, ErrArchiveTooLarge, "b")
	_, err = extract(t, buildZip(t, zentry{name: "a", body: "1"}, zentry{name: "b", body: "1"}, zentry{name: "c", body: "1"}), options{MaxEntries: 2})
	wantReason(t, err, ErrTooManyEntries, "")
	_, err = extract(t, buildRar(t, rarFile{name: "big", data: big}), options{MaxEntryBytes: 50})
	wantReason(t, err, ErrEntryTooLarge, "big")
	_, err = extract(t, "testdata/valid.7z", options{MaxEntryBytes: 5})
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
	_, err = extract(t, writeTemp(t, "lie.zip", b), options{MaxEntryBytes: 10})
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
	_, err = extract(t, writeTemp(t, "bad.zip", zb), options{})
	wantReason(t, err, ErrChecksum, "f")

	sb, err := os.ReadFile("testdata/valid.7z")
	if err != nil {
		t.Fatal(err)
	}
	sb[bytes.Index(sb, []byte("hello mortar"))] ^= 0xff
	_, err = extract(t, writeTemp(t, "bad.7z", sb), options{})
	wantReason(t, err, ErrChecksum, "a.txt")

	rb, err := os.ReadFile(buildRar(t, rarFile{name: "f", data: "payload"}))
	if err != nil {
		t.Fatal(err)
	}
	rb[bytes.Index(rb, []byte("payload"))] ^= 0xff
	_, err = extract(t, writeTemp(t, "bad.rar", rb), options{})
	wantReason(t, err, ErrChecksum, "f")
}

func TestEncryptedRejected(t *testing.T) {
	for _, p := range []string{"testdata/encrypted.7z", "testdata/encrypted-headers.7z"} {
		_, err := extract(t, p, options{})
		if !errors.Is(err, ErrEncrypted) {
			t.Fatalf("%s: got %v", p, err)
		}
	}
	_, err := extract(t, buildRar(t, rarFile{name: "f", data: "x", encrypted: true}), options{})
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
	_, err = extract(t, writeTemp(t, "enc.zip", buf.Bytes()), options{})
	wantReason(t, err, ErrEncrypted, "f")
}

func TestFormatDetectionIgnoresExtension(t *testing.T) {
	b, err := os.ReadFile("testdata/valid.7z")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := extract(t, writeTemp(t, "mod.zip", b), options{}); err != nil {
		t.Fatal(err)
	}
	_, err = extract(t, writeTemp(t, "mod.zip", []byte("not an archive")), options{})
	wantReason(t, err, ErrUnsupportedFormat, "")
}

func TestDiskFullIsDistinguishable(t *testing.T) {
	err := wrap("f", &os.PathError{Op: "write", Path: "f", Err: syscall.ENOSPC})
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatal("ENOSPC lost in the error chain")
	}
}

func TestDecoderMemoryHeadersRejected(t *testing.T) {
	cases := []struct {
		name    string
		archive string
		want    string
	}{
		{
			name:    "rar dictionary",
			archive: buildRar(t, rarFile{name: "x", data: string([]byte{0xc5, 0x05}), compressionInfo: 0x30c0}),
			want:    "dictionary too large",
		},
		{
			name:    "7z lzma2 dictionary",
			archive: buildSevenZipMemoryBomb(t, false),
			want:    "decoder memory exceeds the extraction cap",
		},
		{
			name:    "7z ppmd memory",
			archive: buildSevenZipMemoryBomb(t, true),
			want:    "decoder memory exceeds the extraction cap",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, err := extract(t, tc.archive, options{})
			runtime.ReadMemStats(&after)
			t.Logf("HeapSys delta=%d TotalAlloc delta=%d", after.HeapSys-before.HeapSys, after.TotalAlloc-before.TotalAlloc)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an archive error containing %q", err, tc.want)
			}
			if delta := after.TotalAlloc - before.TotalAlloc; delta >= uint64(DefaultMaxEntryBytes) {
				t.Fatalf("decoder allocated %d bytes before refusing the header", delta)
			}
		})
	}
}

func buildSevenZipMemoryBomb(t *testing.T, ppmd bool) string {
	t.Helper()
	var b []byte
	if ppmd {
		b = []byte{
			0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c, 0x00, 0x04, 0x97, 0xf7, 0xdf, 0xaa, 0x06, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x4a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x25, 0x3f, 0x36, 0xf1,
			0x00, 0x77, 0x88, 0x77, 0x88, 0x00, 0x01, 0x04, 0x06, 0x00, 0x01, 0x09, 0x06, 0x00, 0x07, 0x0b,
			0x01, 0x00, 0x01, 0x23, 0x03, 0x04, 0x01, 0x05, 0x10, 0x00, 0x00, 0x01, 0x00, 0x0c, 0x01, 0x00,
			0x08, 0x0a, 0x01, 0x83, 0x16, 0xdc, 0x8c, 0x00, 0x00, 0x05, 0x01, 0x19, 0x06, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x05, 0x00, 0x78, 0x00, 0x00, 0x00, 0x14, 0x0a,
			0x01, 0x00, 0x8b, 0xdc, 0xd6, 0x48, 0xdf, 0x51, 0xdd, 0x01, 0x15, 0x06, 0x01, 0x00, 0x20, 0x80,
			0xa4, 0x81, 0x00, 0x00,
		}
	} else {
		b = []byte{
			0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c, 0x00, 0x04, 0x1b, 0x48, 0x8c, 0xb3, 0x05, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x4a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0xc6, 0x27, 0x15,
			0x01, 0x00, 0x00, 0x78, 0x00, 0x01, 0x04, 0x06, 0x00, 0x01, 0x09, 0x05, 0x00, 0x07, 0x0b, 0x01,
			0x00, 0x01, 0x21, 0x21, 0x01, 0x00, 0x0c, 0x01, 0x00, 0x08, 0x0a, 0x01, 0x83, 0x16, 0xdc, 0x8c,
			0x00, 0x00, 0x05, 0x01, 0x19, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x11, 0x05, 0x00, 0x78, 0x00, 0x00, 0x00, 0x14, 0x0a, 0x01, 0x00, 0x55, 0x71, 0xd5,
			0x38, 0xdf, 0x51, 0xdd, 0x01, 0x15, 0x06, 0x01, 0x00, 0x20, 0x80, 0xa4, 0x81, 0x00, 0x00,
		}
	}
	headerSize := 0x4a
	headerOffset := len(b) - headerSize
	if ppmd {
		marker := []byte{0x23, 0x03, 0x04, 0x01, 0x05, 0x10, 0x00, 0x00, 0x01, 0x00}
		i := bytes.Index(b[headerOffset:], marker)
		if i < 0 {
			t.Fatal("PPMd coder properties not found")
		}
		binary.LittleEndian.PutUint32(b[headerOffset+i+6:], 512<<20)
	} else {
		marker := []byte{0x21, 0x21, 0x01, 0x00}
		i := bytes.Index(b[headerOffset:], marker)
		if i < 0 {
			t.Fatal("LZMA2 coder properties not found")
		}
		b[headerOffset+i+3] = 0x21
	}
	binary.LittleEndian.PutUint32(b[28:32], crc32.ChecksumIEEE(b[headerOffset:headerOffset+headerSize]))
	binary.LittleEndian.PutUint32(b[8:12], crc32.ChecksumIEEE(b[12:32]))
	return writeTemp(t, "memory-bomb.7z", b)
}

// rarFile is one entry of a hand-built RAR5 archive (stored, no compression).
type rarFile struct {
	name            string
	data            string
	compressionInfo uint64
	symlinkTo       string
	encrypted       bool
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
		c = append(c, vint(f.compressionInfo)...) // stored
		c = append(c, vint(1)...)                 // host OS: unix
		c = append(c, vint(uint64(len(f.name)))...)
		c = append(c, f.name...)
		c = append(c, extra...)
		out = append(out, rarBlock(c)...)
		out = append(out, f.data...)
	}
	out = append(out, rarBlock(append(vint(5), vint(0)...), 0x80, 0)...) // end of archive
	return writeTemp(t, "a.rar", out)
}

func TestZipNamesInALegacyEncodingBecomeUTF8(t *testing.T) {
	// GBK bytes for a left curly quote, as a Chinese-locale zip tool writes them without the UTF-8 flag.
	dest, err := extract(t, buildZip(t, zentry{name: "[FS]Kyuya\xa1\xaes hats Pack/manifest.json", body: "{}"}), options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dest, "[FS]Kyuya‘s hats Pack", "manifest.json")); got != "{}" {
		t.Fatalf("manifest = %q", got)
	}
}

func TestRepairNamesRenamesLegacyEncodedEntries(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Pack\xa1\xae")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "note\xa1\xae.txt", "x")
	n, err := RepairNames(root)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("renamed %d, want 2", n)
	}
	if got := readFile(t, filepath.Join(root, "Pack‘", "note‘.txt")); got != "x" {
		t.Fatalf("file = %q", got)
	}
	if n, err := RepairNames(root); err != nil || n != 0 {
		t.Fatalf("second pass renamed %d, %v", n, err)
	}
}

func TestStreamFormatsExtractATarByMagicBytes(t *testing.T) {
	for _, name := range []string{"valid.tar", "valid.tar.gz", "valid.tar.xz", "valid.tar.zst", "valid.tar.bz2", "valid.tar.lzma"} {
		t.Run(name, func(t *testing.T) {
			// A copy with no extension: only the bytes may decide.
			src, err := fsx.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			dest, err := extract(t, writeTemp(t, "download", src), options{})
			if err != nil {
				t.Fatal(err)
			}
			if got := readFile(t, filepath.Join(dest, "Mod", "Fix.dll")); got != "dll" {
				t.Fatalf("Fix.dll = %q", got)
			}
			if _, err := PreviewArchive(writeTemp(t, "download", src)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStreamFormatsExtractABareFileNamedWithoutItsSuffix(t *testing.T) {
	for _, name := range []string{"bare.txt.gz", "bare.txt.xz", "bare.txt.zst", "bare.txt.bz2", "bare.txt.lzma"} {
		t.Run(name, func(t *testing.T) {
			dest, err := extract(t, filepath.Join("testdata", name), options{})
			if err != nil {
				t.Fatal(err)
			}
			if got := readFile(t, filepath.Join(dest, "bare.txt")); got != "hello archive\n" {
				t.Fatalf("bare.txt = %q", got)
			}
			if n, err := DeclaredSize(filepath.Join("testdata", name)); err != nil || n != 14 {
				t.Fatalf("DeclaredSize = %d, %v", n, err)
			}
		})
	}
}

func TestStreamFormatsRefuseUnsafeEntries(t *testing.T) {
	for file, reason := range map[string]error{
		"traversal.tar.gz": ErrTraversal, "symlink.tar.gz": ErrLink, "hardlink.tar.gz": ErrLink, "fifo.tar.gz": ErrSpecialFile,
	} {
		t.Run(file, func(t *testing.T) {
			_, err := extract(t, filepath.Join("testdata", file), options{})
			if !errors.Is(err, reason) {
				t.Fatalf("got %v, want %v", err, reason)
			}
		})
	}
}

func TestStreamFormatsHonourTheSizeAndEntryCaps(t *testing.T) {
	src := filepath.Join("testdata", "valid.tar.xz")
	_, err := extract(t, src, options{MaxEntryBytes: 2})
	if !errors.Is(err, ErrEntryTooLarge) {
		t.Fatalf("got %v", err)
	}
	if _, err = extract(t, src, options{MaxEntries: 1}); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("got %v", err)
	}
	_, err = extract(t, filepath.Join("testdata", "bare.txt.gz"), options{MaxEntryBytes: 3})
	wantReason(t, err, ErrEntryTooLarge, "bare.txt")
}

// A gzip of zeros unpacks to far more than the cap; the stream's own budget stops it before the disk fills.
func TestStreamBombIsStoppedByTheDecompressionBudget(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(make([]byte, 8<<20)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := extract(t, writeTemp(t, "bomb.gz", buf.Bytes()), options{MaxEntryBytes: 1 << 20, MaxTotalBytes: 1 << 20, MaxEntries: 10})
	if !errors.Is(err, ErrEntryTooLarge) && !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestExtensionsStripLongestEndingFirst(t *testing.T) {
	for in, want := range map[string]string{
		"Mod.TAR.GZ": "Mod", "mod.tgz": "mod", "mod.zip": "mod", "notes.txt.xz": "notes.txt", "plain": "plain", "mod.tar.zst": "mod",
	} {
		if got := StripExtension(in); got != want {
			t.Errorf("StripExtension(%q) = %q, want %q", in, got, want)
		}
	}
	if HasExtension("readme.txt") || !HasExtension("a.tar.bz2") {
		t.Error("HasExtension wrong")
	}
	if !strings.Contains(PickerPattern(), "*.tar.xz;") {
		t.Error(PickerPattern())
	}
}
