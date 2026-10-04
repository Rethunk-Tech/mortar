// Package archive extracts zip, RAR and 7z archives from an untrusted source into a directory.
package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/winname"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

// Default caps, applied to every zero field of options.
const (
	DefaultMaxEntryBytes int64 = 256 << 20
	DefaultMaxTotalBytes int64 = 2 << 30
	DefaultMaxEntries          = 20000
)

// options lowers or raises the extraction caps; a zero field means its default.
type options struct {
	MaxEntryBytes int64
	MaxTotalBytes int64
	MaxEntries    int
}

// Reasons carried by Error. A disk-full write is reported as the underlying
// error, so errors.Is(err, syscall.ENOSPC) holds.
var (
	ErrUnsupportedFormat = errors.New("not a zip, RAR or 7z archive")
	ErrEncrypted         = errors.New("archive is encrypted")
	ErrTraversal         = errors.New("path escapes the destination")
	ErrUnsafeName        = errors.New("name is not allowed")
	ErrCaseCollision     = errors.New("name collides with another entry")
	ErrLink              = errors.New("symlinks and hardlinks are not allowed")
	ErrSpecialFile       = errors.New("device, pipe and socket entries are not allowed")
	ErrEntryTooLarge     = errors.New("entry exceeds the size cap")
	ErrArchiveTooLarge   = errors.New("archive exceeds the extracted size cap")
	ErrTooManyEntries    = errors.New("archive exceeds the entry cap")
	ErrChecksum          = errors.New("checksum mismatch")
)

// Error names the entry that failed and why. Entry is empty for a failure of
// the archive as a whole.
type Error struct {
	Entry  string
	Reason error
}

func (e *Error) Error() string {
	if e.Entry == "" {
		return "archive: " + e.Reason.Error()
	}
	return fmt.Sprintf("archive: entry %q: %v", e.Entry, e.Reason)
}

func (e *Error) Unwrap() error { return e.Reason }

// Extract unpacks the archive at archivePath into dest, detecting the format
// from its magic bytes. dest must exist. Extraction stops at the first
// failure and leaves what it wrote in dest, so the caller passes a temp
// directory and discards it on error.
func Extract(archivePath, dest string) error {
	return extractWith(archivePath, dest, options{})
}

// extractWith is Extract with lowered or raised caps, which the tests use.
func extractWith(archivePath, dest string, opts options) error {
	f, err := fsx.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}

	var magic [8]byte
	n, _ := io.ReadFull(f, magic[:])
	head := magic[:n]

	x := &extractor{dest: dest, opts: opts, seen: map[string]string{}, files: map[string]bool{}}
	if x.opts.MaxEntryBytes == 0 {
		x.opts.MaxEntryBytes = DefaultMaxEntryBytes
	}
	if x.opts.MaxTotalBytes == 0 {
		x.opts.MaxTotalBytes = DefaultMaxTotalBytes
	}
	if x.opts.MaxEntries == 0 {
		x.opts.MaxEntries = DefaultMaxEntries
	}

	switch detect(head) {
	case fmtZip:
		return x.zip(f, info.Size())
	case fmtRAR:
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return x.rar(f)
	case fmtSevenZip:
		return x.sevenZip(f, info.Size())
	}
	return &Error{Reason: ErrUnsupportedFormat}
}

const (
	fmtNone = iota
	fmtZip
	fmtRAR
	fmtSevenZip
)

func detect(head []byte) int {
	switch {
	case bytes.HasPrefix(head, []byte("PK\x03\x04")), bytes.HasPrefix(head, []byte("PK\x05\x06")):
		return fmtZip
	case bytes.HasPrefix(head, []byte("Rar!\x1a\x07\x00")), bytes.HasPrefix(head, []byte("Rar!\x1a\x07\x01\x00")):
		return fmtRAR
	case bytes.HasPrefix(head, []byte("7z\xbc\xaf\x27\x1c")):
		return fmtSevenZip
	}
	return fmtNone
}

type extractor struct {
	dest    string
	opts    options
	entries int
	total   int64
	seen    map[string]string // lower-cased path -> spelling first used
	files   map[string]bool
}

func (x *extractor) zip(r io.ReaderAt, size int64) error {
	zr, err := zip.NewReader(r, size)
	// The path check below is stricter than the reader's own.
	if err != nil && (zr == nil || !errors.Is(err, zip.ErrInsecurePath)) {
		return wrap("", err)
	}
	if len(zr.File) > x.opts.MaxEntries {
		return &Error{Reason: ErrTooManyEntries}
	}
	for _, f := range zr.File {
		f.Name = zipName(f)
		mode := f.Mode()
		if f.Flags&1 != 0 {
			return &Error{Entry: f.Name, Reason: ErrEncrypted}
		}
		if err := checkKind(f.Name, mode); err != nil {
			return err
		}
		if err := x.entry(f.Name, mode.IsDir(), func() (io.ReadCloser, error) { return f.Open() }, &f.CRC32); err != nil {
			return err
		}
	}
	return nil
}

// zipName is an entry's name as UTF-8. Zips written without the UTF-8 flag carry names in the writer's locale;
// for Stardew mods that is most often GBK from Chinese-locale tools, and GB18030 decodes it. A name left as raw
// bytes extracts to a folder the game's .NET runtime cannot open.
func zipName(f *zip.File) string { return utf8Name(f.Name) }

func utf8Name(name string) string {
	if utf8.ValidString(name) {
		return name
	}
	decoded, err := simplifiedchinese.GB18030.NewDecoder().String(name)
	if err != nil || !utf8.ValidString(decoded) {
		return strings.ToValidUTF8(name, "_")
	}
	return decoded
}

// RepairNames renames every file and folder under root whose name is not UTF-8, as zipName would have named it,
// deepest first so a folder's own path stays valid while its children move. A name whose repaired form already
// exists is left alone. It returns how many entries it renamed.
func RepairNames(root string) (int, error) {
	var bad []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if path != root && !utf8.ValidString(d.Name()) {
			bad = append(bad, path)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	renamed := 0
	for _, from := range slices.Backward(bad) {
		to := filepath.Join(filepath.Dir(from), utf8Name(filepath.Base(from)))
		if _, err := os.Lstat(to); err == nil {
			continue
		}
		if err := fsx.Rename(from, to); err != nil {
			return renamed, err
		}
		renamed++
	}
	return renamed, nil
}

func (x *extractor) sevenZip(r io.ReaderAt, size int64) error {
	if err := checkSevenZipLimits(r, size); err != nil {
		return wrap("", err)
	}
	zr, err := sevenzip.NewReader(r, size)
	if err != nil {
		return wrap("", err)
	}
	if len(zr.File) > x.opts.MaxEntries {
		return &Error{Reason: ErrTooManyEntries}
	}
	for _, f := range zr.File {
		mode := f.Mode()
		if err := checkKind(f.Name, mode); err != nil {
			return err
		}
		if err := x.entry(f.Name, mode.IsDir(), f.Open, &f.CRC32); err != nil {
			return err
		}
	}
	return nil
}

func (x *extractor) rar(r io.Reader) error {
	// A larger RAR window cannot produce an entry larger than the extraction cap.
	rr, err := rardecode.NewReader(r, rardecode.MaxDictionarySize(DefaultMaxEntryBytes))
	if err != nil {
		return wrap("", err)
	}
	for {
		h, err := rr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return wrap("", err)
		}
		if h.Encrypted || h.HeaderEncrypted {
			return &Error{Entry: h.Name, Reason: ErrEncrypted}
		}
		if h.LinkType != rardecode.LinkTypeNone {
			return &Error{Entry: h.Name, Reason: ErrLink}
		}
		mode := h.Mode()
		if h.HostOS == rardecode.HostOSUnix {
			// Mode() reports only the symlink type bit; catch devices and pipes too.
			switch h.Attributes & 0xF000 {
			case 0, 0x4000, 0x8000:
			case 0xA000:
				return &Error{Entry: h.Name, Reason: ErrLink}
			default:
				return &Error{Entry: h.Name, Reason: ErrSpecialFile}
			}
		}
		if err := checkKind(h.Name, mode); err != nil {
			return err
		}
		// The reader verifies the RAR checksum itself when the entry ends.
		open := func() (io.ReadCloser, error) { return io.NopCloser(rr), nil }
		if err := x.entry(h.Name, h.IsDir, open, nil); err != nil {
			return err
		}
	}
}

func checkKind(name string, mode fs.FileMode) error {
	switch {
	case mode&fs.ModeSymlink != 0:
		return &Error{Entry: name, Reason: ErrLink}
	case mode&(fs.ModeDevice|fs.ModeCharDevice|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeIrregular) != 0:
		return &Error{Entry: name, Reason: ErrSpecialFile}
	}
	return nil
}

// entry validates one entry's name and caps, then creates the directory or
// writes the file. wantCRC is the CRC32 the archive declares, when it has one.
func (x *extractor) entry(name string, isDir bool, open func() (io.ReadCloser, error), wantCRC *uint32) error {
	rel, err := x.admit(name, isDir)
	if err != nil {
		return err
	}
	if rel == "" {
		return nil
	}
	target := filepath.Join(x.dest, filepath.FromSlash(rel))
	if isDir {
		if err := os.MkdirAll(target, 0o750); err != nil {
			return wrap(name, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return wrap(name, err)
	}
	src, err := open()
	if err != nil {
		return wrap(name, err)
	}
	defer func() { _ = src.Close() }()
	return x.write(name, target, src, wantCRC)
}

func (x *extractor) write(name, target string, src io.Reader, wantCRC *uint32) error {
	// O_EXCL refuses to write through an existing symlink or over a duplicate.
	out, err := fsx.CreateExcl(target, 0o600)
	if err != nil {
		return wrap(name, err)
	}
	limit := min(x.opts.MaxEntryBytes, x.opts.MaxTotalBytes-x.total)
	h := crc32.NewIEEE()
	// One byte past the limit proves the entry is over it without trusting declared sizes.
	n, err := io.Copy(io.MultiWriter(out, h), io.LimitReader(src, limit+1))
	x.total += n
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return wrap(name, err)
	}
	if n > limit {
		if n > x.opts.MaxEntryBytes {
			return &Error{Entry: name, Reason: ErrEntryTooLarge}
		}
		return &Error{Entry: name, Reason: ErrArchiveTooLarge}
	}
	if wantCRC != nil && h.Sum32() != *wantCRC {
		return &Error{Entry: name, Reason: ErrChecksum}
	}
	return nil
}

// admit counts the entry and returns its clean slash-separated path, or ""
// for an entry naming the destination itself.
func (x *extractor) admit(name string, isDir bool) (string, error) {
	rel, err := cleanName(name)
	if err != nil {
		return "", &Error{Entry: name, Reason: err}
	}
	if rel == "" {
		return "", nil
	}
	x.entries++
	if x.entries > x.opts.MaxEntries {
		return "", &Error{Entry: name, Reason: ErrTooManyEntries}
	}
	segs := strings.Split(rel, "/")
	for i := range segs {
		p := strings.Join(segs[:i+1], "/")
		if prev, ok := x.seen[strings.ToLower(p)]; ok && prev != p {
			return "", &Error{Entry: name, Reason: ErrCaseCollision}
		}
		x.seen[strings.ToLower(p)] = p
	}
	if !isDir {
		if x.files[rel] {
			return "", &Error{Entry: name, Reason: ErrCaseCollision}
		}
		x.files[rel] = true
	}
	return rel, nil
}

func cleanName(name string) (string, error) {
	name = strings.ReplaceAll(name, `\`, "/")
	if strings.ContainsRune(name, 0) {
		return "", ErrUnsafeName
	}
	if strings.HasPrefix(name, "/") {
		return "", ErrTraversal
	}
	var segs []string
	for s := range strings.SplitSeq(name, "/") {
		switch s {
		case "", ".":
			continue
		case "..":
			return "", ErrTraversal
		}
		if !winname.Valid(s) {
			return "", ErrUnsafeName
		}
		segs = append(segs, s)
	}
	return path.Join(segs...), nil
}

// wrap types a library or filesystem error as an *Error. Filesystem errors
// stay in the chain, so errors.Is(err, syscall.ENOSPC) reports a full disk.
func wrap(entry string, err error) error {
	if _, typed := errors.AsType[*Error](err); typed {
		return err
	}
	var re *sevenzip.ReadError
	switch {
	case errors.Is(err, zip.ErrChecksum), errors.Is(err, rardecode.ErrBadFileChecksum):
		err = ErrChecksum
	case errors.As(err, &re) && re.Encrypted:
		err = ErrEncrypted
	}
	return &Error{Entry: entry, Reason: err}
}
