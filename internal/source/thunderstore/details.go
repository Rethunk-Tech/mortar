package thunderstore

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// errNotFound is the site's 404, which for a README or CHANGELOG means the version has none.
var errNotFound = errors.New("thunderstore answered 404 Not Found")

// Details reads a package's versions, dependencies and categories from the community index, and its newest version's
// README and CHANGELOG from the site.
func (d Driver) Details(ctx context.Context, key, id, mortarVersion string) (source.Details, error) {
	ua := source.UserAgentWithSite(mortarVersion)
	pk, err := d.packages(ctx, key, ua)
	if err != nil {
		return source.Details{}, err
	}
	owner, name, _ := strings.Cut(id, "-")
	for _, p := range pk {
		if !strings.EqualFold(p.Owner, owner) || !strings.EqualFold(p.Name, name) || len(p.Versions) == 0 {
			continue
		}
		out := source.Details{Versions: []string{}, Dependencies: []string{}, Categories: p.Categories, Donation: p.Donation}
		for _, v := range p.Versions {
			out.Versions = append(out.Versions, v.Number)
			out.Released = append(out.Released, v.Created)
		}
		out.Dependencies = append(out.Dependencies, p.depsOf(p.Versions[0])...)
		latest := p.Versions[0].Number
		// The site answers each document from its origin the first time, so the two are asked together.
		var readmeErr, changelogErr error
		var wg sync.WaitGroup
		wg.Go(func() { out.Description, readmeErr = d.doc(ctx, p, latest, "readme", ua) })
		wg.Go(func() { out.Changelog, changelogErr = d.doc(ctx, p, latest, "changelog", ua) })
		wg.Wait()
		for _, err := range []error{readmeErr, changelogErr} {
			if err != nil && !errors.Is(err, errNotFound) {
				return source.Details{}, err
			}
		}
		if out.Categories == nil {
			out.Categories = []string{}
		}
		return out, nil
	}
	return source.Details{}, fmt.Errorf("%s is not in the %s index", id, key)
}

// doc is one version's README or CHANGELOG as Markdown; errNotFound when the version has none.
func (d Driver) doc(ctx context.Context, p pkg, ver, kind, ua string) (string, error) {
	base := cmp.Or(d.URL, BaseURL)
	u := base + "/api/experimental/package/" + url.PathEscape(p.Owner) + "/" + url.PathEscape(p.Name) + "/" +
		url.PathEscape(ver) + "/" + kind + "/"
	var body struct {
		Markdown string `json:"markdown"`
	}
	if _, err := d.getJSON(ctx, u, ua, &body); err != nil {
		return "", err
	}
	return body.Markdown, nil
}

// Changelog is the CHANGELOG of the version an update lands on: authors keep one file that covers every release, so
// the versions in between add nothing. A version without a CHANGELOG answers no notes.
func (d Driver) Changelog(ctx context.Context, key, id, _, latest, mortarVersion string) ([]source.Note, error) {
	ua := source.UserAgentWithSite(mortarVersion)
	pk, err := d.packages(ctx, key, ua)
	if err != nil {
		return nil, err
	}
	owner, name, _ := strings.Cut(id, "-")
	for _, p := range pk {
		if !strings.EqualFold(p.Owner, owner) || !strings.EqualFold(p.Name, name) || len(p.Versions) == 0 {
			continue
		}
		ver := p.Versions[0].Number
		for _, v := range p.Versions {
			if v.Number == latest {
				ver = latest
			}
		}
		body, err := d.doc(ctx, p, ver, "changelog", ua)
		if errors.Is(err, errNotFound) || (err == nil && strings.TrimSpace(body) == "") {
			return []source.Note{}, nil
		}
		if err != nil {
			return nil, err
		}
		return []source.Note{{Version: ver, Body: strings.TrimSpace(body)}}, nil
	}
	return nil, fmt.Errorf("%s is not in the %s index", id, key)
}

// ReportTarget is the package's GitHub repository when its website links one, else its page on the site.
func (d Driver) ReportTarget(ctx context.Context, key, id, mortarVersion string) (source.ReportTarget, error) {
	items, err := d.Items(ctx, key, mortarVersion, []string{id})
	if err != nil {
		return source.ReportTarget{}, err
	}
	it, ok := items[strings.ToLower(id)]
	if !ok {
		return source.ReportTarget{}, fmt.Errorf("%s is not in the %s index", id, key)
	}
	return source.ReportTarget{Repo: it.Repo, URL: it.URL}, nil
}
