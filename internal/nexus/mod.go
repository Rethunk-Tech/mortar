package nexus

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// Mod is the display data of a mod page.
type Mod struct {
	Name             string `json:"name"`
	Author           string `json:"author"`
	PictureURL       string `json:"pictureUrl"`
	EndorsementCount int    `json:"endorsementCount"`
	Summary          string `json:"summary"`
}

// Changelog is one version's notes, as plain text. A GitHub release carries its markdown Body and publish Date
// instead of Notes.
type Changelog struct {
	Version string   `json:"version"`
	Date    string   `json:"date,omitempty"`
	Notes   []string `json:"notes"`
	Body    string   `json:"body,omitempty"`
}

// CachedMod returns a mod's page data. It is fetched once and kept in cache, since the picture and endorsement
// count are only wanted at install time; a cache that cannot be read or written only costs a refetch. A nil
// cache fetches every time.
func CachedMod(ctx context.Context, c *nexusmods.Client, cache *diskcache.Cache, t nexusmods.Title, modID int) (Mod, error) {
	name := fmt.Sprintf("%s%s-%d.json", meta.NexusModPrefix, t.Domain, modID)
	if m, ok := diskcache.Peek[Mod](cache, name); ok {
		return m.Value, nil
	}
	p, err := c.Page(ctx, t, modID)
	if err != nil {
		return Mod{}, Worded(err)
	}
	m := Mod{Name: p.Name, Author: p.Author, PictureURL: p.PictureURL, EndorsementCount: p.Endorsements, Summary: p.Summary}
	_ = diskcache.Put(cache, name, m)
	return m, nil
}
