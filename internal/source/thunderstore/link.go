package thunderstore

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Schemes is the one-click install link scheme.
func (Driver) Schemes() []string { return []string{"ror2mm"} }

// HandleLinksDefault is false: Mortar takes ror2mm links from the system only when the user opts in.
func (Driver) HandleLinksDefault() bool { return false }

// Ref names one package version in a ror2mm link.
type Ref struct {
	Namespace, Name, Version string
}

var (
	part       = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	versionRe  = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	errBadLink = errors.New("not a ror2mm install link")
)

// ParseLink reads ror2mm://v1/install/thunderstore.io/<namespace>/<name>/<version>/.
func ParseLink(raw string) (Ref, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "ror2mm" || u.Host != "v1" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return Ref{}, errBadLink
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "install" || parts[1] != "thunderstore.io" ||
		!part.MatchString(parts[2]) || !part.MatchString(parts[3]) || !versionRe.MatchString(parts[4]) {
		return Ref{}, errBadLink
	}
	return Ref{Namespace: parts[2], Name: parts[3], Version: parts[4]}, nil
}

// Resolved is what an install needs of one package version.
type Resolved struct {
	Namespace, Name string
	Version         string
	// URL is the download address.
	URL  string
	Size int64
	// Dependencies are Namespace-Name-Version strings.
	Dependencies []string
}

// Resolve finds the package in the community's index; an empty version means the latest.
func (d Driver) Resolve(ctx context.Context, key, namespace, name, ver, mortarVersion string) (Resolved, error) {
	pk, err := d.packages(ctx, key, source.UserAgent(mortarVersion)+" (+https://mortar.rethunk.tech)")
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

func (d Driver) resolved(p pkg, v version) Resolved {
	base := d.URL
	if base == "" {
		base = BaseURL
	}
	return Resolved{
		Namespace: p.Owner, Name: p.Name, Version: v.Number, Size: v.Size, Dependencies: v.Deps,
		URL: base + "/package/download/" + p.Owner + "/" + p.Name + "/" + v.Number + "/",
	}
}
