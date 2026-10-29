package meta

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
)

const (
	compatTTL       = 24 * time.Hour
	CompatCacheFile = "smapi-compat.json"
)

// compatMemo keeps the parsed cache file, so asking again does not re-parse megabytes of JSON while the file on
// disk is unchanged.
type compatMemo struct {
	mu    sync.Mutex
	stamp os.FileInfo
	index smapiweb.CompatIndex
}

// CompatList returns the SMAPI compatibility list, cached a day.
func (c *Client) CompatList(ctx context.Context) (smapiweb.CompatIndex, error) {
	path := c.Cache().Path(CompatCacheFile)
	stamp := func() os.FileInfo {
		if path == "" {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil
		}
		return info
	}
	if now := stamp(); now != nil && c.young(now.ModTime(), compatTTL) {
		c.compat.mu.Lock()
		memo, index := c.compat.stamp, c.compat.index
		c.compat.mu.Unlock()
		if memo != nil && memo.Size() == now.Size() && memo.ModTime().Equal(now.ModTime()) {
			return index, nil
		}
	}
	index, err := diskcache.CachedStamped(c.Cache(), CompatCacheFile, appversion.Build(), compatTTL, func() (smapiweb.CompatIndex, error) {
		return c.web().CompatList(ctx)
	})
	if err == nil {
		c.compat.mu.Lock()
		c.compat.stamp, c.compat.index = stamp(), index
		c.compat.mu.Unlock()
	}
	return index, err
}
