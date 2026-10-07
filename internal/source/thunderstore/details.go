package thunderstore

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// errNotFound is the site's 404, which for a README or CHANGELOG means the version has none.
var errNotFound = errors.New("thunderstore answered 404 Not Found")

// Details reads a package's versions, dependencies and categories from the community index, and its newest version's
// README and CHANGELOG from the site.
func (d Driver) Details(ctx context.Context, key, id, mortarVersion string) (source.Details, error) {
	ua := source.UserAgent(mortarVersion) + " (+https://mortar.rethunk.tech)"
	pk, err := d.packages(ctx, key, ua)
	if err != nil {
		return source.Details{}, err
	}
	owner, name, _ := strings.Cut(id, "-")
	for _, p := range pk {
		if !strings.EqualFold(p.Owner, owner) || !strings.EqualFold(p.Name, name) || len(p.Versions) == 0 {
			continue
		}
		out := source.Details{Versions: []string{}, Dependencies: []string{}, Categories: p.Categories}
		for _, v := range p.Versions {
			out.Versions = append(out.Versions, v.Number)
		}
		out.Dependencies = append(out.Dependencies, p.depsOf(p.Versions[0])...)
		latest := p.Versions[0].Number
		if out.Description, err = d.doc(ctx, p, latest, "readme", ua); err != nil && !errors.Is(err, errNotFound) {
			return source.Details{}, err
		}
		if out.Changelog, err = d.doc(ctx, p, latest, "changelog", ua); err != nil && !errors.Is(err, errNotFound) {
			return source.Details{}, err
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
