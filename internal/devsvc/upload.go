package devsvc

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-safefs"
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
	p, zipPath, err := s.uploadPackage(gameID, dir)
	if err != nil {
		return "", err
	}
	community := tsupload.Community(gameID)
	if community == "" {
		return "", usererr.Wrap(usererr.Invalid, errors.New("this game has no Thunderstore community"))
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

// UploadSites lists the sites the game can be uploaded to: Thunderstore when it has a community, Nexus Mods when it
// has a Nexus domain and CurseForge when its source names an upload subdomain, and GitHub when the project's origin is there. Edit shows the id fields of those; Upload
// offers the Configured ones.
func (s *Service) UploadSites(ctx context.Context, gameID, dir string) ([]UploadSite, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return nil, err
	}
	sites := []UploadSite{}
	if tsupload.Community(gameID) != "" {
		sites = append(sites, UploadSite{ID: "thunderstore", Configured: true, Ready: s.HasThunderstoreToken(gameID, dir)})
	}
	ids := readSites(p.Dir)
	if nexusDomain(gameID) != "" {
		_, err := nexusAPIKey()
		sites = append(sites, UploadSite{ID: "nexus", Configured: ids.Nexus.FileID != "", Ready: err == nil})
	}
	if curseForgeBase(gameID) != "" {
		sites = append(sites, UploadSite{ID: "curseforge", Configured: ids.CurseForge.ProjectID > 0, Ready: s.HasCurseForgeToken()})
	}
	if _, _, ok := s.githubOrigin(ctx, p.Dir); ok {
		sites = append(sites, UploadSite{ID: "github", Configured: true, Ready: gh.DefaultAuth.LoggedIn(ctx)})
	}
	return sites, nil
}

// releaseNotes is the project's CHANGELOG.md notes for version, empty when it has none.
func releaseNotes(dir, version string) (string, error) {
	if !packageVersion.MatchString(version) {
		return "", usererr.Wrap(usererr.Invalid, errors.New("the version must be three numbers, such as 1.0.0"))
	}
	body, err := safefs.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return changelogOf(string(body), version), nil
}

// UploadNexus uploads the project's newest package as the next version of its Nexus Mods file, archiving the one it
// replaces, and returns the mod's files page (empty without a mod id).
func (s *Service) UploadNexus(ctx context.Context, gameID, dir, version string) (string, error) {
	p, zipPath, err := s.uploadPackage(gameID, dir)
	if err != nil {
		return "", err
	}
	ids := readSites(p.Dir).Nexus
	if ids.FileID == "" || nexusDomain(gameID) == "" {
		return "", usererr.Wrap(usererr.Invalid, errors.New("set the project's Nexus Mods file id first"))
	}
	notes, err := releaseNotes(p.Dir, version)
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
	up := nexusUploader{client: s.httpClient(), base: nexusAPI, key: key, version: s.Version, poll: 2 * time.Second}
	if err := up.upload(ctx, zipPath, info.Name, ids, version, notes); err != nil {
		return "", err
	}
	return nexusPage(gameID, ids), nil
}

// UploadCurseForge uploads the project's newest package as a release of its CurseForge project and returns the
// project's files page on the authors site.
func (s *Service) UploadCurseForge(ctx context.Context, gameID, dir, version string) (string, error) {
	p, zipPath, err := s.uploadPackage(gameID, dir)
	if err != nil {
		return "", err
	}
	id := readSites(p.Dir).CurseForge.ProjectID
	base := curseForgeBase(gameID)
	if id <= 0 || base == "" {
		return "", usererr.Wrap(usererr.Invalid, errors.New("set the project's CurseForge project id first"))
	}
	notes, err := releaseNotes(p.Dir, version)
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
	up := curseForgeUploader{client: s.httpClient(), base: base, token: token, version: s.Version}
	if _, err := up.upload(ctx, zipPath, id, info.Name+" "+version, notes); err != nil {
		return "", err
	}
	return curseForgePage(id), nil
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
