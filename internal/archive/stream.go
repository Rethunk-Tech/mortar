package archive

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
)

// headLen is how much of a file's start detect reads: enough for the "ustar" marker at offset 257 of a tar header.
const headLen = 262

// dictLimit caps a decompressor's window, so a header cannot make it allocate more than an entry may extract to.
const dictLimit = DefaultMaxEntryBytes

// tarOverhead is what a tar adds around each entry's bytes (header, padding), counted into the stream's budget so
// skipped or unread data cannot decompress without bound.
const tarOverhead = 1536

// looksLikeLZMA recognises the classic .lzma header, which has no magic: valid properties, a dictionary size an
// encoder would write (a power of two, or one and a half of one) and an unknown or sane uncompressed size.
func looksLikeLZMA(head []byte) bool {
	if len(head) < lzma.HeaderLen || head[0] >= 225 {
		return false
	}
	dict := binary.LittleEndian.Uint32(head[1:5])
	pow2 := func(v uint32) bool { return v != 0 && v&(v-1) == 0 }
	if dict < 1<<12 || (!pow2(dict) && (dict%3 != 0 || !pow2(dict/3))) {
		return false
	}
	size := binary.LittleEndian.Uint64(head[5:13])
	return size == math.MaxUint64 || size < 1<<50
}

func detectStream(head []byte) int {
	switch {
	case bytes.HasPrefix(head, []byte{0x1f, 0x8b}):
		return fmtGzip
	case bytes.HasPrefix(head, []byte{0xfd, '7', 'z', 'X', 'Z', 0x00}):
		return fmtXz
	case bytes.HasPrefix(head, []byte{0x28, 0xb5, 0x2f, 0xfd}):
		return fmtZstd
	case len(head) >= 4 && bytes.HasPrefix(head, []byte("BZh")) && head[3] >= '1' && head[3] <= '9':
		return fmtBzip2
	case isTarHeader(head):
		return fmtTar
	case looksLikeLZMA(head):
		return fmtLzma
	}
	return fmtNone
}

func isTarHeader(head []byte) bool {
	return len(head) >= 262 && bytes.Equal(head[257:262], []byte("ustar"))
}

// capReader fails with ErrArchiveTooLarge once more than its budget has been read, whatever the stream claims.
type capReader struct {
	r    io.Reader
	left int64
}

func (c *capReader) Read(p []byte) (int, error) {
	if c.left <= 0 {
		var one [1]byte
		if n, err := c.r.Read(one[:]); n > 0 {
			return 0, &Error{Reason: ErrArchiveTooLarge}
		} else if err != nil {
			return 0, err
		}
		return 0, nil
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}

// payload is the decompressed bytes of a stream format, positioned at the start, and whether they are a tar.
type payload struct {
	r     *bufio.Reader
	isTar bool
	close func()
}

// openPayload decompresses r (already at the file's start) as kind, within the extraction caps.
func openPayload(r io.Reader, kind int, opts options) (*payload, error) {
	closeFn := func() {}
	var plain io.Reader
	switch kind {
	case fmtTar:
		plain = r
	case fmtGzip:
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, wrap("", err)
		}
		plain, closeFn = zr, func() { _ = zr.Close() }
	case fmtXz:
		xr, err := xz.ReaderConfig{DictCap: int(dictLimit)}.NewReader(r)
		if err != nil {
			return nil, wrap("", err)
		}
		plain = xr
	case fmtLzma:
		lr, err := lzma.ReaderConfig{DictCap: int(dictLimit)}.NewReader(r)
		if err != nil {
			return nil, wrap("", err)
		}
		plain = lr
	case fmtZstd:
		zr, err := zstd.NewReader(r,
			zstd.WithDecoderConcurrency(1),
			zstd.WithDecoderMaxMemory(uint64(dictLimit)),
			zstd.WithDecoderMaxWindow(uint64(dictLimit)))
		if err != nil {
			return nil, wrap("", err)
		}
		plain, closeFn = zr, zr.Close
	case fmtBzip2:
		plain = bzip2.NewReader(r)
	}
	budget := opts.MaxTotalBytes + int64(opts.MaxEntries)*tarOverhead + 1<<20
	br := bufio.NewReaderSize(&capReader{r: plain, left: budget}, 4096)
	head, err := br.Peek(512)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) && len(head) == 0 {
		closeFn()
		return nil, wrap("", err)
	}
	return &payload{r: br, isTar: kind == fmtTar || isTarHeader(head), close: closeFn}, nil
}

// singleName is the name a bare compressed file extracts under: the archive's own, without its suffix.
func singleName(archivePath string) string {
	base := filepath.Base(archivePath)
	if _, original, ok := strings.Cut(base, nameSep); ok {
		base = original
	}
	name := StripExtension(base)
	if name == "" || name == "." {
		return "file"
	}
	return name
}

// tarEntry validates one tar header's kind and returns whether it is a directory; skip is true for the pax global
// header, which names no file.
func tarKind(h *tar.Header) (isDir, skip bool, err error) {
	switch h.Typeflag {
	case tar.TypeReg, tar.TypeCont:
		return false, false, nil
	case tar.TypeDir:
		return true, false, nil
	case tar.TypeXGlobalHeader:
		return false, true, nil
	case tar.TypeSymlink, tar.TypeLink:
		return false, false, &Error{Entry: h.Name, Reason: ErrLink}
	}
	return false, false, &Error{Entry: h.Name, Reason: ErrSpecialFile}
}

func (x *extractor) stream(r io.Reader, kind int, archivePath string) error {
	p, err := openPayload(r, kind, x.opts)
	if err != nil {
		return err
	}
	defer p.close()
	if !p.isTar {
		open := func() (io.ReadCloser, error) { return io.NopCloser(p.r), nil }
		return x.entry(singleName(archivePath), false, open, nil)
	}
	tr := tar.NewReader(p.r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return wrap("", err)
		}
		isDir, skip, err := tarKind(h)
		if err != nil {
			return err
		}
		if skip {
			continue
		}
		open := func() (io.ReadCloser, error) { return io.NopCloser(tr), nil }
		if err := x.entry(h.Name, isDir, open, nil); err != nil {
			return err
		}
	}
}

func (v *previewer) stream(r io.Reader, kind int, archivePath string) error {
	p, err := openPayload(r, kind, options{MaxTotalBytes: DefaultMaxTotalBytes, MaxEntries: DefaultMaxEntries})
	if err != nil {
		return err
	}
	defer p.close()
	if !p.isTar {
		n, err := io.Copy(io.Discard, p.r)
		if err != nil {
			return wrap("", err)
		}
		v.add(singleName(archivePath), n, false, func() (io.ReadCloser, error) { return nil, io.EOF })
		return nil
	}
	tr := tar.NewReader(p.r)
	for !v.full() {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return wrap("", err)
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		v.add(h.Name, max(h.Size, 0), h.Typeflag == tar.TypeDir, func() (io.ReadCloser, error) { return io.NopCloser(tr), nil })
	}
	return nil
}

// streamSize is the bytes a stream format unpacks to, counted by reading it within the caps; a stream over them
// reports the cap.
func streamSize(r io.Reader, kind int) (int64, error) {
	p, err := openPayload(r, kind, options{MaxTotalBytes: DefaultMaxTotalBytes, MaxEntries: DefaultMaxEntries})
	if err != nil {
		return 0, err
	}
	defer p.close()
	n, err := io.Copy(io.Discard, p.r)
	if _, typed := errors.AsType[*Error](err); typed {
		return DefaultMaxTotalBytes, nil
	}
	return n, err
}
