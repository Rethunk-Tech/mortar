package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Mod is the display data of a mod page.
type Mod struct {
	Name             string `json:"name"`
	Author           string `json:"author"`
	PictureURL       string `json:"pictureUrl"`
	EndorsementCount int    `json:"endorsementCount"`
	Summary          string `json:"summary"`
}

// Mod returns a mod's page data. It is fetched once and kept on disk, since the picture and endorsement count are
// only wanted at install time; a cache that cannot be read or written only costs a refetch.
func (c *Client) Mod(ctx context.Context, modID int) (Mod, error) {
	dir, dirErr := c.cacheDir()
	path := filepath.Join(dir, fmt.Sprintf("mod-%s-%d.json", Game, modID))
	if dirErr == nil {
		if b, err := fsx.ReadFile(path); err == nil {
			var m Mod
			if json.Unmarshal(b, &m) == nil {
				return m, nil
			}
		}
	}
	var raw struct {
		Name             string `json:"name"`
		Author           string `json:"author"`
		PictureURL       string `json:"picture_url"`
		EndorsementCount int    `json:"endorsement_count"`
		Summary          string `json:"summary"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/games/%s/mods/%d.json", Game, modID), false, &raw); err != nil {
		return Mod{}, err
	}
	m := Mod{Name: raw.Name, Author: raw.Author, PictureURL: raw.PictureURL, EndorsementCount: raw.EndorsementCount, Summary: raw.Summary}
	if dirErr == nil && os.MkdirAll(dir, 0o700) == nil {
		_ = datadir.WriteJSON(path, m)
	}
	return m, nil
}
