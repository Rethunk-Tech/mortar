package pack

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	body := codePrefix + "\n" + base64.StdEncoding.EncodeToString(z)
	base := cmp.Or(c.URL, BaseURL)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/experimental/legacyprofile/create/", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", source.UserAgentWithSite(""))
	client := cmp.Or(c.HTTP, http.DefaultClient)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		return "", errors.New("profile is too large to export as a code")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("thunderstore answered %s", resp.Status)
	}
	var out struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(safefs.CapReader(resp.Body, 1<<20)).Decode(&out); err != nil || out.Key == "" {
		return "", errors.New("thunderstore returned no profile key")
	}
	return out.Key, nil
}

// WriteR2z writes d as an r2modman .r2z file at dest. It does not post a public code.
func WriteR2z(dest string, d Draft) error {
	z, err := ExportZip(d)
	if err != nil {
		return err
	}
	return safefs.WriteFile(dest, z, 0o600)
}
