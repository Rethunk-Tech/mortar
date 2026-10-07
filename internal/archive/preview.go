package archive

import (
	"archive/zip"
	"errors"
	"io"
	"math"
	"path"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"

	"github.com/nwaples/rardecode/v2"
)

const (
	// MaxPreviewEntries caps the file tree handed to the UI; the totals still count every entry.
	MaxPreviewEntries   = 2000
	maxPreviewManifests = 100
	maxManifestBytes    = 1 << 20
)

// PreviewEntry is one file or folder in an archive listing.
type PreviewEntry struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}

// PreviewManifest is a SMAPI manifest found in an archive.
type PreviewManifest struct {
	Folder  string `json:"folder"`
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Preview is what an archive holds, read from its listing without extracting anything.
type Preview struct {
	Entries   []PreviewEntry    `json:"entries"`
	Manifests []PreviewManifest `json:"manifests"`
	// Fomod is true when the archive carries a FOMOD installer script.
	Fomod     bool  `json:"fomod"`
	TotalSize int64 `json:"totalSize"`
	// Truncated is true when Entries was cut at MaxPreviewEntries or the archive has more than DefaultMaxEntries.
	Truncated bool `json:"truncated"`
}

type previewer struct {
	p     Preview
	count int
}

// add records one entry; open is called only for a manifest, so file bodies are otherwise never read.
func (v *previewer) add(name string, size int64, isDir bool, open func() (io.ReadCloser, error)) {
	name = strings.TrimSuffix(strings.ReplaceAll(name, "\\", "/"), "/")
	v.count++
	v.p.TotalSize += size
	if len(v.p.Entries) < MaxPreviewEntries {
		v.p.Entries = append(v.p.Entries, PreviewEntry{Path: name, Size: size, IsDir: isDir})
	} else {
		v.p.Truncated = true
	}
	if isDir {
		return
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "fomod/moduleconfig.xml") {
		v.p.Fomod = true
	}
	if path.Base(lower) != manifest.FileName || size > maxManifestBytes || len(v.p.Manifests) >= maxPreviewManifests {
		return
	}
	rc, err := open()
	if err != nil {
		return
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, maxManifestBytes))
	if err != nil {
		return
	}
	m, err := manifest.Parse(b)
	if err != nil {
		return
	}
	folder := path.Dir(name)
	if folder == "." {
		folder = ""
	}
	v.p.Manifests = append(v.p.Manifests, PreviewManifest{Folder: folder, ID: m.ModID(), Name: m.Name, Version: m.Version})
}

// full reports whether the archive is past the entry cap Extract would refuse.
func (v *previewer) full() bool {
	if v.count < DefaultMaxEntries {
		return false
	}
	v.p.Truncated = true
	return true
}

// PreviewArchive lists the archive at archivePath without extracting it.
func PreviewArchive(archivePath string) (_ Preview, err error) {
	defer RecoverMalformed(&err)
	f, err := fsx.Open(archivePath)
	if err != nil {
		return Preview{}, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return Preview{}, err
	}
	var magic [headLen]byte
	n, _ := io.ReadFull(f, magic[:])
	v := &previewer{p: Preview{Entries: []PreviewEntry{}, Manifests: []PreviewManifest{}}}
	switch detect(magic[:n]) {
	case fmtZip:
		err = v.zip(f, info.Size())
	case fmtSevenZip:
		err = v.sevenZip(f, info.Size())
	case fmtRAR:
		if _, err = f.Seek(0, io.SeekStart); err == nil {
			err = v.rar(f)
		}
	case fmtTar, fmtGzip, fmtXz, fmtLzma, fmtZstd, fmtBzip2:
		if _, err = f.Seek(0, io.SeekStart); err == nil {
			err = v.stream(f, detect(magic[:n]), archivePath)
		}
	default:
		err = &Error{Reason: ErrUnsupportedFormat}
	}
	if err != nil {
		return Preview{}, err
	}
	return v.p, nil
}

func (v *previewer) zip(r io.ReaderAt, size int64) error {
	zr, err := zip.NewReader(r, size)
	if err != nil && (zr == nil || !errors.Is(err, zip.ErrInsecurePath)) {
		return wrap("", err)
	}
	for _, f := range zr.File {
		if v.full() {
			break
		}
		v.add(zipName(f), int64(min(f.UncompressedSize64, math.MaxInt64)), f.Mode().IsDir(), func() (io.ReadCloser, error) { return f.Open() })
	}
	return nil
}

func (v *previewer) sevenZip(r io.ReaderAt, size int64) error {
	if err := checkSevenZipLimits(r, size); err != nil {
		return wrap("", err)
	}
	zr, err := OpenSevenZip(r, size)
	if err != nil {
		return wrap("", err)
	}
	for _, f := range zr.File {
		if v.full() {
			break
		}
		v.add(f.Name, int64(min(f.UncompressedSize, math.MaxInt64)), f.Mode().IsDir(), f.Open)
	}
	return nil
}

func (v *previewer) rar(r io.Reader) error {
	rr, err := rardecode.NewReader(r, rardecode.MaxDictionarySize(DefaultMaxEntryBytes))
	if err != nil {
		return wrap("", err)
	}
	for !v.full() {
		h, err := rr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return wrap("", err)
		}
		v.add(h.Name, h.UnPackedSize, h.IsDir, func() (io.ReadCloser, error) { return io.NopCloser(rr), nil })
	}
	return nil
}
