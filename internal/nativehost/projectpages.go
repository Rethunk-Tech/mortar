package nativehost

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
	"github.com/Rethunk-Tech/mortar/internal/source/modrinth"
	"github.com/Rethunk-Tech/mortar/internal/source/vsmoddb"
)

// The browser pages name a mod by slug (CurseForge) or alias (Mod DB), which a profile entry does not keep, so the
// site is asked once per project and the answer is cached on disk. A lookup runs in the background: the reply that
// triggers it lacks the slug and a later page load has it.
const pageLookupTimeout = 2 * time.Minute

var (
	pagesMu       sync.Mutex
	pagesFilling  = map[string]bool{}
	pagesCacheRel = filepath.Join("cache", "extension-project-pages.json")
)

func readPages(dir string) map[string]string {
	pages := map[string]string{}
	if b, err := fsx.ReadFile(filepath.Join(dir, pagesCacheRel)); err == nil {
		_ = json.Unmarshal(b, &pages)
	}
	return pages
}

// slugOf is the page address's last segment, or "" for a Mod DB mod that has no alias (its page is /show/mod/<id>).
func slugOf(src, page string) string {
	u, err := url.Parse(page)
	if err != nil || (src == "vsmoddb" && strings.HasPrefix(u.Path, "/show/mod/")) {
		return ""
	}
	return path.Base(strings.TrimRight(u.Path, "/"))
}

// fetchPage asks the site for one project's page address; id is what the site's driver takes.
func fetchPage(ctx context.Context, src, id string) (string, error) {
	switch src {
	case "curseforge":
		return curseforge.Driver{}.PageURL(ctx, id)
	case "modrinth":
		return modrinth.Driver{}.PageURL(ctx, id)
	}
	items, err := vsmoddb.Driver{}.Items(ctx, "", "", []string{id})
	for _, it := range items {
		return it.URL, err
	}
	return "", err
}

// withSlugs fills in the cached slug of each project and starts a background lookup for the rest. ids are the
// site-driver ids, parallel to projects.
func withSlugs(src string, projects []hostProject, ids []string) {
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	pages := readPages(dir)
	var missing []string
	for i := range projects {
		if page, ok := pages[src+":"+ids[i]]; ok {
			projects[i].Slug = slugOf(src, page)
		} else {
			missing = append(missing, ids[i])
		}
	}
	pagesMu.Lock()
	defer pagesMu.Unlock()
	var todo []string
	for _, id := range missing {
		if !pagesFilling[src+":"+id] {
			pagesFilling[src+":"+id] = true
			todo = append(todo, id)
		}
	}
	if len(todo) > 0 {
		go fillPages(dir, src, todo)
	}
}

func fillPages(dir, src string, ids []string) {
	ctx, cancel := context.WithTimeout(context.Background(), pageLookupTimeout)
	defer cancel()
	for _, id := range ids {
		page, err := fetchPage(ctx, src, id)
		if err == nil && page != "" {
			pagesMu.Lock()
			pages := readPages(dir)
			pages[src+":"+id] = page
			if b, err := json.Marshal(pages); err == nil {
				_ = os.MkdirAll(filepath.Join(dir, "cache"), 0o700)
				_ = fsx.WriteFile(filepath.Join(dir, pagesCacheRel), b, 0o600)
			}
			pagesMu.Unlock()
		}
		pagesMu.Lock()
		delete(pagesFilling, src+":"+id)
		pagesMu.Unlock()
	}
}
