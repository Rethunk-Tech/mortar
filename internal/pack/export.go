package pack

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-modmanagers/r2modman"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// ExportZip is the .r2z body: export.r2x plus the draft's config files.
func ExportZip(d Draft) ([]byte, error) {
	e := r2modman.Export{ProfileName: d.Name, Mods: make([]r2modman.Mod, 0, len(d.Packages)), Files: make(map[string][]byte, len(d.Configs))}
	for _, p := range d.Packages {
		if p.Source != thunderstore {
			return nil, fmt.Errorf("%s package %s cannot go in an r2modman code", p.Source, p.Native)
		}
		ver, err := r2modman.ParseVersion(p.Version)
		if err != nil {
			return nil, fmt.Errorf("%s has version %q, not major.minor.patch", p.Native, p.Version)
		}
		e.Mods = append(e.Mods, r2modman.Mod{Name: p.Native, Version: ver, Enabled: !p.Disabled})
	}
	for _, f := range d.Configs {
		e.Files[f.Path] = f.Data
	}
	return r2modman.WriteExport(e)
}

// ExportCode stores d on Thunderstore as an r2modman profile code and returns its key. It publishes the profile's
// mod list and config files to a public service, so the caller must have the player's confirmation first.
func (c Code) ExportCode(ctx context.Context, d Draft) (string, error) {
	z, err := ExportZip(d)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	hc := cmp.Or(c.HTTP, http.DefaultClient)
	client := *hc
	client.Transport = &httpkit.Transport{Base: hc.Transport, UserAgent: source.UserAgentWithSite("")}
	return r2modman.CreateCode(ctx, &client, cmp.Or(c.URL, BaseURL), z)
}

// WriteR2z writes d as an r2modman .r2z file at dest. It does not post a public code.
func WriteR2z(dest string, d Draft) error {
	z, err := ExportZip(d)
	if err != nil {
		return err
	}
	return safefs.WriteFile(dest, z, 0o600)
}
