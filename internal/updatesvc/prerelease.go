package updatesvc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

const mortarRepo = "Rethunk-AI/mortar"

type prereleaseProvider struct {
	client *http.Client
}

func newPrereleaseProvider() *prereleaseProvider {
	return &prereleaseProvider{client: &http.Client{Timeout: 30 * time.Second}}
}

func (p *prereleaseProvider) Name() string { return "prerelease" }

func (p *prereleaseProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	tag, err := p.latestPrereleaseTag(ctx, req.CurrentVersion)
	if err != nil || tag == "" {
		return nil, err
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/manifest.json", mortarRepo, tag)
	ep, err := endpoint.New(endpoint.Config{URL: url})
	if err != nil {
		return nil, err
	}
	return ep.Check(ctx, req)
}

func (p *prereleaseProvider) Download(ctx context.Context, rel *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	if rel == nil {
		return fmt.Errorf("prerelease: missing release")
	}
	tag := "v" + rel.Version
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/manifest.json", mortarRepo, tag)
	ep, err := endpoint.New(endpoint.Config{URL: url})
	if err != nil {
		return err
	}
	return ep.Download(ctx, rel, dst, onProgress)
}

func (p *prereleaseProvider) latestPrereleaseTag(ctx context.Context, current string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+mortarRepo+"/releases?per_page=30", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}
	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	for _, r := range releases {
		if r.Draft || !r.Prerelease {
			continue
		}
		v := strings.TrimPrefix(strings.TrimSpace(r.TagName), "v")
		if meta.Newer(v, current) {
			return strings.TrimSpace(r.TagName), nil
		}
	}
	return "", nil
}
