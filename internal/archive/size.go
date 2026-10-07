package archive

import (
	"archive/zip"
	"errors"
	"io"
	"math"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/nwaples/rardecode/v2"
)

// DeclaredSize sums the uncompressed sizes the archive's headers declare. It
// is informational (an untrusted archive can lie); Extract's caps count bytes.
func DeclaredSize(archivePath string) (_ int64, err error) {
	defer RecoverMalformed(&err)
	f, err := fsx.Open(archivePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	var magic [headLen]byte
	n, _ := io.ReadFull(f, magic[:])

	var total int64
	switch detect(magic[:n]) {
	case fmtZip:
		zr, err := zip.NewReader(f, info.Size())
		if err != nil && (zr == nil || !errors.Is(err, zip.ErrInsecurePath)) {
			return 0, wrap("", err)
		}
		for _, e := range zr.File {
			total += int64(min(e.UncompressedSize64, math.MaxInt64))
		}
	case fmtSevenZip:
		zr, err := OpenSevenZip(f, info.Size())
		if err != nil {
			return 0, wrap("", err)
		}
		for _, e := range zr.File {
			total += int64(min(e.UncompressedSize, math.MaxInt64))
		}
	case fmtRAR:
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		rr, err := rardecode.NewReader(f)
		if err != nil {
			return 0, wrap("", err)
		}
		for {
			h, err := rr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return 0, wrap("", err)
			}
			total += h.UnPackedSize
		}
	case fmtTar, fmtGzip, fmtXz, fmtLzma, fmtZstd, fmtBzip2:
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return streamSize(f, detect(magic[:n]))
	default:
		return 0, &Error{Reason: ErrUnsupportedFormat}
	}
	return total, nil
}
