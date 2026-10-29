// Package archive is Mortar's face on github.com/Rethunk-Tech/go-archive, which reads every untrusted archive: it
// opens files by path, applies Windows' name rules to extraction, and previews an archive's contents for the UI.
package archive

import (
	"bytes"
	"io"
	"os"

	ga "github.com/Rethunk-Tech/go-archive"

	"github.com/Rethunk-Tech/go-safefs"
)

type (
	// Error names the entry that failed and why.
	Error = ga.Error
	// Entry is one archive entry.
	Entry = ga.Entry
)

const (
	DefaultMaxEntryBytes = ga.DefaultMaxEntryBytes
	DefaultMaxTotalBytes = ga.DefaultMaxTotalBytes
	DefaultMaxEntries    = ga.DefaultMaxEntries
)

var (
	ErrUnsupportedFormat = ga.ErrUnsupportedFormat
	ErrEncrypted         = ga.ErrEncrypted
	ErrTraversal         = ga.ErrTraversal
	ErrUnsafeName        = ga.ErrUnsafeName
	ErrCaseCollision     = ga.ErrCaseCollision
	ErrLink              = ga.ErrLink
	ErrSpecialFile       = ga.ErrSpecialFile
	ErrEntryTooLarge     = ga.ErrEntryTooLarge
	ErrArchiveTooLarge   = ga.ErrArchiveTooLarge
	ErrTooManyEntries    = ga.ErrTooManyEntries
	ErrRatio             = ga.ErrRatio
	ErrChecksum          = ga.ErrChecksum
	ErrMalformed         = ga.ErrMalformed
	ErrDecoderMemory     = ga.ErrDecoderMemory
)

// RecoverMalformed turns a panic in a format reader into ErrMalformed for *err.
func RecoverMalformed(err *error) { ga.RecoverMalformed(err) }

// Extract unpacks the archive at archivePath into dest, which must exist. It stops at the first failure and leaves
// what it wrote, so the caller passes a temp directory and discards it on error.
func Extract(archivePath, dest string) error {
	f, size, err := openFile(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return ga.Extract(f, size, originalName(archivePath), root, ga.Options{})
}

// Options lowers or raises extraction's caps; a zero field takes go-archive's default.
type Options = ga.Options

// ExtractReader is Extract for a stream with no random access, such as a network body: tar, RAR and the compressed
// streams. A body cut short fails as io.ErrUnexpectedEOF.
func ExtractReader(r io.Reader, baseName string, dest *os.Root, opts Options) error {
	return ga.ExtractReader(r, baseName, dest, opts)
}

// ZipFunc hands each entry of the zip src (size bytes) to fn under go-archive's admission rules, writing nothing.
// Anything but a zip is ErrUnsupportedFormat, since a format whose contract is "a zip" must not take a tarball
// because the bytes allow one. Every segment Windows refuses is admitted: each caller judges the names by its own
// rules once it has them.
func ZipFunc(src io.ReaderAt, size int64, opts Options, fn func(name string, isDir bool, r io.Reader) error) error {
	var magic [4]byte
	if _, err := src.ReadAt(magic[:], 0); err != nil || (string(magic[:]) != "PK\x03\x04" && string(magic[:]) != "PK\x05\x06") {
		return &Error{Reason: ErrUnsupportedFormat}
	}
	opts.Segment = func(string) bool { return true }
	return ga.ExtractFunc(src, size, "", opts, fn)
}

// DeclaredSize sums the uncompressed sizes the archive's headers declare. It is informational: an untrusted
// archive can lie.
func DeclaredSize(archivePath string) (int64, error) {
	f, size, err := openFile(archivePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	return ga.DeclaredSize(f, size)
}

func openFile(archivePath string) (*os.File, int64, error) {
	f, err := safefs.Open(archivePath)
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

// Listing walks an archive's entries forward-only; Close releases the decoders and the file.
type Listing struct {
	*ga.Lister
	f *os.File
}

// List opens the archive at archivePath for a forward walk, bounding what one entry may decompress to maxEntryBytes
// (zero takes the module's default).
func List(archivePath string, maxEntryBytes int64) (*Listing, error) {
	f, size, err := openFile(archivePath)
	if err != nil {
		return nil, err
	}
	return newListing(f, size, archivePath, maxEntryBytes)
}

// ListBytes is List for an archive already in memory.
func ListBytes(raw []byte, name string, maxEntryBytes int64) (*Listing, error) {
	l, err := ga.NewLister(bytes.NewReader(raw), int64(len(raw)), originalName(name), ga.ListerOptions{MaxEntryBytes: maxEntryBytes})
	if err != nil {
		return nil, err
	}
	return &Listing{Lister: l}, nil
}

func newListing(f *os.File, size int64, name string, maxEntryBytes int64) (*Listing, error) {
	l, err := ga.NewLister(f, size, originalName(name), ga.ListerOptions{MaxEntryBytes: maxEntryBytes})
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Listing{Lister: l, f: f}, nil
}

// Close releases the walk and the file.
func (l *Listing) Close() error {
	err := l.Lister.Close()
	if l.f != nil {
		if cerr := l.f.Close(); err == nil {
			err = cerr
		}
	}
	return err
}
