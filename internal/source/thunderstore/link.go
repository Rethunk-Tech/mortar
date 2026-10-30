package thunderstore

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Schemes is the one-click install link scheme.
func (Driver) Schemes() []string { return []string{tstore.Scheme} }

// HandleLinksDefault is false: Mortar takes ror2mm links from the system only when the user opts in.
func (Driver) HandleLinksDefault() bool { return false }

// Resolved is what an install needs of one package version.
type Resolved struct {
	Namespace, Name string
	Version         string
	// URL is the download address.
	URL  string
	Size int64
	// Dependencies are Namespace-Name-Version strings.
	Dependencies []string
	// Missing lists the "Namespace-Name" dependencies the index no longer has, which Closure left out.
	Missing []string
	Look
}

// Resolve finds the package in the community's index; an empty version means the latest.
func (d Driver) Resolve(ctx context.Context, key, namespace, name, ver, mortarVersion string) (Resolved, error) {
	pk, err := d.packages(ctx, key, mortarVersion)
	if err != nil {
		return Resolved{}, err
	}
	for _, p := range pk {
		if !strings.EqualFold(p.Owner, namespace) || !strings.EqualFold(p.Name, name) {
			continue
		}
		for _, v := range p.Versions {
			if ver == "" || v.Number == ver {
				return d.resolved(p, v), nil
			}
		}
	}
	return Resolved{}, fmt.Errorf("%s-%s %s is not in the %s index", namespace, name, ver, key)
}

// Publisher is the namespace of the one package in the community's index with this name and version; ok is false when
// none or several match.
func (d Driver) Publisher(ctx context.Context, key, name, ver, mortarVersion string) (namespace string, ok bool, err error) {
	pk, err := d.packages(ctx, key, mortarVersion)
	if err != nil {
		return "", false, err
	}
	for _, p := range pk {
		if !strings.EqualFold(p.Name, name) || !slices.ContainsFunc(p.Versions, func(v tstore.Version) bool { return v.Number == ver }) {
			continue
		}
		if namespace != "" {
			return "", false, nil
		}
		namespace = p.Owner
	}
	return namespace, namespace != "", nil
}

// Versions lists the package's versions in the community's index, newest first.
func (d Driver) Versions(ctx context.Context, key, namespace, name, mortarVersion string) ([]string, error) {
	c, err := d.client(mortarVersion)
	if err != nil {
		return nil, err
	}
	return c.Versions(ctx, key, namespace, name)
}

// Dependencies reads the dependencies of the listed package versions from the community's index, one package per
// "Namespace-Name" id, with no request beyond the index's own refresh.
func (d Driver) Dependencies(ctx context.Context, key, mortarVersion string, refs []source.VersionRef) (map[source.VersionRef][]string, error) {
	c, err := d.client(mortarVersion)
	if err != nil {
		return nil, err
	}
	x, err := c.Snapshot(ctx, key)
	if err != nil {
		return nil, err
	}
	want := map[string]source.VersionRef{}
	for _, r := range refs {
		want[strings.ToLower(r.ID)+"@"+r.Version] = r
	}
	out := map[source.VersionRef][]string{}
	for _, p := range x.Packages() {
		id := p.Owner + "-" + p.Name
		for _, v := range p.Versions {
			r, ok := want[strings.ToLower(id)+"@"+v.Number]
			if !ok {
				continue
			}
			needs := p.Dependencies(v)
			names := make([]string, 0, len(needs))
			for _, dep := range needs {
				if pkg, _, ok := deps.ThunderstoreRef(dep); ok {
					dep = pkg
				}
				names = append(names, dep)
			}
			out[r] = names
		}
	}
	return out, nil
}

func (d Driver) resolved(p tstore.Package, v tstore.Version) Resolved {
	base := cmp.Or(d.URL, tstore.BaseURL)
	return Resolved{
		Namespace: p.Owner, Name: p.Name, Version: v.Number, Size: v.Size, Dependencies: p.Dependencies(v),
		URL:  base + "/package/download/" + p.Owner + "/" + p.Name + "/" + v.Number + "/",
		Icon: p.Icon, Category: Category(p.Categories),
	}
}
