package pack

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"go.yaml.in/yaml/v3"
)

type exportMod struct {
	Name    string    `yaml:"name"`
	Version r2Version `yaml:"version"`
	Enabled bool      `yaml:"enabled"`
}

type exportDoc struct {
	Name string      `yaml:"profileName"`
	Mods []exportMod `yaml:"mods"`
}

// ExportZip is the .r2z body: export.r2x plus the draft's config files.
func ExportZip(d Draft) ([]byte, error) {
	doc := exportDoc{Name: d.Name, Mods: make([]exportMod, 0, len(d.Packages))}
	for _, p := range d.Packages {
		if p.Source != thunderstore {
			return nil, fmt.Errorf("%s package %s cannot go in an r2modman code", p.Source, p.Native)
		}
		m := exportMod{Name: p.Native, Enabled: !p.Disabled}
		parts := strings.Split(p.Version, ".")
		nums := []*int{&m.Version.Major, &m.Version.Minor, &m.Version.Patch}
		if len(parts) != len(nums) {
			return nil, fmt.Errorf("%s has version %q, not major.minor.patch", p.Native, p.Version)
		}
		for i, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("%s has version %q, not major.minor.patch", p.Native, p.Version)
			}
			*nums[i] = n
		}
		doc.Mods = append(doc.Mods, m)
	}
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return Zip(append([]File{{Path: exportFile, Data: raw}}, d.Configs...))
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
