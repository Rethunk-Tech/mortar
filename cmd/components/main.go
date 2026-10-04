package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/github"
)

const maxAsset = 2 << 30

type options struct {
	source    string
	output    string
	bundled   string
	signature string
	public    string
}

func main() {
	var o options
	flag.StringVar(&o.source, "source", "components.source.json", "component source file")
	flag.StringVar(&o.output, "output", "components.json", "resolved manifest path")
	flag.StringVar(&o.signature, "signature", "components.json.sig", "detached signature path")
	flag.StringVar(&o.bundled, "bundled", "internal/components/components.json", "embedded manifest path")
	flag.StringVar(&o.public, "public-key", "build/updater/public.key", "Ed25519 public key path")
	flag.Parse()
	if err := run(context.Background(), o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, o options) error {
	source, err := readSource(o.source)
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	if token := firstNonEmpty(os.Getenv("GH_TOKEN"), os.Getenv("GITHUB_TOKEN")); token != "" {
		httpClient.Transport = bearerTransport{token: token, next: http.DefaultTransport}
	}
	cacheDir, err := os.MkdirTemp("", "mortar-components-cache-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()
	client := &github.Client{HTTP: httpClient, CacheDir: cacheDir}
	var resolved []components.Component
	for _, item := range source.Components {
		component, err := resolve(ctx, client, item)
		if err != nil {
			return err
		}
		resolved = append(resolved, component)
	}
	serial, err := nextSerial(o.output)
	if err != nil {
		return err
	}
	manifest := components.Manifest{Serial: serial, Components: resolved, Games: source.Games}
	if err := manifest.Validate(); err != nil {
		return err
	}
	// Unescaped so version ranges such as ">=1.6.14" stay as the committed manifest has them.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		return err
	}
	body := buf.Bytes()
	if err := os.WriteFile(o.output, body, 0o600); err != nil {
		return fmt.Errorf("write component manifest: %w", err)
	}
	if keyPath := os.Getenv("MORTAR_UPDATE_KEY"); keyPath != "" {
		private, err := readPrivateKey(keyPath)
		if err != nil {
			return err
		}
		signature := ed25519.Sign(private, body)
		if err := writeIn(o.signature, signature); err != nil {
			return fmt.Errorf("write component signature: %w", err)
		}
		public, err := os.ReadFile(o.public)
		if err != nil {
			return fmt.Errorf("read component public key: %w", err)
		}
		if err := components.Verify(body, signature, public); err != nil {
			return fmt.Errorf("verify component manifest: %w", err)
		}
	}
	if err := os.WriteFile(o.bundled, body, 0o600); err != nil {
		return fmt.Errorf("write bundled component manifest: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type bearerTransport struct {
	token string
	next  http.RoundTripper
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if clone.URL.Host == "api.github.com" || clone.URL.Host == "github.com" {
		clone.Header.Set("Authorization", "Bearer "+t.token)
	} else {
		clone.Header.Del("Authorization")
	}
	if clone.URL.Host == "api.github.com" && strings.Contains(clone.URL.Path, "/releases/assets/") {
		clone.Header.Set("Accept", "application/octet-stream")
	}
	return t.next.RoundTrip(clone)
}

func readSource(file string) (components.SourceFile, error) {
	body, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return components.SourceFile{}, fmt.Errorf("read component source: %w", err)
	}
	var source components.SourceFile
	if err := json.Unmarshal(body, &source); err != nil {
		return components.SourceFile{}, fmt.Errorf("decode component source: %w", err)
	}
	if len(source.Components) == 0 {
		return components.SourceFile{}, errors.New("component source has no components")
	}
	return source, nil
}

func resolve(ctx context.Context, client *github.Client, source components.SourceComponent) (components.Component, error) {
	if source.Game == "" || source.Name == "" || source.AssetPattern == "" || source.Version == "" {
		return components.Component{}, fmt.Errorf("component %q is missing a required field", source.Name)
	}
	if source.Kind != "loader" && source.Kind != "bridge" {
		return components.Component{}, fmt.Errorf("component %q has unknown kind %q", source.Name, source.Kind)
	}
	if err := source.Source.Validate(); err != nil {
		return components.Component{}, err
	}
	if strings.ToLower(source.Source.Host) != "github.com" {
		return components.Component{}, fmt.Errorf("component %q uses an unsupported generator host %q", source.Name, source.Source.Host)
	}
	releases, err := client.Releases(ctx, source.Source.Owner, source.Source.Repo)
	if err != nil {
		return components.Component{}, fmt.Errorf("look up %s/%s: %w", source.Source.Owner, source.Source.Repo, err)
	}
	release, asset, err := selectAsset(releases, source.Source.Owner+"/"+source.Source.Repo, source.Version, source.AssetPattern)
	if err != nil {
		return components.Component{}, fmt.Errorf("resolve %s: %w", source.Name, err)
	}
	temp, err := os.CreateTemp("", "mortar-component-source-*")
	if err != nil {
		return components.Component{}, err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return components.Component{}, err
	}
	defer func() { _ = os.Remove(tempPath) }()
	if err := downloadAsset(ctx, client.HTTP, downloadURL(client, source, asset), tempPath); err != nil {
		return components.Component{}, fmt.Errorf("download %s: %w", asset.Name, err)
	}
	sum, err := fileSHA256(tempPath)
	if err != nil {
		return components.Component{}, err
	}
	version := strings.TrimPrefix(release.Tag, "v")
	return components.Component{
		Game: source.Game, Name: source.Name, Kind: source.Kind, Source: source.Source,
		Tag: release.Tag, Asset: asset.Name, Version: version, Accepted: source.Accepted, SHA256: sum,
	}, nil
}

func downloadURL(client *github.Client, source components.SourceComponent, asset github.Asset) string {
	if asset.ID != 0 {
		base := client.APIBase
		if base == "" {
			base = "https://api.github.com"
		}
		return fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", base, source.Source.Owner, source.Source.Repo, asset.ID)
	}
	return asset.URL
}

func downloadAsset(ctx context.Context, client *http.Client, address, dest string) error {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server answered %s", resp.Status)
	}
	if resp.ContentLength > maxAsset {
		return fmt.Errorf("larger than %d MiB", maxAsset>>20)
	}
	file, err := os.OpenFile(filepath.Clean(dest), os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(file, io.LimitReader(resp.Body, maxAsset+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n > maxAsset {
		return fmt.Errorf("larger than %d MiB", maxAsset>>20)
	}
	return nil
}

func selectAsset(releases []github.Release, repo, version, pattern string) (github.Release, github.Asset, error) {
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		releaseVersion := strings.TrimPrefix(release.Tag, "v")
		if version != "latest" && normalizeVersion(version) != normalizeVersion(releaseVersion) &&
			normalizeVersion(version) != normalizeVersion(release.Tag) {
			continue
		}
		resolvedPattern := strings.ReplaceAll(pattern, "{version}", releaseVersion)
		resolvedPattern = strings.ReplaceAll(resolvedPattern, "{tag}", release.Tag)
		for _, asset := range release.Assets {
			matched, err := path.Match(resolvedPattern, asset.Name)
			if err != nil {
				return github.Release{}, github.Asset{}, fmt.Errorf("invalid asset pattern %q: %w", pattern, err)
			}
			if matched {
				if asset.URL == "" {
					asset.URL = "https://github.com/" + repo + "/releases/download/" + release.Tag + "/" + asset.Name
				}
				return release, asset, nil
			}
		}
	}
	return github.Release{}, github.Asset{}, fmt.Errorf("no release asset matches %q", pattern)
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func nextSerial(output string) (uint64, error) {
	var previous components.Manifest
	if body, err := os.ReadFile(filepath.Clean(output)); err == nil {
		if json.Unmarshal(body, &previous) != nil {
			previous.Serial = 0
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, fmt.Errorf("read existing component manifest: %w", err)
	}
	now := uint64(0)
	if seconds := time.Now().Unix(); seconds > 0 {
		now = uint64(seconds)
	}
	if previous.Serial >= now {
		return previous.Serial + 1, nil
	}
	return now, nil
}

func readPrivateKey(path string) (ed25519.PrivateKey, error) {
	body, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read signing key: %w", err)
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, errors.New("signing key is not PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse signing key: %w", err)
	}
	private, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("signing key is not Ed25519")
	}
	return private, nil
}

func fileSHA256(file string) (string, error) {
	f, err := os.Open(filepath.Clean(file))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeIn writes data to path through an os.Root on its directory, so the write cannot leave that directory.
func writeIn(path string, data []byte) error {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return root.WriteFile(filepath.Base(path), data, 0o600)
}
