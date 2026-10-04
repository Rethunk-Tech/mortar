package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	p, err := c.Page(ctx, modID)
	if err != nil {
		return Mod{}, err
	}
	m := Mod{Name: p.Name, Author: p.Author, PictureURL: p.PictureURL, EndorsementCount: p.Endorsements, Summary: p.Summary}
	if dirErr == nil && os.MkdirAll(dir, 0o700) == nil {
		_ = datadir.WriteJSON(path, m)
	}
	return m, nil
}

// Page is everything a mod page reports. Description is Nexus's BBCode with <br /> line breaks, unrendered.
// Status is published, under_moderation, not_published, publish_with_game, removed, wastebinned or hidden; an
// unavailable page comes without its text.
type Page struct {
	ModID           int       `json:"modId"`
	Name            string    `json:"name"`
	Summary         string    `json:"summary"`
	Description     string    `json:"description"`
	PictureURL      string    `json:"pictureUrl"`
	Version         string    `json:"version"`
	Author          string    `json:"author"`
	UploadedBy      string    `json:"uploadedBy"`
	UploaderURL     string    `json:"uploaderUrl"`
	CategoryID      int       `json:"categoryId"`
	Endorsements    int       `json:"endorsements"`
	Downloads       int       `json:"downloads"`
	UniqueDownloads int       `json:"uniqueDownloads"`
	Created         time.Time `json:"created"`
	Updated         time.Time `json:"updated"`
	Adult           bool      `json:"adult"`
	Status          string    `json:"status"`
	Available       bool      `json:"available"`
	// Endorsement is the signed-in user's endorse_status (Endorsed, Abstained, Undecided); empty when Nexus omitted it.
	Endorsement string `json:"endorsement"`
}

// Page fetches a mod page, uncached.
func (c *Client) Page(ctx context.Context, modID int) (Page, error) {
	var raw struct {
		ModID           int       `json:"mod_id"`
		Name            string    `json:"name"`
		Summary         string    `json:"summary"`
		Description     string    `json:"description"`
		PictureURL      string    `json:"picture_url"`
		Version         string    `json:"version"`
		Author          string    `json:"author"`
		UploadedBy      string    `json:"uploaded_by"`
		UploaderURL     string    `json:"uploaded_users_profile_url"`
		CategoryID      int       `json:"category_id"`
		Endorsements    int       `json:"endorsement_count"`
		Downloads       int       `json:"mod_downloads"`
		UniqueDownloads int       `json:"mod_unique_downloads"`
		Created         time.Time `json:"created_time"`
		Updated         time.Time `json:"updated_time"`
		Adult           bool      `json:"contains_adult_content"`
		Status          string    `json:"status"`
		Available       bool      `json:"available"`
		Endorsement     *struct {
			Status string `json:"endorse_status"`
		} `json:"endorsement"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/games/%s/mods/%d.json", Game, modID), false, &raw); err != nil {
		return Page{}, err
	}
	p := Page{
		ModID: raw.ModID, Name: raw.Name, Summary: raw.Summary, Description: raw.Description,
		PictureURL: raw.PictureURL, Version: raw.Version, Author: raw.Author, UploadedBy: raw.UploadedBy,
		UploaderURL: raw.UploaderURL, CategoryID: raw.CategoryID, Endorsements: raw.Endorsements,
		Downloads: raw.Downloads, UniqueDownloads: raw.UniqueDownloads, Created: raw.Created.UTC(),
		Updated: raw.Updated.UTC(), Adult: raw.Adult, Status: raw.Status, Available: raw.Available,
	}
	if raw.Endorsement != nil {
		p.Endorsement = raw.Endorsement.Status
	}
	return p, nil
}

// Changelog is one version's notes, as plain text. A GitHub release carries its markdown Body and publish Date
// instead of Notes.
type Changelog struct {
	Version string   `json:"version"`
	Date    string   `json:"date,omitempty"`
	Notes   []string `json:"notes"`
	Body    string   `json:"body,omitempty"`
}

// Changelogs returns a mod's limit newest changelog versions. The versions are object keys, which a map would
// lose along with duplicates, so the object is read token by token; a mod with none answers [].
func (c *Client) Changelogs(ctx context.Context, modID, limit int) ([]Changelog, error) {
	var raw json.RawMessage
	if err := c.get(ctx, fmt.Sprintf("/v1/games/%s/mods/%d/changelogs.json", Game, modID), false, &raw); err != nil {
		return nil, err
	}
	return parseChangelogs(raw, limit)
}

func parseChangelogs(raw []byte, limit int) ([]Changelog, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, nil
	}
	var all []Changelog
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var notes []string
		if err := dec.Decode(&notes); err != nil {
			return nil, err
		}
		for i, n := range notes {
			notes[i] = html.UnescapeString(n)
		}
		if v, ok := key.(string); ok && len(notes) > 0 {
			all = append(all, Changelog{Version: v, Notes: notes})
		}
	}
	// Nexus's key order is not reliably chronological, so versions are ordered by value; one that is not a
	// version keeps its place after the ones that are.
	slices.Reverse(all)
	slices.SortStableFunc(all, func(a, b Changelog) int {
		c, ok := meta.CompareVersions(b.Version, a.Version)
		if !ok {
			_, okA := meta.CompareVersions(a.Version, a.Version)
			_, okB := meta.CompareVersions(b.Version, b.Version)
			return boolRank(okB) - boolRank(okA)
		}
		return c
	})
	return all[:min(limit, len(all))], nil
}

// Categories maps the game's category IDs to their names.
func (c *Client) Categories(ctx context.Context) (map[int]string, error) {
	var raw struct {
		Categories []struct {
			ID   int    `json:"category_id"`
			Name string `json:"name"`
		} `json:"categories"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/games/%s.json", Game), false, &raw); err != nil {
		return nil, err
	}
	out := make(map[int]string, len(raw.Categories))
	for _, cat := range raw.Categories {
		out[cat.ID] = cat.Name
	}
	return out, nil
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}
