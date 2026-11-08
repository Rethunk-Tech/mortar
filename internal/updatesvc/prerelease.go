package updatesvc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

const mortarRepo = "Rethunk-Tech/mortar"

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
	releases, err := gh.FetchReleases(ctx, p.client, "https://api.github.com/repos/"+mortarRepo+"/releases?per_page=30")
	if err != nil {
		return "", fmt.Errorf("github releases: %w", err)
	}
	for _, r := range releases {
		if r.Draft || !r.Prerelease {
			continue
		}
		v := strings.TrimPrefix(strings.TrimSpace(r.Tag), "v")
		if meta.Newer(v, current) {
			return strings.TrimSpace(r.Tag), nil
		}
	}
	return "", nil
}
