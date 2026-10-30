package devsvc

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// releaseOutput is where an adopted project's Release build goes, apart from the Debug output that installs.
func releaseOutput(p settings.DevProject) string {
	return filepath.Join(p.Dir, "artifacts", "bin", p.Name, "release")
}

// providedPrefixes name assemblies the game, Unity, .NET or BepInEx already supply; a build copies them beside the
// mod's own DLL when a reference is not marked private, and a package must not carry them.
var providedPrefixes = []string{
	"System.", "Microsoft.", "Mono.", "MonoMod", "UnityEngine", "Unity.", "Assembly-CSharp", "BepInEx", "0Harmony",
	"mscorlib", "netstandard", "Il2Cpp", "Newtonsoft.", "Facepunch.", "Netcode.", "com.unity.",
}

var (
	assemblyNameTag = regexp.MustCompile(`<AssemblyName>\s*([^<\s]+)\s*</AssemblyName>`)
	versionTag      = regexp.MustCompile(`<Version>\s*(\d+\.\d+\.\d+)(?:\.\d+)?\s*</Version>`)
	notNameChar     = regexp.MustCompile(`[^A-Za-z0-9_]`)
)

// packBuilt packs the Release output of an adopted project for Thunderstore: its assembly, the other DLLs and asset
// bundles the build copied that nothing else supplies, the manifest at the tag's version (else the project file's),
// and README.md and icon.png when the project has them. It returns the zip and the names it holds.
func (s *Service) packBuilt(ctx context.Context, gameID string, p settings.DevProject, csproj string) (string, []string, error) {
	rec, _ := readRecord(p.Dir)
	project, _ := safefs.ReadFile(csproj)
	assembly := strings.TrimSuffix(filepath.Base(csproj), filepath.Ext(csproj))
	if m := assemblyNameTag.FindSubmatch(project); m != nil {
		assembly = string(m[1])
	}
	version := ""
	if m := versionTag.FindSubmatch(project); m != nil {
		version = string(m[1])
	}
	if v, err := s.NextVersions(ctx, gameID, p.Dir); err == nil && v.Current != "" {
		version = v.Current
	}
	version = cmp.Or(version, "0.1.0")
	built, _ := filepath.Glob(filepath.Join(releaseOutput(p), "*"))
	var payload []string
	for _, f := range built {
		base := filepath.Base(f)
		ext := strings.ToLower(filepath.Ext(base))
		own := strings.EqualFold(base, assembly+".dll")
		provided := slices.ContainsFunc(providedPrefixes, func(pre string) bool { return strings.HasPrefix(base, pre) })
		if info, err := os.Stat(f); err != nil || !info.Mode().IsRegular() {
			continue
		}
		if ext == ".dll" && (own || !provided) || ext == ".bundle" {
			payload = append(payload, f)
		}
	}
	if !slices.ContainsFunc(payload, func(f string) bool { return strings.EqualFold(filepath.Base(f), assembly+".dll") }) {
		return "", nil, usererr.Wrap(usererr.NotFound, fmt.Errorf("the build wrote no %s.dll", assembly))
	}
	name := notNameChar.ReplaceAllString(p.Name, "_")
	manifest, err := completeManifest(p.Dir, name, version)
	if err != nil {
		return "", nil, err
	}
	out := filepath.Join(p.Dir, "artifacts", "thunderstore", "release")
	if err := os.MkdirAll(out, 0o750); err != nil {
		return "", nil, err
	}
	zipPath := filepath.Join(out, fmt.Sprintf("%s-%s-%s.zip", cmp.Or(rec.Team, "Team"), name, version))
	f, err := safefs.Create(zipPath)
	if err != nil {
		return "", nil, err
	}
	zw := zip.NewWriter(f)
	files := []string{"manifest.json"}
	err = addZipFile(zw, "manifest.json", bytes.NewReader(manifest))
	for _, doc := range []string{"README.md", "icon.png"} {
		for _, dir := range []string{p.Dir, filepath.Dir(csproj)} {
			if body, rerr := safefs.ReadFile(filepath.Join(dir, doc)); rerr == nil {
				err = errors.Join(err, addZipFile(zw, doc, bytes.NewReader(body)))
				files = append(files, doc)
				break
			}
		}
	}
	for _, f := range payload {
		body, rerr := safefs.ReadFile(f)
		err = errors.Join(err, rerr, addZipFile(zw, filepath.Base(f), bytes.NewReader(body)))
		files = append(files, filepath.Base(f))
	}
	return zipPath, files, errors.Join(err, zw.Close(), f.Close())
}
