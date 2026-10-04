package sharesvc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"

	"github.com/bodgit/sevenzip"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/share"
)

const (
	collectionManifestName = "collection.json"
	collectionBundleDir    = "bundled"
	maxCollectionJSON      = 8 << 20
)

type modFile struct{ modID, fileID int }

// collectionDetails is what the curator's archive adds to the mod list Nexus's GraphQL gives.
type collectionDetails struct {
	// Fomod is the curator's installer choices by Nexus file, ready for Ref.Fomod.
	Fomod map[modFile]map[string]map[string][]string
	// Configs are config files of the collection's bundled mods, written once a mod with that UniqueID is installed.
	Configs []share.Config
}

// readCollectionArchive reads the curator's 7z: collection.json for FOMOD choices, and the config files inside
// bundled/<mod>/. Only a bundled mod's config.json and config/*.json are taken, never its content.
func readCollectionArchive(raw []byte) (collectionDetails, error) {
	zr, err := sevenzip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return collectionDetails{}, err
	}
	var d collectionDetails
	var found bool
	bundles := map[string]*bundle{}
	var total int64
	for _, f := range zr.File {
		name := path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))
		switch {
		case name == collectionManifestName:
			b, err := readCapped(f, maxCollectionJSON)
			if err != nil {
				return collectionDetails{}, err
			}
			if d.Fomod, err = parseChoices(b); err != nil {
				return collectionDetails{}, err
			}
			found = true
		case strings.HasPrefix(name, collectionBundleDir+"/") && !f.FileInfo().IsDir() && strings.HasSuffix(strings.ToLower(name), ".json"):
			rest := strings.TrimPrefix(name, collectionBundleDir+"/")
			dir, rel, ok := strings.Cut(rest, "/")
			if !ok {
				continue
			}
			b, err := readCapped(f, share.MaxConfigBytes)
			if err != nil {
				continue
			}
			if total += int64(len(b)); total > share.MaxConfigTotal {
				continue
			}
			bd := bundles[dir]
			if bd == nil {
				bd = &bundle{files: map[string][]byte{}}
				bundles[dir] = bd
			}
			bd.files[rel] = b
		}
	}
	if !found {
		return collectionDetails{}, errors.New("the collection archive has no collection.json")
	}
	for _, bd := range bundles {
		d.Configs = append(d.Configs, bd.configs()...)
	}
	return d, nil
}

type bundle struct{ files map[string][]byte }

// configs maps a bundle's config files to the UniqueID of the manifest at the shallowest folder, paths relative to it.
func (b *bundle) configs() []share.Config {
	root, uid := "", ""
	for rel, data := range b.files {
		if path.Base(rel) != manifest.FileName {
			continue
		}
		m, err := manifest.Parse(data)
		if err != nil {
			continue
		}
		if dir := path.Dir(rel); uid == "" || len(dir) < len(root) {
			root, uid = dir, m.UniqueID
		}
	}
	if uid == "" {
		return nil
	}
	var out []share.Config
	for rel, data := range b.files {
		local := rel
		if root != "." {
			var ok bool
			if local, ok = strings.CutPrefix(rel, root+"/"); !ok {
				continue
			}
		}
		lower := strings.ToLower(local)
		if (lower != "config.json" && !strings.HasPrefix(lower, "config/")) || !share.ValidConfigPath(local) {
			continue
		}
		out = append(out, share.Config{UniqueID: uid, Path: local, Data: data})
	}
	return out
}

func readCapped(f *sevenzip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("file too large")
	}
	return b, nil
}

// parseChoices keeps Vortex's FOMOD choices (step, group and plugin names; the index is redundant) for Nexus mods.
func parseChoices(raw []byte) (map[modFile]map[string]map[string][]string, error) {
	var doc collectionDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := map[modFile]map[string]map[string][]string{}
	for _, m := range doc.Mods {
		if m.Source.Type != "nexus" || m.Source.ModID == 0 || m.Source.FileID == 0 || m.Choices == nil || m.Choices.Type != "fomod" {
			continue
		}
		choices := map[string]map[string][]string{}
		for _, step := range m.Choices.Options {
			groups := map[string][]string{}
			for _, g := range step.Groups {
				plugins := make([]string, 0, len(g.Choices))
				for _, c := range g.Choices {
					plugins = append(plugins, c.Name)
				}
				groups[g.Name] = plugins
			}
			choices[step.Name] = groups
		}
		if len(choices) > 0 && share.ValidFomod(choices) {
			out[modFile{m.Source.ModID, m.Source.FileID}] = choices
		}
	}
	return out, nil
}
