package devsvc

import (
	"archive/zip"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-curseforge"
	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/tsupload"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// SetThunderstoreToken keeps the service account token of the project's Thunderstore team in the keyring for uploads.
func (s *Service) SetThunderstoreToken(gameID, dir, token string) error {
	if token == "" {
		return usererr.Wrap(usererr.Invalid, errors.New("the token is empty"))
	}
	team, err := s.projectTeam(gameID, dir)
	if err != nil {
		return err
	}
	return tsupload.SetToken(team, token)
}

// HasThunderstoreToken reports whether the project's team has an upload token kept.
func (s *Service) HasThunderstoreToken(gameID, dir string) bool {
	team, err := s.projectTeam(gameID, dir)
	if err != nil {
		return false
	}
	return tsupload.HasToken(team)
}

// ForgetThunderstoreToken removes the project's team's kept upload token.
func (s *Service) ForgetThunderstoreToken(gameID, dir string) error {
	team, err := s.projectTeam(gameID, dir)
	if err != nil {
		return err
	}
	return tsupload.ForgetToken(team)
}

func (s *Service) projectTeam(gameID, dir string) (string, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return "", err
	}
	info, err := projectInfo(p)
	if err != nil {
		return "", err
	}
	if info.Team == "" {
		return "", usererr.Wrap(usererr.Invalid, errors.New("the project names no Thunderstore team"))
	}
	return info.Team, nil
}

// projectInfo reads the team and categories an upload needs: from the record of an asset project, else the project file.
func projectInfo(p settings.DevProject) (ProjectInfo, error) {
	if rec, ok := readAssetRecord(p.Dir); ok {
		return assetInfo(p.Dir, rec), nil
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return ProjectInfo{}, usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	return readInfo(p.Dir, csproj)
}

// uploadPackage is the trusted project in dir and the newest package Pack wrote for it.
func (s *Service) uploadPackage(gameID, dir string) (settings.DevProject, string, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return p, "", err
	}
	if err := requireTrust(p); err != nil {
		return p, "", err
	}
	zipPath := newestPackage(p.Dir)
	if zipPath == "" {
		return p, "", usererr.Wrap(usererr.NotFound, errors.New("pack the mod first"))
	}
	return p, zipPath, nil
}

// UploadThunderstore uploads the project's newest package to the game's Thunderstore community under the project's
// team and categories, as Thunderstore's own CLI does, and returns the package's page.
func (s *Service) UploadThunderstore(ctx context.Context, gameID, dir string) (string, error) {
	community := tsupload.Community(gameID)
	p, zipPath, _, err := s.prepareUpload(gameID, dir, "", false, func(settings.DevProject) error {
		if community == "" {
			return usererr.Wrap(usererr.Invalid, errors.New("this game has no Thunderstore community"))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	info, err := projectInfo(p)
	if err != nil {
		return "", err
	}
	return tsupload.Upload(ctx, s.httpClient(), s.thunderstoreSite(), zipPath, info.Team, community, info.Categories)
}

// UploadSite is a site the game can be uploaded to: its id is thunderstore, nexus, curseforge or github. Configured is set when
// the project's ids for it are filled in (Thunderstore needs none), and Ready when the credentials it needs are kept.
type UploadSite struct {
	ID         string `json:"id"`
	Configured bool   `json:"configured"`
	Ready      bool   `json:"ready"`
}

// uploadSiteTable is each site's probe: whether the game offers it, whether the project's ids for it are filled in,
// and whether the credentials it needs are kept.
var uploadSiteTable = []struct {
	id    string
	probe func(ctx context.Context, s *Service, p settings.DevProject, gameID, dir string) (offered, configured, ready bool)
}{
	{"thunderstore", func(_ context.Context, s *Service, _ settings.DevProject, gameID, dir string) (bool, bool, bool) {
		return tsupload.Community(gameID) != "", true, s.HasThunderstoreToken(gameID, dir)
	}},
	{"nexus", func(_ context.Context, _ *Service, p settings.DevProject, gameID, _ string) (bool, bool, bool) {
		_, err := nexusAPIKey()
		g, _ := components.Game(gameID)
		return g.NexusDomain() != "", readSites(p.Dir).Nexus.FileID != "", err == nil
	}},
	{"curseforge", func(_ context.Context, s *Service, p settings.DevProject, gameID, _ string) (bool, bool, bool) {
		return curseForgeBase(gameID) != "", readSites(p.Dir).CurseForge.ProjectID > 0, s.HasCurseForgeToken()
	}},
	{"github", func(ctx context.Context, s *Service, p settings.DevProject, _, _ string) (bool, bool, bool) {
		_, _, ok := s.githubOrigin(ctx, p.Dir)
		return ok, true, ok && gh.DefaultAuth.LoggedIn(ctx)
	}},
}

// UploadSites lists the sites the game can be uploaded to: Thunderstore when it has a community, Nexus Mods when it
// has a Nexus domain and CurseForge when its source names an upload subdomain, and GitHub when the project's origin is there. Edit shows the id fields of those; Upload
// offers the Configured ones.
func (s *Service) UploadSites(ctx context.Context, gameID, dir string) ([]UploadSite, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return nil, err
	}
	sites := []UploadSite{}
	for _, t := range uploadSiteTable {
		if offered, configured, ready := t.probe(ctx, s, p, gameID, dir); offered {
			sites = append(sites, UploadSite{ID: t.id, Configured: configured, Ready: ready})
		}
	}
	return sites, nil
}

// prepareUpload is the opening every upload shares: the trusted project and its newest package, the site's own check of
// the project's ids, and the CHANGELOG notes for version when the site takes any.
func (s *Service) prepareUpload(gameID, dir, version string, notes bool, check func(p settings.DevProject) error) (p settings.DevProject, zipPath, body string, err error) {
	if p, zipPath, err = s.uploadPackage(gameID, dir); err != nil {
		return
	}
	if version != "" {
		if err = requirePackageVersion(zipPath, version); err != nil {
			return
		}
	}
	if check != nil {
		if err = check(p); err != nil {
			return
		}
	}
	if notes {
		body, err = releaseNotes(p.Dir, version)
	}
	return
}

// requirePackageVersion refuses an upload whose package was packed at another version than the one given.
func requirePackageVersion(zipPath, version string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return usererr.Wrap(usererr.Damaged, fmt.Errorf("the package cannot be read: %w", err))
	}
	defer func() { _ = zr.Close() }()
	var m struct {
		Version string `json:"version_number"`
	}
	for _, f := range zr.File {
		if f.Name != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return usererr.Wrap(usererr.Damaged, err)
		}
		err = json.NewDecoder(io.LimitReader(rc, 1<<20)).Decode(&m)
		_ = rc.Close()
		if err != nil {
			return usererr.Wrap(usererr.Damaged, fmt.Errorf("the package's manifest.json: %w", err))
		}
	}
	if m.Version != version {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("the newest package is version %q, not %s: pack the mod at %s first", m.Version, version, version))
	}
	return nil
}

// releaseNotes is the project's changelog notes for version, empty when it has none.
func releaseNotes(dir, version string) (string, error) {
	if !packageVersion.MatchString(version) {
		return "", usererr.Wrap(usererr.Invalid, errors.New("the version must be three numbers, such as 1.0.0"))
	}
	body, _, err := safefs.ReadFileIfExists(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		return "", err
	}
	return changelogOf(string(body), version), nil
}

// UploadNexus uploads the project's newest package as the next version of its Nexus Mods file, archiving the one it
// replaces, and returns the mod's files page (empty without a mod id).
func (s *Service) UploadNexus(ctx context.Context, gameID, dir, version string) (string, error) {
	var ids NexusIDs
	p, zipPath, notes, err := s.prepareUpload(gameID, dir, version, true, func(p settings.DevProject) error {
		g, _ := components.Game(gameID)
		if ids = readSites(p.Dir).Nexus; ids.FileID == "" || g.NexusDomain() == "" {
			return usererr.Wrap(usererr.Invalid, errors.New("set the project's Nexus Mods file id first"))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	key, err := nexusAPIKey()
	if err != nil {
		return "", err
	}
	info, err := projectInfo(p)
	if err != nil {
		return "", err
	}
	if err := s.uploadNexus(ctx, key, zipPath, info.Name, ids, version, notes); err != nil {
		return "", err
	}
	return nexusPage(gameID, ids), nil
}

// UploadCurseForge uploads the project's newest package as a release of its CurseForge project and returns the
// project's files page on the authors site.
func (s *Service) UploadCurseForge(ctx context.Context, gameID, dir, version string) (string, error) {
	var id int
	base := cmp.Or(s.curseForgeAPI, curseForgeBase(gameID))
	p, zipPath, notes, err := s.prepareUpload(gameID, dir, version, true, func(p settings.DevProject) error {
		if id = readSites(p.Dir).CurseForge.ProjectID; id <= 0 || base == "" {
			return usererr.Wrap(usererr.Invalid, errors.New("set the project's CurseForge project id first"))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	token, err := secret.Get(curseForgeKeyring)
	if err != nil || token == "" {
		return "", usererr.Wrap(usererr.Permission, errors.New("add a CurseForge upload token first"))
	}
	info, err := projectInfo(p)
	if err != nil {
		return "", err
	}
	_, err = curseforge.NewUploader(curseforge.UploadConfig{
		Token: token, BaseURL: base, HTTP: s.httpClient(), UserAgent: appversion.UserAgent(s.Version),
	}).Upload(ctx, zipPath, id, info.Name+" "+version, notes)
	if err = siteError(curseForgeName, err); err != nil {
		return "", err
	}
	return curseforge.AuthorFilesURL(id), nil
}

// newestPackage is the most recent zip Pack wrote.
func newestPackage(dir string) string {
	zips, _ := filepath.Glob(filepath.Join(dir, "artifacts", "thunderstore", "release", "*.zip"))
	newest, at := "", int64(0)
	for _, z := range zips {
		if st, err := os.Stat(z); err == nil && st.ModTime().UnixNano() > at {
			newest, at = z, st.ModTime().UnixNano()
		}
	}
	return newest
}

func (s *Service) httpClient() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return tsupload.Client
}

func (s *Service) thunderstoreSite() string {
	if s.ThunderstoreURL != "" {
		return s.ThunderstoreURL
	}
	return tsupload.Site
}
