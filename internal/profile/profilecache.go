package profile

import (
	"maps"
	"os"
	"slices"
	"sync"
)

// parsedProfiles holds each profile.json decoded, by path, while the file is still the one it was decoded from: every
// write replaces the file through a rename, so a changed identity, size or modification time means it was rewritten.
// A start decodes each profile dozens of times (every List), and the file only changes when Mortar writes it.
var parsedProfiles sync.Map

type parsedProfile struct {
	file os.FileInfo
	p    Profile
}

func cachedProfile(path string, fi os.FileInfo) (Profile, bool) {
	c, ok := parsedProfiles.Load(path)
	if !ok {
		return Profile{}, false
	}
	pp, ok := c.(parsedProfile)
	if !ok || !os.SameFile(fi, pp.file) || !fi.ModTime().Equal(pp.file.ModTime()) || fi.Size() != pp.file.Size() {
		return Profile{}, false
	}
	return pp.p.Clone(), true
}

func rememberProfile(path string, fi os.FileInfo, p Profile) {
	parsedProfiles.Store(path, parsedProfile{fi, p.Clone()})
}

func forgetProfile(path string) { parsedProfiles.Delete(path) }

// Clone copies p with no slice, map or pointer shared, so a caller changing its copy never reaches the original.
// Unexported fields are copied as they are: decoding never sets them.
func (p Profile) Clone() Profile {
	if p.Entries != nil {
		entries := make([]Entry, len(p.Entries))
		for i, e := range p.Entries {
			entries[i] = e.Clone()
		}
		p.Entries = entries
	}
	p.Groups = cloneSlice(p.Groups, Group.clone)
	p.Collection = clonePtr(p.Collection)
	p.LaunchPresets = slices.Clone(p.LaunchPresets)
	p.Overrides = maps.Clone(p.Overrides)
	return p
}

// Clone copies e with no slice, map or pointer shared.
func (e Entry) Clone() Entry {
	e.PreviousSource = clonePtr(e.PreviousSource)
	e.Mods = cloneSlice(e.Mods, Component.clone)
	e.TrayFiles = slices.Clone(e.TrayFiles)
	e.Disabled = slices.Clone(e.Disabled)
	e.SkipSources = slices.Clone(e.SkipSources)
	e.Tags = slices.Clone(e.Tags)
	e.Fomod = cloneFomod(e.Fomod)
	e.ExtraStoreKeys = slices.Clone(e.ExtraStoreKeys)
	e.PreviousExtraStoreKeys = slices.Clone(e.PreviousExtraStoreKeys)
	return e
}

func (g Group) clone() Group {
	g.Keys = slices.Clone(g.Keys)
	return g
}

func (c Component) clone() Component {
	c.Needs = slices.Clone(c.Needs)
	c.Optional = slices.Clone(c.Optional)
	c.LoadAfter = slices.Clone(c.LoadAfter)
	return c
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	c := *p
	return &c
}

// cloneSlice copies in with each element cloned, keeping nil and empty apart.
func cloneSlice[T any](in []T, clone func(T) T) []T {
	if in == nil {
		return nil
	}
	out := make([]T, len(in))
	for i, v := range in {
		out[i] = clone(v)
	}
	return out
}
