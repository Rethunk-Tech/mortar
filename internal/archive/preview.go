package archive

import (
	"errors"
	"io"
	"path"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
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

// ManifestFile is the manifest every mod format ships: SMAPI's, and a Thunderstore package's.
const ManifestFile = "manifest.json"

// PreviewManifest is a manifest.json found in an archive: a SMAPI mod's, or a Thunderstore package's, which has no ID.
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

// previewMaxEntryBytes bounds one entry's decompressed bytes in a listing; the largest Thunderstore package measured was 4.881 GiB.
const previewMaxEntryBytes = 8 << 30

type previewer struct {
	p     Preview
	count int
}

// add records the listing's current entry; its body is opened only for a manifest.
func (v *previewer) add(l *Listing, e Entry) {
	name, size, isDir := e.Name, max(e.Size, 0), e.IsDir
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
	if path.Base(lower) != ManifestFile || size > maxManifestBytes || len(v.p.Manifests) >= maxPreviewManifests {
		return
	}
	rc, err := l.Open()
	if err != nil {
		return
	}
	defer func() { _ = rc.Close() }()
	b, err := safefs.ReadCapped(rc, maxManifestBytes)
	if err != nil {
		return
	}
	pm, ok := ManifestInfo(b)
	if !ok {
		return
	}
	folder := path.Dir(name)
	if folder == "." {
		folder = ""
	}
	pm.Folder = folder
	v.p.Manifests = append(v.p.Manifests, pm)
}

// ManifestInfo reads the identity a manifest.json states (Folder is left empty): a SMAPI manifest gives an ID, a Thunderstore
// package's manifest only its name and version. ok is false when the file is neither.
func ManifestInfo(b []byte) (PreviewManifest, bool) {
	if m, err := smapimanifest.Parse(b); err == nil {
		return PreviewManifest{ID: m.ModID(), Name: m.Name, Version: m.Version}, true
	}
	pkg, err := tstore.ParseManifest(b)
	if err != nil || pkg.Name == "" {
		return PreviewManifest{}, false
	}
	return PreviewManifest{Name: pkg.Name, Version: pkg.Version}, true
}

// PreviewArchive lists the archive at archivePath without extracting it.
func PreviewArchive(archivePath string) (_ Preview, err error) {
	defer RecoverMalformed(&err)
	l, err := List(archivePath, previewMaxEntryBytes)
	if err != nil {
		return Preview{}, err
	}
	defer func() { _ = l.Close() }()
	v := &previewer{p: Preview{Entries: []PreviewEntry{}, Manifests: []PreviewManifest{}}}
	for {
		if v.count >= DefaultMaxEntries {
			v.p.Truncated = true
			break
		}
		e, err := l.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Preview{}, err
		}
		v.add(l, e)
	}
	return v.p, nil
}
