package nexussvc

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

// PageName is the cache file under cache/ for a Nexus mod's page data from the batched lookup.
func PageName(domain string, modID int) string {
	return fmt.Sprintf("%sv2-%s-%d.json", meta.NexusPagePrefix, domain, modID)
}

// absentName marks a mod the batched lookup asked Nexus for and got nothing back (hidden, removed, a wrong id), so
// the next look within the details TTL does not ask again.
func absentName(domain string, modID int) string {
	return fmt.Sprintf("%sabsent-v1-%s-%d.json", meta.NexusPagePrefix, domain, modID)
}

type absent struct{}

// Primed is what PrimeDetails holds for the mods it was asked about. Error says why the rest could not be fetched
// (offline, rate limited, signed out), "" when nothing failed; the details found are kept either way.
type Primed struct {
	Details map[int]Details `json:"details"`
	Error   string          `json:"error,omitempty"`
}

// PrimeDetails is Prime for the window, which keeps what arrived when the rest could not be fetched.
func (s *Service) PrimeDetails(ctx context.Context, gameID string, modIDs []int) (Primed, error) {
	if _, err := game.NexusTitle(gameID); err != nil {
		return Primed{}, err
	}
	d, err := s.Prime(ctx, gameID, modIDs)
	out := Primed{Details: d}
	if err != nil {
		out.Error = err.Error()
	}
	return out, nil
}

// Prime fills the page data of many mods at once: whatever is cached and fresh is kept, and the rest comes
// from one GraphQL request per 50 mods, so a list of hundreds costs a handful of requests instead of several each.
// It returns the best details held for each mod, which are partial (the page's headline data, no files or
// changelogs) unless the full details were cached; Details fills in the rest when a mod is opened. Signed out, it
// asks without a key. A rate limit ends the lookup and returns what the cache has with the error.
//
//wails:ignore
func (s *Service) Prime(ctx context.Context, gameID string, modIDs []int) (map[int]Details, error) {
	t, err := game.NexusTitle(gameID)
	if err != nil {
		return nil, err
	}
	var missing []int
	seen := map[int]bool{}
	for _, id := range modIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		if !meta.Fresh[Details](s.meta, DetailsName(t.Domain, id), detailsTTL) && !meta.Fresh[Details](s.meta, PageName(t.Domain, id), detailsTTL) &&
			!meta.Fresh[absent](s.meta, absentName(t.Domain, id), detailsTTL) {
			missing = append(missing, id)
		}
	}
	var fetchErr error
	if len(missing) > 0 {
		fetchErr = s.fetchPages(ctx, t.Domain, missing)
	}
	out := make(map[int]Details, len(seen))
	for id := range seen {
		if d, ok := PeekDetails(s.meta, t.Domain, id); ok {
			out[id] = d
		}
	}
	return out, fetchErr
}

// PeekDetails reads the page data Mortar keeps for a mod without fetching: its full details, else the batched page.
func PeekDetails(c *meta.Client, domain string, id int) (Details, bool) {
	if d, ok := meta.Peek[Details](c, DetailsName(domain, id)); ok {
		return d, true
	}
	return meta.Peek[Details](c, PageName(domain, id))
}

func (s *Service) fetchPages(ctx context.Context, domain string, ids []int) error {
	// The page data is public: signed out, the same request goes without a key, so requirements still show.
	c, err := Authed(s.store, s.client)
	if errors.Is(err, ErrSignedOut) {
		c, err = s.client, nil
	}
	if err != nil {
		return err
	}
	infos, err := c.ModsByDomain(ctx, domain, ids)
	if err != nil {
		log.Printf("nexus pages: %d of %d %s mods, then %v", len(infos), len(ids), domain, err)
	} else {
		log.Printf("nexus pages: %d of %d %s mods in %d requests", len(infos), len(ids), domain, nexus.Requests(len(ids)))
	}
	for id, info := range infos {
		meta.Put(s.meta, PageName(domain, id), Details{Page: info.Page(), Category: info.Category, Partial: true})
	}
	if err == nil {
		for _, id := range ids {
			if _, ok := infos[id]; !ok {
				meta.Put(s.meta, absentName(domain, id), absent{})
			}
		}
	}
	return err
}
