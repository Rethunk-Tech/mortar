package dlwatch

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/manifest"
)

func readManifestName(r io.Reader) string {
	b, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	m, err := manifest.Parse(b)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(m.Name)
}

// PeekName reads a Stardew-style manifest.json Name from a zip, shortest path first.
func PeekName(path string) string {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return ""
	}
	defer func() { _ = zr.Close() }()
	best := ""
	bestDepth := 1 << 20
	for _, f := range zr.File {
		if !strings.EqualFold(filepath.Base(f.Name), "manifest.json") {
			continue
		}
		depth := strings.Count(filepath.ToSlash(f.Name), "/")
		if depth >= bestDepth {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		name := readManifestName(rc)
		_ = rc.Close()
		if name == "" {
			continue
		}
		best = name
		bestDepth = depth
	}
	return best
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
