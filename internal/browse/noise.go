package browse

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// noiseWords start the words of category names that Browse excludes by default: modpacks, translations, and cheats
// or hacks. Matching is by word prefix and case-insensitive, so it covers each source's own spelling ("Modpacks",
// "Translation", "Language", "Cheats and God items", "Hacks").
var noiseWords = []string{"modpack", "translation", "language", "cheat", "hack"}

func noisy(name string) bool {
	for _, w := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return r == ' ' || r == '-' || r == '_' || r == '/' || r == '&' }) {
		for _, p := range noiseWords {
			if strings.HasPrefix(w, p) {
				return true
			}
		}
	}
	return false
}

func noisyNames(names []string) []string {
	return slices.DeleteFunc(slices.Clone(names), func(n string) bool { return !noisy(n) })
}

// categoriesCached is the source's category names for the game, reused for categoryTTL; a source without categories
// gives none, and a failed lookup gives none and its error.
func categoriesCached(ctx context.Context, game string, s source.Source, key string) ([]string, error) {
	cat, ok := s.(source.Categorizer)
	if !ok {
		return nil, nil
	}
	id := game + "|" + s.ID()
	catMu.Lock()
	if e, hit := catCache[id]; hit && time.Now().Before(e.until) {
		catMu.Unlock()
		return e.names, nil
	}
	// Concurrent misses for one source share a single lookup.
	if call, busy := catFlight[id]; busy {
		catMu.Unlock()
		select {
		case <-call.done:
			return call.names, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &catCall{done: make(chan struct{})}
	catFlight[id] = call
	catMu.Unlock()

	var keep bool
	call.names, keep, call.err = lookupCategories(ctx, game, s, cat, key)
	catMu.Lock()
	delete(catFlight, id)
	if keep {
		catCache[id] = catEntry{names: call.names, until: time.Now().Add(categoryTTL)}
	}
	catMu.Unlock()
	close(call.done)
	return call.names, call.err
}

type catCall struct {
	done  chan struct{}
	names []string
	err   error
}

func lookupCategories(ctx context.Context, game string, s source.Source, cat source.Categorizer, key string) (names []string, keep bool, err error) {
	names, err = cat.Categories(ctx, key)
	// Signed out, the source simply has no categories to offer; nothing is kept, so signing in shows them.
	if errors.Is(err, source.ErrSignedOut) {
		return nil, false, nil
	}
	if err != nil {
		slog.Warn("could not list categories", "game", game, "source", s.ID(), "err", err)
		return nil, false, err
	}
	return names, true, nil
}
