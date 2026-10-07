package problems

import (
	"context"
	"errors"
	"regexp"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
)

// cfResolver is the CurseForge driver's file lookup, which also refuses a project that forbids outside downloads.
type cfResolver interface {
	Unavailable() string
	Resolve(ctx context.Context, id, version string) (curseforge.Resolved, error)
}

var versionToken = regexp.MustCompile(`\d+(?:\.\d+)+`)

// cfProject is the CurseForge project of the update keys, "" when none names one.
func cfProject(keys []string) string {
	for _, k := range keys {
		if id, ok := manifest.CurseForgeUpdateKey(k); ok {
			return id
		}
	}
	return ""
}

// cfFileIsNewer decides whether the CurseForge file named name is newer than installed. A name that carries a version
// number is compared by it; one that does not is taken as newer only when SMAPI already reported suggested above installed.
func cfFileIsNewer(name, installed, suggested string) bool {
	if v := versionToken.FindString(name); v != "" {
		c, ok := meta.CompareVersions(v, installed)
		return ok && c > 0
	}
	c, ok := meta.CompareVersions(suggested, installed)
	return ok && c > 0
}

// curseforgeSwitches turns the updates SMAPI suggests from a mod's CurseForge update key, which name no file, into
// updates of the newest stable CurseForge file. An update whose file is not newer than the installed one is dropped, and
// one the project forbids outside CurseForge is kept for the review as NotDistributable. A mod installed from another
// site is marked Switch. When CurseForge cannot be asked the updates stay as they were.
func (s *Service) curseforgeSwitches(ctx context.Context, mods []framework.Mod, updates []Update) []Update {
	entry, ok := source.Get(profile.KindCurseForge)
	driver, canResolve := entry.Source.(cfResolver)
	if !ok || !canResolve || driver.Unavailable() != "" {
		return updates
	}
	return s.switchUpdates(ctx, driver, mods, updates)
}

func (s *Service) switchUpdates(ctx context.Context, driver cfResolver, mods []framework.Mod, updates []Update) []Update {
	byKey := make(map[string]framework.Mod, len(mods))
	for _, m := range mods {
		byKey[m.Key+"\x00"+string(m.ModID())] = m
	}
	out := make([]Update, 0, len(updates))
	for _, u := range updates {
		x, found := byKey[u.Key+"\x00"+string(u.ID)]
		project := cfProject(x.UpdateKeys)
		if !found || project == "" || u.Source != "CurseForge" || u.Package != "" || u.GitHubRepo != "" || u.NexusID != 0 {
			out = append(out, u)
			continue
		}
		release := func() {}
		if s.Throttle != nil {
			r, err := s.Throttle(ctx, profile.KindCurseForge)
			if err != nil {
				out = append(out, u)
				continue
			}
			release = r
		}
		res, err := driver.Resolve(ctx, project, "")
		release()
		var manual *curseforge.NotDistributableError
		switch {
		case errors.As(err, &manual):
			u.NotDistributable = true
		case err != nil:
			out = append(out, u)
			continue
		case !cfFileIsNewer(res.Version, u.Installed, u.Version):
			continue
		default:
			u.Package, u.PackageSource, u.PackageVersion = project, profile.KindCurseForge, res.VersionID
			if v := versionToken.FindString(res.Version); v != "" {
				u.Version = v
			}
		}
		if x.SourceKind != profile.KindCurseForge {
			u.Switch, u.FromSource = true, sourceLabel(x.SourceKind)
		}
		out = append(out, u)
	}
	return out
}

// sourceLabel is the site's name for a source id.
func sourceLabel(kind string) string {
	if e, ok := source.Get(kind); ok {
		return e.Source.Name()
	}
	return kind
}
