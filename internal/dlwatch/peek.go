package dlwatch

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
)

// maxManifestBytes bounds one manifest.json read: this peek runs before the capped extractor,
// so a crafted archive in Downloads must not be able to make it inflate gigabytes.
const maxManifestBytes = 1 << 20

func readManifestName(r io.Reader) string {
	b, err := safefs.ReadCapped(r, maxManifestBytes)
	if err != nil {
		return ""
	}
	m, _ := archive.ManifestInfo(b)
	return strings.TrimSpace(m.Name)
}

// PeekName reads a SMAPI or Thunderstore manifest.json Name from an archive of any format go-archive reads, shortest
// path first. Every entry read is capped at maxManifestBytes.
func PeekName(path string) string {
	l, err := archive.List(path, maxManifestBytes)
	if err != nil {
		return ""
	}
	defer func() { _ = l.Close() }()
	best := ""
	bestDepth := 1 << 20
	for {
		e, err := l.Next()
		if err != nil {
			return best
		}
		if e.IsDir || !strings.EqualFold(e.Name[strings.LastIndex(e.Name, "/")+1:], archive.ManifestFile) || e.Size > maxManifestBytes {
			continue
		}
		depth := strings.Count(e.Name, "/")
		if depth >= bestDepth {
			continue
		}
		rc, err := l.Open()
		if err != nil {
			continue
		}
		name := readManifestName(rc)
		_ = rc.Close()
		if name != "" {
			best, bestDepth = name, depth
		}
	}
}

// Identify uses the Nexus file-name pattern, then a manifest peek, then the basename.
func Identify(path string) Info {
	if inf, ok := ParseNexusFilename(path); ok {
		return inf
	}
	if name := PeekName(path); name != "" {
		return Info{Name: name}
	}
	return Info{Name: filepath.Base(path)}
}
