package sharesvc

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// The collection.json shape is Vortex's ICollection (Nexus-Mods/Vortex,
// src/renderer/src/extensions/collections/types/ICollection.ts). The importer reads it from the curator's archive
// and the exporter writes it, through the same types.
type collectionDoc struct {
	Info     collectionInfo  `json:"info"`
	Mods     []collectionMod `json:"mods"`
	ModRules []struct{}      `json:"modRules"`
}

type collectionInfo struct {
	Author              string `json:"author"`
	AuthorURL           string `json:"authorUrl"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	InstallInstructions string `json:"installInstructions"`
	DomainName          string `json:"domainName"`
}

type collectionMod struct {
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Optional     bool               `json:"optional"`
	DomainName   string             `json:"domainName"`
	Source       collectionSource   `json:"source"`
	Choices      *collectionChoices `json:"choices,omitempty"`
	Instructions string             `json:"instructions,omitempty"`
}

type collectionSource struct {
	Type            string `json:"type"`
	URL             string `json:"url,omitempty"`
	Instructions    string `json:"instructions,omitempty"`
	ModID           int    `json:"modId,omitempty"`
	FileID          int    `json:"fileId,omitempty"`
	UpdatePolicy    string `json:"updatePolicy,omitempty"`
	MD5             string `json:"md5,omitempty"`
	FileSize        int64  `json:"fileSize,omitempty"`
	LogicalFilename string `json:"logicalFilename,omitempty"`
}

type collectionChoices struct {
	Type    string       `json:"type"`
	Options []choiceStep `json:"options"`
}

type choiceStep struct {
	Name   string        `json:"name"`
	Groups []choiceGroup `json:"groups"`
}

type choiceGroup struct {
	Name    string         `json:"name"`
	Choices []choicePlugin `json:"choices"`
}

type choicePlugin struct {
	Name string `json:"name"`
	Idx  int    `json:"idx"`
}

// fileFacts looks up what the store records lack: Nexus's md5, size and category of a file. ok is false when
// unknown (signed out, offline, file gone).
type fileFacts func(modID, fileID int) (nexus.File, bool)

func choicesOf(fomod map[string]map[string][]string) *collectionChoices {
	if len(fomod) == 0 {
		return nil
	}
	c := &collectionChoices{Type: "fomod"}
	for _, step := range slices.Sorted(maps.Keys(fomod)) {
		st := choiceStep{Name: step}
		for _, group := range slices.Sorted(maps.Keys(fomod[step])) {
			g := choiceGroup{Name: group}
			for i, plugin := range fomod[step][group] {
				g.Choices = append(g.Choices, choicePlugin{Name: plugin, Idx: i})
			}
			st.Groups = append(st.Groups, g)
		}
		c.Options = append(c.Options, st)
	}
	return c
}

// modOf maps one enabled entry to a collection mod. Nexus files keep their ids; GitHub assets become direct
// downloads; a local archive has no public source, so it is a manual one the curator must complete.
func modOf(e profile.Entry, domain string, facts fileFacts) collectionMod {
	name := e.Source.Name
	if len(e.Mods) > 0 && e.Mods[0].Name != "" {
		name = e.Mods[0].Name
	}
	name = cmp.Or(name, e.Key)
	m := collectionMod{Name: name, Version: e.Source.Version, DomainName: domain, Instructions: e.Note}
	if len(e.Mods) > 0 && m.Version == "" {
		m.Version = e.Mods[0].Version
	}
	switch e.Source.Kind {
	case profile.KindNexus:
		m.Source = collectionSource{Type: "nexus", ModID: e.Source.ModID, FileID: e.Source.FileID, UpdatePolicy: "exact", LogicalFilename: e.Source.Name}
		if f, ok := facts(e.Source.ModID, e.Source.FileID); ok {
			m.Source.MD5 = f.MD5
			m.Source.FileSize = f.SizeKB * 1024
			m.Optional = strings.EqualFold(f.Category, "OPTIONAL")
		}
		m.Choices = choicesOf(e.Fomod)
	case profile.KindGitHub:
		m.Source = collectionSource{Type: "direct", URL: "https://github.com/" + e.Source.Repo + "/releases/download/" + e.Source.Tag + "/" + e.Source.Asset, LogicalFilename: e.Source.Asset}
	default:
		m.Source = collectionSource{
			Type: "manual", LogicalFilename: e.Source.Name,
			Instructions: "Mortar had this mod from a local archive. Host it and replace this source before publishing.",
		}
	}
	return m
}

// bundledFile is one file of the export's bundled/ folder, in the layout readCollectionArchive reads.
type bundledFile struct {
	Path string
	Data []byte
}

// buildCollection is the draft of a profile: its enabled mods and notes, plus each mod's .json config files with
// its manifest (which names the mod to the importer). Mortar cannot fill info.author, and a Nexus file's md5 and
// size only when facts knows it.
func buildCollection(p profile.Profile, domain, modsDir string, facts fileFacts) (collectionDoc, []bundledFile, []string, error) {
	doc := collectionDoc{
		Info: collectionInfo{Name: p.Name, Description: p.Description, InstallInstructions: p.Notes, DomainName: domain},
		Mods: []collectionMod{}, ModRules: []struct{}{},
	}
	var files []bundledFile
	var skipped []string
	for _, e := range p.Entries {
		if e.Source.Bundled() || !share.Enabled(e) {
			continue
		}
		doc.Mods = append(doc.Mods, modOf(e, domain, facts))
		for _, m := range e.Mods {
			if !e.Enabled(m.ID) {
				continue
			}
			found, skip, err := share.ReadConfigs(modsDir, e.Key, m)
			if err != nil {
				return collectionDoc{}, nil, nil, err
			}
			skipped = append(skipped, skip...)
			if len(found) == 0 {
				continue
			}
			mf := filepath.Join(modsDir, e.Key)
			if m.Folder != "." {
				mf = filepath.Join(mf, filepath.FromSlash(m.Folder))
			}
			raw, err := fsx.ReadFile(filepath.Join(mf, manifest.FileName))
			if err != nil {
				skipped = append(skipped, m.ID.Local()+"/"+manifest.FileName)
				continue
			}
			dir := "bundled/" + m.ID.Local() + "/"
			files = append(files, bundledFile{dir + manifest.FileName, raw})
			for _, c := range found {
				files = append(files, bundledFile{dir + c.Path, c.Data})
			}
		}
	}
	return doc, files, skipped, nil
}

// zipCollection packs collection.json and the bundled/ files into one archive. Vortex packs a 7z and no Go library
// here writes one, so the curator repacks this zip before uploading it to Nexus.
func zipCollection(manifestJSON []byte, files []bundledFile) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	all := append([]bundledFile{{collectionManifestName, manifestJSON}}, files...)
	for _, f := range all {
		w, err := zw.Create(f.Path)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportedCollection is the outcome of ExportCollection. Path is empty when the dialog was cancelled; Skipped lists
// config files left out for their size or name.
type ExportedCollection struct {
	Path     string   `json:"path"`
	Skipped  []string `json:"skipped"`
	External int      `json:"external"`
}

// ExportCollection asks where to save the profile's collection draft, a zip of collection.json and a bundled/
// folder of the mods' config files. It makes no Nexus call that changes anything.
func (s *Service) ExportCollection(ctx context.Context, game, profileID string) (ExportedCollection, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return ExportedCollection{}, err
	}
	modsDir, err := s.d.Profiles.ModsDir(game, profileID)
	if err != nil {
		return ExportedCollection{}, err
	}
	info, ok := components.Game(game)
	if !ok || info.NexusDomain() == "" {
		return ExportedCollection{}, fmt.Errorf("%s has no Nexus page to make a collection for", game)
	}
	t := nexus.Title{Domain: info.NexusDomain(), ID: info.NexusID()}
	dest, err := s.App.SaveFile(winhost.Dialog{Filename: "collection.zip", Filters: []winhost.Filter{{Name: "Nexus collection draft (zip)", Pattern: "*.zip"}}})
	if err != nil || dest == "" {
		return ExportedCollection{Skipped: []string{}}, err
	}
	return s.writeCollection(ctx, p, t, modsDir, dest)
}

func (s *Service) writeCollection(ctx context.Context, p profile.Profile, t nexus.Title, modsDir, dest string) (ExportedCollection, error) {
	cache := map[int]map[int]nexus.File{}
	facts := func(modID, fileID int) (nexus.File, bool) {
		if s.d.Files == nil || !s.d.SignedIn() {
			return nexus.File{}, false
		}
		byID, ok := cache[modID]
		if !ok {
			byID = map[int]nexus.File{}
			if files, err := s.d.Files(ctx, t, modID); err == nil {
				for _, f := range files {
					byID[f.FileID] = f
				}
			}
			cache[modID] = byID
		}
		f, ok := byID[fileID]
		return f, ok
	}
	doc, files, skipped, err := buildCollection(p, t.Domain, modsDir, facts)
	if err != nil {
		return ExportedCollection{}, err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ExportedCollection{}, err
	}
	packed, err := zipCollection(raw, files)
	if err != nil {
		return ExportedCollection{}, err
	}
	if err := datadir.WriteFile(dest, packed, filePerm); err != nil {
		return ExportedCollection{}, err
	}
	external := 0
	for _, m := range doc.Mods {
		if m.Source.Type != "nexus" {
			external++
		}
	}
	s.mu.Lock()
	s.lastExport = dest
	s.mu.Unlock()
	return ExportedCollection{Path: dest, Skipped: append([]string{}, skipped...), External: external}, nil
}

// ShowExportedCollection opens the folder of the draft ExportCollection last wrote.
func (s *Service) ShowExportedCollection() error {
	s.mu.Lock()
	dest := s.lastExport
	s.mu.Unlock()
	if dest == "" {
		return nil
	}
	return datadir.Open(filepath.Dir(dest))
}
