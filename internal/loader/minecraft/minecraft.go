// Package minecraft installs Java instances with Fabric, Forge or NeoForge. A Mortar profile is one instance.
package minecraft

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	Fabric     = "fabric"
	Forge      = "forge"
	NeoForge   = "neoforge"
	markerFile = ".mortar-minecraft.json"
)

var _ = []bool{
	loader.Register(Loader{Kind: Fabric}),
	loader.Register(Loader{Kind: Forge}),
	loader.Register(Loader{Kind: NeoForge}),
}

// Family reports whether loaders includes a Minecraft loader.
func Family(loaders []string) bool {
	for _, id := range loaders {
		if id == Fabric || id == Forge || id == NeoForge {
			return true
		}
	}
	return false
}

// Loader is one of Fabric, Forge or NeoForge. The zero Kind is invalid; HTTP and Java are for tests.
type Loader struct {
	Kind string
	HTTP *http.Client
	Java string
}

func (l Loader) ID() string { return l.Kind }

func (Loader) Formats() []string { return nil }

func (Loader) ProfileFiles() []string { return []string{markerFile, "mods"} }

type marker struct {
	Version     string `json:"version"`
	GameVersion string `json:"gameVersion,omitempty"`
}

func readMarker(dir string) marker {
	var m marker
	_, _ = datadir.ReadJSON(filepath.Join(dir, markerFile), &m)
	return m
}

func writeMarker(dir string, m marker) error {
	return datadir.WriteJSON(filepath.Join(dir, markerFile), m)
}

func (l Loader) Status(t loader.Target) (loader.Status, error) {
	m := readMarker(t.ProfileDir)
	if m.Version == "" {
		return loader.Status{}, nil
	}
	return loader.Status{Installed: true, Version: m.Version, GameVersion: m.GameVersion, PerProfile: true}, nil
}

func (l Loader) Install(_ context.Context, t loader.Target, pkg loader.Package, progress func(loader.Step)) (string, error) {
	if err := os.MkdirAll(filepath.Join(t.ProfileDir, "mods"), 0o700); err != nil {
		return "", err
	}
	m := marker{Version: pkg.Version, GameVersion: gameVersion(t.InstallDir)}
	if err := writeMarker(t.ProfileDir, m); err != nil {
		return "", err
	}
	if progress != nil {
		progress(loader.StepFiles)
	}
	return pkg.Version, nil
}

func gameVersion(install string) string {
	entries, err := os.ReadDir(filepath.Join(install, "versions"))
	if err != nil {
		return ""
	}
	var best string
	for _, e := range entries {
		if e.IsDir() && e.Name() > best {
			best = e.Name()
		}
	}
	return best
}

func (l Loader) Contribute(_ context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	m := readMarker(p.Dir)
	if m.Version == "" {
		return errors.New("the " + l.Kind + " loader is not installed in this profile")
	}
	java, err := l.java()
	if err != nil {
		return err
	}
	if err := plan.OverrideExe(l.Kind, java); err != nil {
		return err
	}
	plan.RequireRuntime(launchplan.RuntimeReq{Kind: "java", Key: "17", Value: java})
	plan.AddProcessName(filepath.Base(java))
	main := map[string]string{
		Fabric:   "net.fabricmc.loader.impl.launch.knot.KnotClient",
		Forge:    "cpw.mods.bootstraplauncher.BootstrapLauncher",
		NeoForge: "cpw.mods.bootstraplauncher.BootstrapLauncher",
	}[l.Kind]
	plan.AddArgs(main, "--gameDir", p.Dir)
	return nil
}

func (l Loader) java() (string, error) {
	if l.Java != "" {
		if !safefs.IsFile(l.Java) {
			return "", usererr.New(usererr.NotFound, "Java is not installed.")
		}
		return l.Java, nil
	}
	path, err := exec.LookPath("java")
	if err != nil {
		return "", usererr.New(usererr.NotFound, "Java is not installed.")
	}
	return path, nil
}

func (l Loader) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	all, err := l.Versions(ctx, g)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", errors.New("no " + l.Kind + " releases")
	}
	return all[0], nil
}

func (l Loader) Versions(ctx context.Context, _ components.GameInfo) ([]string, error) {
	body, err := l.get(ctx, l.versionsURL())
	if err != nil {
		return nil, err
	}
	switch l.Kind {
	case Fabric:
		return fabricVersions(body)
	case Forge, NeoForge:
		return mavenVersions(body)
	default:
		return nil, errors.New("unknown minecraft loader")
	}
}

func (l Loader) Fetch(ctx context.Context, _ components.GameInfo, version, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return safefs.WriteFileAtomic(dst, []byte(version), 0o600)
}

func (l Loader) versionsURL() string {
	switch l.Kind {
	case Fabric:
		return "https://meta.fabricmc.net/v2/versions/loader"
	case Forge:
		return "https://maven.minecraftforge.net/net/minecraftforge/forge/maven-metadata.xml"
	default:
		return "https://maven.neoforged.net/releases/net/neoforged/neoforge/maven-metadata.xml"
	}
}

func (l Loader) get(ctx context.Context, url string) ([]byte, error) {
	c := l.HTTP
	if c == nil {
		c = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, res.Status)
	}
	return io.ReadAll(res.Body)
}

func fabricVersions(body []byte) ([]string, error) {
	var rows []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	var out []string
	for _, r := range rows {
		if r.Stable && r.Version != "" {
			out = append(out, r.Version)
		}
	}
	return out, nil
}

func mavenVersions(body []byte) ([]string, error) {
	var meta struct {
		Versioning struct {
			Latest   string   `xml:"latest"`
			Versions []string `xml:"versions>version"`
		} `xml:"versioning"`
	}
	if err := xml.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	out := meta.Versioning.Versions
	if len(out) == 0 && meta.Versioning.Latest != "" {
		return []string{meta.Versioning.Latest}, nil
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
