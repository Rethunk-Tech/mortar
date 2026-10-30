package devsvc

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	maxManifest = 1 << 20
	// maxDescription is the longest description Thunderstore takes.
	maxDescription = 250
	iconSide       = 256
)

var (
	packageName    = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	packageVersion = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	dependencyRef  = regexp.MustCompile(`^[A-Za-z0-9_]+-[A-Za-z0-9_]+-\d+\.\d+\.\d+$`)
)

// PackResult is a release build's package and what Thunderstore would refuse in it.
type PackResult struct {
	BuildResult
	Zip    string   `json:"zip"`
	Issues []string `json:"issues"`
}

// Pack builds the project in Release, which packs it for Thunderstore, and checks the newest package it wrote.
func (s *Service) Pack(ctx context.Context, gameID, dir string) (PackResult, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return PackResult{}, err
	}
	if err := requireTrust(p); err != nil {
		return PackResult{}, err
	}
	if p.Kind != "" {
		return s.packAssetProject(ctx, gameID, p)
	}
	defer s.lockedBuild(ctx, gameID, p)()
	started := time.Now()
	out, err := s.run(ctx, p.Dir, "dotnet", buildArgs(p, "Release")...)
	r := PackResult{OK: err == nil, Problems: ParseProblems(string(out)), Log: tail(string(out)), Issues: []string{}}
	if !r.OK {
		return r, nil
	}
	zips, _ := filepath.Glob(filepath.Join(p.Dir, "artifacts", "thunderstore", "release", "*.zip"))
	for _, z := range zips {
		if info, err := os.Stat(z); err == nil && !info.ModTime().Before(started.Add(-time.Second)) {
			r.Zip = z
		}
	}
	if r.Zip == "" {
		r.Issues = append(r.Issues, "the build wrote no package")
		return r, nil
	}
	r.Issues = Validate(r.Zip)
	if p.TemplateIcon != "" && zipFileHash(r.Zip, "icon.png") == p.TemplateIcon {
		r.Issues = append(r.Issues, "icon.png is still the template's; draw the mod its own")
	}
	return r, nil
}

func tail(out string) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	return strings.Join(lines[max(0, len(lines)-buildLogLines):], "\n")
}

type thunderstoreManifest struct {
	Name          string   `json:"name"`
	VersionNumber string   `json:"version_number"`
	Description   string   `json:"description"`
	WebsiteURL    *string  `json:"website_url"`
	Dependencies  []string `json:"dependencies"`
}

// Validate lists what Thunderstore's upload would refuse in the package at path: its manifest's name, version,
// description and dependencies, a 256×256 PNG icon, and a README.
func Validate(path string) []string {
	issues := []string{}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return append(issues, "the package is not a zip: "+err.Error())
	}
	defer func() { _ = zr.Close() }()
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	if files["README.md"] == nil {
		issues = append(issues, "README.md is missing")
	}
	if f := files["icon.png"]; f == nil {
		issues = append(issues, "icon.png is missing")
	} else if rc, err := f.Open(); err == nil {
		cfg, err := png.DecodeConfig(rc)
		_ = rc.Close()
		if err != nil || cfg.Width != iconSide || cfg.Height != iconSide {
			issues = append(issues, fmt.Sprintf("icon.png must be a %d×%d PNG", iconSide, iconSide))
		}
	}
	f := files["manifest.json"]
	if f == nil {
		return append(issues, "manifest.json is missing")
	}
	rc, err := f.Open()
	if err != nil {
		return append(issues, err.Error())
	}
	body, err := safefs.ReadCapped(rc, maxManifest)
	_ = rc.Close()
	var m thunderstoreManifest
	if err != nil || json.Unmarshal(body, &m) != nil {
		return append(issues, "manifest.json does not parse")
	}
	if !packageName.MatchString(m.Name) || len(m.Name) > 128 {
		issues = append(issues, "the name holds only letters, digits and underscores")
	}
	if !packageVersion.MatchString(m.VersionNumber) {
		issues = append(issues, "the version must be three numbers, such as 1.0.0")
	}
	if n := len([]rune(m.Description)); n > maxDescription {
		issues = append(issues, fmt.Sprintf("the description is %d characters; Thunderstore takes %d", n, maxDescription))
	}
	if m.WebsiteURL == nil {
		issues = append(issues, "website_url is missing (it may be empty)")
	}
	for _, d := range m.Dependencies {
		if !dependencyRef.MatchString(d) {
			issues = append(issues, fmt.Sprintf("dependency %q is not Team-Name-1.2.3", d))
		}
	}
	return issues
}

// zipFileHash is the SHA-256 of one file in the zip at path, empty when it cannot be read.
func zipFileHash(path, name string) string {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return ""
	}
	defer func() { _ = zr.Close() }()
	rc, err := zr.Open(name)
	if err != nil {
		return ""
	}
	defer func() { _ = rc.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, safefs.CapReader(rc, maxManifest)); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Versions are the project's released version (its latest vX.Y.Z tag, empty before the first) and the next patch,
// minor and major after it; before the first release each is 0.1.0, the version the template starts at.
type Versions struct {
	// GitHub is the remote a release is pushed to so the template's CI publishes it; empty means the release stays
	// local and Mortar uploads the package itself.
	GitHub  string `json:"github"`
	Current string `json:"current"`
	Patch   string `json:"patch"`
	Minor   string `json:"minor"`
	Major   string `json:"major"`
	// Changes are the subjects of the commits since that tag (all of them before the first release), newest first,
	// without merges or Mortar's own release commits: a draft of what changed.
	Changes []string `json:"changes"`
}

// NextVersions reads the project's latest release tag.
func (s *Service) NextVersions(ctx context.Context, gameID, dir string) (Versions, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return Versions{}, err
	}
	if p.Untrusted {
		return nextVersions(""), nil
	}
	latest := s.latestRelease(ctx, p.Dir)
	v := nextVersions(latest)
	if p.Kind == "" {
		// An asset project has no CI workflow; its release is uploaded from here.
		v.GitHub = s.githubRemote(ctx, p.Dir)
	}
	since := "HEAD"
	if latest != "" {
		since = "v" + latest + "..HEAD"
	}
	v.Changes = []string{}
	if out, err := s.run(ctx, p.Dir, "git", "log", "--no-merges", "--format=%s", since); err == nil {
		v.Changes = changeLines(string(out))
	}
	return v, nil
}

// latestRelease is the newest released version's tag, without the v. Before the first release git finds no tag, and
// the template's version stands.
func (s *Service) latestRelease(ctx context.Context, dir string) string {
	out, err := s.run(ctx, dir, "git", "describe", "--tags", "--abbrev=0", "--match", "v*")
	if err != nil {
		return ""
	}
	return nextVersions(strings.TrimPrefix(strings.TrimSpace(string(out)), "v")).Current
}

// changeLines keeps git log's subjects, dropping blanks and the "Release X.Y.Z" commits Release makes.
func changeLines(out string) []string {
	lines := []string{}
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		version, release := strings.CutPrefix(line, "Release ")
		if line == "" || release && packageVersion.MatchString(version) {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func nextVersions(current string) Versions {
	m := packageVersion.FindStringSubmatch(current)
	if m == nil {
		return Versions{Patch: "0.1.0", Minor: "0.1.0", Major: "0.1.0"}
	}
	n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
	return Versions{
		Current: current,
		Patch:   fmt.Sprintf("%d.%d.%d", n(1), n(2), n(3)+1),
		Minor:   fmt.Sprintf("%d.%d.0", n(1), n(2)+1),
		Major:   fmt.Sprintf("%d.0.0", n(1)+1),
	}
}

var remoteName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// githubRemote is the remote the current branch tracks, else origin, when its URL is on GitHub; empty otherwise.
func (s *Service) githubRemote(ctx context.Context, dir string) string {
	remote := "origin"
	if out, err := s.run(ctx, dir, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
		if name, _, ok := strings.Cut(strings.TrimSpace(string(out)), "/"); ok {
			remote = name
		}
	}
	if !remoteName.MatchString(remote) {
		return ""
	}
	out, err := s.run(ctx, dir, "git", "remote", "get-url", remote)
	if err != nil || !strings.Contains(strings.ToLower(string(out)), "github.com") {
		return ""
	}
	return remote
}

// Release adds version and its notes to CHANGELOG.md, commits that, and tags it vX.Y.Z, which the template takes as
// the version. The tree must be otherwise committed, so the tag holds what was tested. With a GitHub remote the
// commit and tag are pushed there, and the template's CI publishes the package; it returns that remote, or "" when
// the release stayed local.
func (s *Service) Release(ctx context.Context, gameID, dir, version string, notes []string) (string, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return "", err
	}
	if err := requireTrust(p); err != nil {
		return "", err
	}
	if err := s.tagRelease(ctx, p.Dir, version, notes); err != nil {
		return "", err
	}
	remote := s.githubRemote(ctx, p.Dir)
	if remote == "" {
		return "", nil
	}
	if _, err := s.run(ctx, p.Dir, "git", "push", remote, "HEAD", "refs/tags/v"+version); err != nil {
		return "", fmt.Errorf("v%s is tagged here, but pushing it to %s failed: %w", version, remote, err)
	}
	if p.Kind != "" {
		return "", nil
	}
	return remote, nil
}

func (s *Service) tagRelease(ctx context.Context, dir, version string, notes []string) error {
	if !packageVersion.MatchString(version) {
		return usererr.Wrap(usererr.Invalid, errors.New("the version must be three numbers, such as 1.0.0"))
	}
	notes = slices.DeleteFunc(slices.Clone(notes), func(n string) bool { return strings.TrimSpace(n) == "" })
	if len(notes) == 0 {
		return usererr.Wrap(usererr.Invalid, errors.New("say what changed"))
	}
	if out, err := s.run(ctx, dir, "git", "status", "--porcelain"); err != nil {
		return err
	} else if strings.TrimSpace(string(out)) != "" {
		return usererr.Wrap(usererr.Invalid, errors.New("commit your changes first, so the release holds what you tested"))
	}
	if tagged, err := s.run(ctx, dir, "git", "rev-parse", "-q", "--verify", "refs/tags/v"+version+"^{commit}"); err == nil {
		// A release whose push failed is tagged on HEAD; trying again goes on to the push.
		if head, err := s.run(ctx, dir, "git", "rev-parse", "HEAD"); err == nil && strings.TrimSpace(string(head)) == strings.TrimSpace(string(tagged)) {
			return nil
		}
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("v%s is already released", version))
	}
	log := filepath.Join(dir, "CHANGELOG.md")
	body, err := safefs.ReadFile(log)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	next := AddChangelogEntry(string(body), version, notes)
	if next != string(body) {
		if err := safefs.WriteFile(log, []byte(next), 0o644); err != nil { // #nosec G306 -- a tracked text file
			return err
		}
		_, err = s.run(ctx, dir, "git", "add", "CHANGELOG.md")
		if err == nil {
			_, err = s.run(ctx, dir, "git", "commit", "-q", "-m", "Release "+version)
		}
		if err != nil {
			_, _ = s.run(ctx, dir, "git", "reset", "-q", "--", "CHANGELOG.md")
			if len(body) == 0 {
				_ = safefs.Remove(log)
			} else {
				_ = safefs.WriteFile(log, body, 0o644) // #nosec G306 -- a tracked text file
			}
			return err
		}
	}
	_, err = s.run(ctx, dir, "git", "tag", "-a", "v"+version, "-m", version)
	return err
}

// AddChangelogEntry puts a "## version" section with notes as bullets above the newest one, under the title; a
// changelog that already has the version is left alone.
func AddChangelogEntry(changelog, version string, notes []string) string {
	heading := "## " + version
	for line := range strings.Lines(changelog) {
		if strings.TrimSpace(line) == heading {
			return changelog
		}
	}
	var entry strings.Builder
	entry.WriteString(heading + "\n\n")
	for _, n := range notes {
		entry.WriteString("- " + strings.TrimSpace(n) + "\n")
	}
	entry.WriteString("\n")
	if changelog == "" {
		return "# Changelog\n\n" + entry.String()
	}
	if i := strings.Index(changelog, "\n## "); i >= 0 {
		return changelog[:i+1] + entry.String() + changelog[i+1:]
	}
	return strings.TrimRight(changelog, "\n") + "\n\n" + entry.String()
}
