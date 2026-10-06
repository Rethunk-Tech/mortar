package pack

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

// Modpack reads a Thunderstore modpack package: a package zip whose manifest dependencies are the mod list and
// whose config/ folder holds the pack's settings. A package that ships code (a .dll) is a mod, not a modpack.
type Modpack struct{}

// ID names the format.
func (Modpack) ID() string { return "thunderstore-modpack" }

func readModpack(path string) (bepinex5.Manifest, map[string][]byte, bool) {
	data, err := readCapped(path, maxInput)
	if err != nil {
		return bepinex5.Manifest{}, nil, false
	}
	files, err := readZip(data)
	if err != nil {
		return bepinex5.Manifest{}, nil, false
	}
	m, err := bepinex5.ParseManifest(files["manifest.json"])
	if err != nil || len(m.Dependencies) == 0 {
		return bepinex5.Manifest{}, nil, false
	}
	for name := range files {
		if strings.EqualFold(filepath.Ext(name), ".dll") {
			return bepinex5.Manifest{}, nil, false
		}
	}
	return m, files, true
}

// Detect accepts a .zip path whose manifest has dependencies and no code.
func (Modpack) Detect(in Input) bool {
	if !strings.EqualFold(filepath.Ext(in.Path), ".zip") {
		return false
	}
	_, _, ok := readModpack(in.Path)
	return ok
}

// Parse lists the dependencies as packages and the zip's config/ as Configs.
func (Modpack) Parse(_ context.Context, in Input) (Draft, error) {
	m, files, ok := readModpack(in.Path)
	if !ok {
		return Draft{}, errors.New("not a Thunderstore modpack")
	}
	d := Draft{Name: m.Name, Configs: filesUnder(files, "config/")}
	for _, dep := range m.Dependencies {
		i := strings.LastIndex(dep, "-")
		if i < 1 || !nativeID.MatchString(dep[:i]) || !versionNumber.MatchString(dep[i+1:]) {
			return Draft{}, fmt.Errorf("malformed dependency %q", dep)
		}
		d.Packages = append(d.Packages, Ref{Source: thunderstore, Native: dep[:i], Version: dep[i+1:]})
	}
	return d, nil
}
