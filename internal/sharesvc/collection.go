package sharesvc

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/share"
)

func parseCollectionURL(text string) (domain, slug string, revision int, ok bool) {
	u, err := url.Parse(strings.TrimSpace(text))
	if err != nil || u.Scheme != "https" {
		return "", "", 0, false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if host != "nexusmods.com" && host != "next.nexusmods.com" {
		return "", "", 0, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[0] != "games" || parts[2] != "collections" || parts[1] == "" || parts[3] == "" {
		return "", "", 0, false
	}
	domain, slug = parts[1], parts[3]
	switch len(parts) {
	case 4:
		return domain, slug, 0, true
	case 6:
		if parts[4] != "revisions" {
			return "", "", 0, false
		}
		n, err := strconv.Atoi(parts[5])
		if err != nil || n < 1 {
			return "", "", 0, false
		}
		return domain, slug, n, true
	default:
		return "", "", 0, false
	}
}

func (s *Service) previewCollection(ctx context.Context, game, domain, slug string, revision int, profileID string) (Preview, error) {
	if err := checkDomain(game, domain, "collection"); err != nil {
		return Preview{}, err
	}
	col, err := s.d.Meta.Collection(ctx, domain, slug, revision)
	if err != nil {
		return Preview{}, err
	}
	entries := make([]share.Ref, 0, len(col.Files))
	for _, f := range col.Files {
		entries = append(entries, share.Ref{ModID: f.ModID, FileID: f.FileID})
	}
	pv, err := s.preview(ctx, game, share.Shared{Name: col.Name, Entries: entries}, "", nil, profileID, profile.OriginCollection)
	if err != nil {
		return Preview{}, err
	}
	s.mu.Lock()
	if s.current != nil && s.current.id == pv.Session {
		s.current.collection = &profile.CollectionRef{
			Domain: domain, Slug: slug, Name: col.Name, Revision: col.Revision,
		}
	}
	s.mu.Unlock()
	return pv, nil
}

// CollectionStatus is whether a profile was imported from a Nexus collection, and whether a newer revision exists.
type CollectionStatus struct {
	Linked   bool   `json:"linked"`
	Name     string `json:"name"`
	Revision int    `json:"revision"`
	Latest   int    `json:"latest"`
	URL      string `json:"url"`
}

func collectionURL(domain, slug string) string {
	return "https://www.nexusmods.com/games/" + domain + "/collections/" + slug
}

// CollectionStatus reports the stored collection revision and the latest one. A Nexus error leaves Latest 0.
func (s *Service) CollectionStatus(ctx context.Context, game, profileID string) (CollectionStatus, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return CollectionStatus{}, err
	}
	if p.Collection == nil || p.Collection.Domain == "" || p.Collection.Slug == "" {
		return CollectionStatus{}, nil
	}
	st := CollectionStatus{
		Linked:   true,
		Name:     p.Collection.Name,
		Revision: p.Collection.Revision,
		URL:      collectionURL(p.Collection.Domain, p.Collection.Slug),
	}
	col, metaErr := s.d.Meta.Collection(ctx, p.Collection.Domain, p.Collection.Slug, 0)
	if metaErr == nil {
		st.Latest = col.Revision
		if col.Name != "" {
			st.Name = col.Name
		}
	}
	return st, nil
}

// PreviewCollectionUpdate previews the latest revision of the profile's collection against that profile.
func (s *Service) PreviewCollectionUpdate(ctx context.Context, game, profileID string) (Preview, error) {
	p, err := s.find(game, profileID)
	if err != nil {
		return Preview{}, err
	}
	if p.Collection == nil || p.Collection.Domain == "" || p.Collection.Slug == "" {
		return Preview{}, fmt.Errorf("this profile is not from a collection")
	}
	return s.previewCollection(ctx, game, p.Collection.Domain, p.Collection.Slug, 0, profileID)
}

// checkDomain refuses a Nexus link whose game domain is not this game's; what names the link ("collection", "mod")
// for the message.
func checkDomain(game, domain, what string) error {
	info, ok := components.BundledGame(game)
	name, want := game, ""
	if ok {
		name, want = info.Name, info.Nexus.Domain
	}
	if want == "" || !strings.EqualFold(want, domain) {
		return fmt.Errorf("that %s is not for %s", what, name)
	}
	return nil
}
