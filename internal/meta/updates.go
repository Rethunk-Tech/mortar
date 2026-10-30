package meta

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
)

const (
	updatesTTL     = time.Hour
	updatesFile    = "smapi-updates.json"
	updatesBackoff = 5 * time.Minute
	// updatesRefreshTimeout bounds a background refresh, which nobody waits on.
	updatesRefreshTimeout = 2 * time.Minute
)

// UpdateRequest describes the installed environment and mods to check.
type UpdateRequest struct {
	APIVersion  string
	GameVersion string
	Platform    string // Android, Linux, Mac or Windows
	Mods        []smapiweb.InstalledMod
	// Fresh asks the API even for answers still young enough to come from cache, for an explicit check.
	Fresh bool
}

func (r UpdateRequest) key(m smapiweb.InstalledMod) string {
	return strings.ToLower(strings.Join([]string{m.ID, strings.Join(m.UpdateKeys, ","), m.Version, r.GameVersion, r.APIVersion, r.Platform}, "|"))
}

// CheckUpdates returns one result per requested mod, in order. Answers younger than an hour come from cache. An older
// answer this build cached is served at once and asked again in the background, so an unreachable smapi.io never
// holds up a check that has something to show; mods with no answer, or every mod on an explicit Fresh check, are
// asked in batches, and a batch that fails falls back to stale cache, else unknown.
func (c *Client) CheckUpdates(ctx context.Context, req UpdateRequest) []smapiweb.UpdateResult {
	c.updatesMu.Lock()
	defer c.updatesMu.Unlock()
	cache := c.Cache()
	var store map[string]diskcache.Entry[smapiweb.UpdateResult]
	if e, ok := diskcache.Peek[map[string]diskcache.Entry[smapiweb.UpdateResult]](cache, updatesFile); ok {
		store = e.Value
		// An answer carries the build that fetched it; one written before answers carried it takes the file's.
		for k, v := range store {
			if v.Stamp == "" {
				v.Stamp = e.Stamp
				store[k] = v
			}
		}
	}
	if store == nil {
		store = map[string]diskcache.Entry[smapiweb.UpdateResult]{}
	}
	now := c.now()
	build := appversion.Build()
	out := make([]smapiweb.UpdateResult, len(req.Mods))
	var stale []int
	var later []smapiweb.InstalledMod
	for i, m := range req.Mods {
		e, ok := store[req.key(m)]
		switch {
		// Another build's answers are asked now, and serve only while the API cannot be reached.
		case ok && !req.Fresh && e.Stamp == build && c.young(e.Fetched, updatesTTL):
			out[i] = e.Value
		case ok && !req.Fresh && e.Stamp == build && e.Value.Known:
			out[i] = e.Value
			later = append(later, req.Mods[i])
		default:
			stale = append(stale, i)
		}
	}
	defer c.refreshLater(ctx, req, later)
	// A failed ask pauses asking: the stale answer, else unknown, is served until the pause ends.
	var got []smapiweb.UpdateResult
	if len(stale) > 0 && !now.Before(c.updatesPause) {
		asked := make([]smapiweb.InstalledMod, len(stale))
		for j, i := range stale {
			asked[j] = req.Mods[i]
		}
		var err error
		got, err = c.web().Updates(ctx, smapiweb.UpdateRequest{APIVersion: req.APIVersion, GameVersion: req.GameVersion, Platform: req.Platform, Mods: asked})
		// The window's offline banner then names smapi.io, as it does any source that stops answering.
		netstate.Record("smapi", smapiOutcome(got, err))
		if err != nil {
			unknown := 0
			for _, r := range got {
				if !r.Known {
					unknown++
				}
			}
			log.Printf("updates: smapi.io did not answer for %d of %d mods, pausing update checks for %s: %v", unknown, len(asked), updatesBackoff, err)
			c.updatesPause = now.Add(updatesBackoff)
		}
	}
	dirty := false
	for j, i := range stale {
		k := req.key(req.Mods[i])
		switch {
		case got != nil && got[j].Known:
			out[i] = got[j]
			store[k] = diskcache.Entry[smapiweb.UpdateResult]{Fetched: now, Value: out[i], Stamp: build}
			dirty = true
		case store[k].Value.Known:
			out[i] = store[k].Value
		default:
			out[i] = smapiweb.UpdateResult{ID: req.Mods[i].ID}
		}
	}
	if dirty {
		for k, e := range store {
			if age := now.Sub(e.Fetched); age < 0 || age > 24*updatesTTL {
				delete(store, k)
			}
		}
		_ = diskcache.PutEntry(cache, updatesFile, diskcache.Entry[map[string]diskcache.Entry[smapiweb.UpdateResult]]{Fetched: now, Value: store, Stamp: build})
	}
	return out
}

// refreshLater asks smapi.io about mods again in the background, one refresh at a time; the next check reads what it
// cached. It outlives the check that started it, so it keeps ctx's values but not its cancellation.
func (c *Client) refreshLater(ctx context.Context, req UpdateRequest, mods []smapiweb.InstalledMod) {
	if len(mods) == 0 || !c.updatesRefreshing.CompareAndSwap(false, true) {
		return
	}
	req.Mods, req.Fresh = mods, true
	go func() {
		defer c.updatesRefreshing.Store(false)
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), updatesRefreshTimeout)
		defer cancel()
		c.CheckUpdates(ctx, req)
	}()
}

// smapiOutcome is what the offline banner learns from one check: smapi.io answered when any mod came back known,
// otherwise the first failed batch's error, which is always a real failure. Not the joined error: one cancelled batch
// beside a DNS failure would hide the network error, and the banner's tooltip shows the text on a single line.
func smapiOutcome(got []smapiweb.UpdateResult, err error) error {
	if err == nil {
		return nil
	}
	for _, r := range got {
		if r.Known {
			return nil
		}
	}
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		if errs := j.Unwrap(); len(errs) > 0 {
			return errs[0]
		}
	}
	return err
}
