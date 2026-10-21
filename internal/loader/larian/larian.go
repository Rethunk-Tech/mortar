// Package larian is the loader of Baldur's Gate 3: its mods are .pak files in a folder the game reads, as for the
// folder loader, and the game loads a pak with a meta.lsx only when modsettings.lsx lists it. Before each launch the
// profile's own copy of modsettings.lsx, which the options swap puts in place, is written from its enabled paks.
package larian

import (
	"encoding/xml"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
)

// ID is the catalog's id for this loader.
const ID = "larian"

// settingsFile is the name the options swap gives the profile's copy of the game's modsettings.lsx.
const settingsFile = "modsettings.lsx"

var _ = loader.Register(Loader{})

// Loader places paks as the folder loader does and lists them in modsettings.lsx.
type Loader struct{ folder.Loader }

func (Loader) ID() string { return ID }

// Module is one entry of modsettings.lsx, read from a pak's meta.lsx.
type Module struct {
	Folder, MD5, Name, PublishHandle, UUID, Version64 string
}

// gustav is the game's own module, which modsettings.lsx lists first (Nexus-Mods/vortex-games game-baldursgate3
// common.ts DEFAULT_MOD_SETTINGS_V8).
var gustav = Module{Folder: "GustavX", Name: "GustavX", PublishHandle: "0", UUID: "cb555efe-2d9e-131f-8195-a89329d218ea", Version64: "36028797018963968"}

// Prelaunch writes the profile's modsettings.lsx: the game's module, then every enabled pak that carries a meta.lsx,
// in path order. A pak without one replaces game files and loads without an entry.
func (Loader) Prelaunch(p loader.ProfileView) error {
	info, ok := components.Game(p.Game)
	if !ok {
		return errors.New("unknown game " + p.Game)
	}
	t, ok := info.Target(folder.TargetID)
	if !ok {
		return errors.New(info.Name + " has no " + folder.TargetID + " target")
	}
	root := filepath.Join(p.Dir, filepath.FromSlash(t.ProfileFolder()))
	var paks []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() && strings.EqualFold(filepath.Ext(path), ".pak") {
			paks = append(paks, path)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	slices.Sort(paks)
	mods := []Module{gustav}
	for _, pak := range paks {
		m, ok, err := moduleOf(pak)
		if errors.Is(err, errNotPak) {
			continue
		}
		if err != nil {
			return errors.New(filepath.Base(pak) + ": " + err.Error())
		}
		if ok && !slices.ContainsFunc(mods, func(x Module) bool { return strings.EqualFold(x.UUID, m.UUID) }) {
			mods = append(mods, m)
		}
	}
	return datadir.WriteFile(filepath.Join(p.Dir, settingsFile), modSettings(mods), 0o600)
}

// moduleOf reads the module a pak declares in its Mods/<folder>/meta.lsx.
func moduleOf(pak string) (Module, bool, error) {
	raw, _, ok, err := readPakFile(pak, func(name string) bool {
		dir, file := path.Split(name)
		return strings.EqualFold(file, "meta.lsx") && strings.Count(dir, "/") == 2 && strings.HasPrefix(strings.ToLower(dir), "mods/")
	})
	if err != nil || !ok {
		return Module{}, false, err
	}
	return parseMeta(raw)
}

// lsxNode is a region or node of an .lsx file: a region holds its nodes directly, a node under <children>.
type lsxNode struct {
	ID         string    `xml:"id,attr"`
	Attributes []lsxAttr `xml:"attribute"`
	Nodes      []lsxNode `xml:"node"`
	Children   []lsxNode `xml:"children>node"`
}

type lsxAttr struct {
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

// parseMeta reads the ModuleInfo node of a meta.lsx; a pak without a UUID declares no module.
func parseMeta(raw []byte) (Module, bool, error) {
	var doc struct {
		Regions []lsxNode `xml:"region"`
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return Module{}, false, err
	}
	var find func(nodes []lsxNode) (lsxNode, bool)
	find = func(nodes []lsxNode) (lsxNode, bool) {
		for _, n := range nodes {
			if n.ID == "ModuleInfo" {
				return n, true
			}
			if m, ok := find(append(slices.Clone(n.Nodes), n.Children...)); ok {
				return m, true
			}
		}
		return lsxNode{}, false
	}
	n, ok := find(doc.Regions)
	if !ok {
		return Module{}, false, nil
	}
	m := Module{PublishHandle: "0"}
	for _, a := range n.Attributes {
		switch a.ID {
		case "Folder":
			m.Folder = a.Value
		case "MD5":
			m.MD5 = a.Value
		case "Name":
			m.Name = a.Value
		case "PublishHandle":
			m.PublishHandle = a.Value
		case "UUID":
			m.UUID = a.Value
		case "Version64", "Version":
			m.Version64 = a.Value
		}
	}
	if m.UUID == "" || m.Folder == "" {
		return Module{}, false, nil
	}
	return m, true, nil
}

// modSettings is modsettings.lsx in the game's v8 format (Patch 7 on): one ModuleShortDesc per module, in load order.
func modSettings(mods []Module) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<save>
    <version major="4" minor="8" revision="0" build="10"/>
    <region id="ModuleSettings">
        <node id="root">
            <children>
                <node id="Mods">
                    <children>
`)
	attr := func(id, kind, value string) {
		b.WriteString(`                            <attribute id="` + id + `" type="` + kind + `" value="`)
		_ = xml.EscapeText(&b, []byte(value))
		b.WriteString("\"/>\n")
	}
	for _, m := range mods {
		b.WriteString("                        <node id=\"ModuleShortDesc\">\n")
		attr("Folder", "LSString", m.Folder)
		attr("MD5", "LSString", m.MD5)
		attr("Name", "LSString", m.Name)
		attr("PublishHandle", "uint64", m.PublishHandle)
		attr("UUID", "guid", m.UUID)
		attr("Version64", "int64", m.Version64)
		b.WriteString("                        </node>\n")
	}
	b.WriteString(`                    </children>
                </node>
            </children>
        </node>
    </region>
</save>
`)
	return []byte(b.String())
}
