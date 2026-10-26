package devsvc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

var githubRepo = regexp.MustCompile(`^(?:https://github\.com/|git@github\.com:)?([A-Za-z0-9][A-Za-z0-9-]{0,38})/([A-Za-z0-9._-]{1,100}?)(?:\.git)?/?$`)

// enroll makes dir's build output an extra mods folder of the project's profile (created as "<game> Development"
// when profileID is empty, filled from copyFrom) and records the project.
func (s *Service) enroll(gameID, dir, name, profileID, copyFrom string) (settings.DevProject, error) {
	if profileID == "" {
		gameName := gameID
		if info, ok := components.Game(gameID); ok {
			gameName = info.Name
		}
		p, err := s.Profiles.Create(gameID, gameName+" Development")
		if err != nil {
			return settings.DevProject{}, err
		}
		profileID = p.ID
		// Test runs host games, and a host writes its save; the player's own saves stay out of it.
		if _, err := s.Profiles.SetSeparateSaves(gameID, profileID, true, false); err != nil {
			return settings.DevProject{}, err
		}
		if copyFrom != "" {
			if err := s.copyMods(gameID, copyFrom, profileID); err != nil {
				return settings.DevProject{}, err
			}
		}
	}
	// The template builds an unpacked package here; it exists before the first build so the folder watch can start.
	output := filepath.Join(dir, "artifacts", "bin", name, "debug")
	if err := os.MkdirAll(output, 0o750); err != nil {
		return settings.DevProject{}, err
	}
	p := settings.DevProject{Dir: dir, Name: name, Profile: profileID, Output: output}
	if icon, err := fsx.ReadFile(filepath.Join(dir, "icon.png")); err == nil {
		p.TemplateIcon = fmt.Sprintf("%x", sha256.Sum256(icon))
	}
	return p, s.Settings.AddDevProject(gameID, p)
}

// projectName is the mod's name in a project made from the template: the one its record names, else the only folder
// under src/ that holds a project file of its own name.
func projectName(dir string) (string, bool) {
	has := func(name string) bool {
		if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
			return false
		}
		_, err := os.Stat(filepath.Join(dir, "src", name, name+".csproj"))
		return err == nil
	}
	if body, err := fsx.ReadFile(filepath.Join(dir, ".mortar", "template.json")); err == nil {
		var rec projectRecord
		if json.Unmarshal(body, &rec) == nil && has(rec.Name) {
			return rec.Name, true
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "src"))
	var found []string
	for _, e := range entries {
		if e.IsDir() && has(e.Name()) {
			found = append(found, e.Name())
		}
	}
	if len(found) == 1 {
		return found[0], true
	}
	return "", false
}

// Add puts a mod project that already exists in dir (a clone of a repository made from the game's template) in the
// Develop tab. Its Config.Build.user.props is written only when it has none. profileID is the profile its builds
// install into; empty creates "<game> Development".
func (s *Service) Add(ctx context.Context, gameID, dir, profileID string) (settings.DevProject, error) {
	bad := func(what string, args ...any) (settings.DevProject, error) {
		return settings.DevProject{}, usererr.Wrap(usererr.Invalid, fmt.Errorf(what, args...))
	}
	if _, ok := Template(gameID); !ok {
		return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no mod template", gameID))
	}
	if !filepath.IsAbs(dir) {
		return bad("pick the project's folder")
	}
	dir = filepath.Clean(dir)
	name, ok := projectName(dir)
	if !ok {
		return bad("%s is not a mod project from the template: it has no src/<Name>/<Name>.csproj", dir)
	}
	if slices.ContainsFunc(s.Projects(gameID), func(p settings.DevProject) bool { return filepath.Clean(p.Dir) == dir }) {
		return bad("%s is already in the Develop tab", dir)
	}
	if profileID != "" {
		profiles, err := s.Profiles.List(gameID)
		if err != nil {
			return settings.DevProject{}, err
		}
		if !slices.ContainsFunc(profiles, func(p profile.Profile) bool { return p.ID == profileID }) {
			return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no profile %q", profileID))
		}
	}
	props := filepath.Join(dir, "Config.Build.user.props")
	if _, err := os.Lstat(props); errors.Is(err, os.ErrNotExist) {
		if err := writeUserProps(dir, s.Tools(ctx, gameID).GameDir); err != nil {
			return settings.DevProject{}, err
		}
	}
	return s.enroll(gameID, dir, name, profileID, "")
}

// ParseRepo reads owner/name, or a github.com address (https or git@), as owner and name.
func ParseRepo(repo string) (owner, name string, err error) {
	m := githubRepo.FindStringSubmatch(strings.TrimSpace(repo))
	if m == nil {
		return "", "", usererr.Wrap(usererr.Invalid, errors.New("name the repository as owner/name or its github.com address"))
	}
	return m[1], m[2], nil
}

// Clone copies the GitHub repository (owner/name or its address) into parentDir/<name> and adds it as Add does; a
// repository that is not a mod project from the template is removed again.
func (s *Service) Clone(ctx context.Context, gameID, repo, parentDir, profileID string) (settings.DevProject, error) {
	owner, name, err := ParseRepo(repo)
	if err != nil {
		return settings.DevProject{}, err
	}
	if !filepath.IsAbs(parentDir) {
		return settings.DevProject{}, usererr.Wrap(usererr.Invalid, errors.New("pick the folder the project goes in"))
	}
	dest := filepath.Join(parentDir, name)
	if _, err := os.Lstat(dest); err == nil {
		return settings.DevProject{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%s already exists", dest))
	}
	full := owner + "/" + name
	if s.Tools(ctx, gameID).GitHubUser != "" {
		_, err = s.run(ctx, parentDir, "gh", "repo", "clone", full, name)
	} else {
		_, err = s.run(ctx, parentDir, "git", "clone", "https://github.com/"+full+".git", name)
	}
	if err != nil {
		_ = fsx.RemoveAll(dest)
		return settings.DevProject{}, err
	}
	p, err := s.Add(ctx, gameID, dest, profileID)
	if err != nil {
		_ = fsx.RemoveAll(dest)
	}
	return p, err
}
