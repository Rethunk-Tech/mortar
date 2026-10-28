package devsvc

import (
	"bytes"
	"context"
	"crypto/md5" // #nosec G501 -- S3's Content-MD5 integrity header, not a security hash
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// thunderstoreKeyring is the keyring service a team's Thunderstore service account token is kept under, named by the
// team: the template's own publish reads the same item.
const thunderstoreKeyring = "thunderstore"

const thunderstoreSite = "https://thunderstore.io/"

// SetThunderstoreToken keeps the service account token of the project's Thunderstore team in the keyring for uploads.
func (s *Service) SetThunderstoreToken(gameID, dir, token string) error {
	if token == "" {
		return usererr.Wrap(usererr.Invalid, errors.New("the token is empty"))
	}
	team, err := s.projectTeam(gameID, dir)
	if err != nil {
		return err
	}
	return secret.SetIn(thunderstoreKeyring, team, token)
}

// HasThunderstoreToken reports whether the project's team has an upload token kept.
func (s *Service) HasThunderstoreToken(gameID, dir string) bool {
	team, err := s.projectTeam(gameID, dir)
	if err != nil {
		return false
	}
	token, err := secret.GetIn(thunderstoreKeyring, team)
	return err == nil && token != ""
}

// ForgetThunderstoreToken removes the project's team's kept upload token.
func (s *Service) ForgetThunderstoreToken(gameID, dir string) error {
	team, err := s.projectTeam(gameID, dir)
	if err != nil {
		return err
	}
	return secret.DeleteIn(thunderstoreKeyring, team)
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
	if p.Kind != "" {
		rec, ok := readAssetRecord(p.Dir)
		if !ok {
			return ProjectInfo{}, usererr.Wrap(usererr.NotFound, errors.New("the project's record is missing"))
		}
		return assetInfo(p.Dir, rec), nil
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return ProjectInfo{}, usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	return readInfo(p.Dir, csproj)
}

type uploadPart struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
	Offset     int64  `json:"offset"`
	Length     int64  `json:"length"`
}

type finishedPart struct {
	ETag       string `json:"ETag"`
	PartNumber int    `json:"PartNumber"`
}

// UploadThunderstore uploads the project's newest package to the game's Thunderstore community under the project's
// team and categories, as Thunderstore's own CLI does, and returns the package's page.
func (s *Service) UploadThunderstore(ctx context.Context, gameID, dir string) (string, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return "", err
	}
	if err := requireTrust(p); err != nil {
		return "", err
	}
	community := thunderstoreCommunity(gameID)
	if community == "" {
		return "", usererr.Wrap(usererr.Invalid, errors.New("this game has no Thunderstore community"))
	}
	zipPath := newestPackage(p.Dir)
	if zipPath == "" {
		return "", usererr.Wrap(usererr.NotFound, errors.New("pack the mod first"))
	}
	info, err := projectInfo(p)
	if err != nil {
		return "", err
	}
	token, err := secret.GetIn(thunderstoreKeyring, info.Team)
	if err != nil || token == "" {
		return "", usererr.Wrap(usererr.Permission, errors.New("add a Thunderstore service account token first"))
	}
	ts := thunderstoreAPI{client: s.httpClient(), site: s.thunderstoreSite(), token: token}
	uuid, err := ts.uploadFile(ctx, zipPath)
	if err != nil {
		return "", err
	}
	return ts.submit(ctx, uuid, info.Team, community, info.Categories)
}

func thunderstoreCommunity(gameID string) string {
	g, _ := components.Game(gameID)
	for _, src := range g.Sources {
		if src.ID == "thunderstore" {
			return src.Key
		}
	}
	return ""
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
	return http.DefaultClient
}

func (s *Service) thunderstoreSite() string {
	if s.ThunderstoreURL != "" {
		return s.ThunderstoreURL
	}
	return thunderstoreSite
}

type thunderstoreAPI struct {
	client *http.Client
	site   string
	token  string
}

func (t thunderstoreAPI) call(ctx context.Context, path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.site+"api/experimental/"+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)
	res, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return usererr.Wrap(usererr.Permission, fmt.Errorf("thunderstore refused the token: %s", reply))
	}
	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("thunderstore %s: %s: %s", path, res.Status, reply)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(reply, out)
}

// uploadFile sends the package in the parts Thunderstore's storage asks for, each to its presigned URL with its MD5,
// and returns the upload's id.
func (t thunderstoreAPI) uploadFile(ctx context.Context, path string) (string, error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return "", err
	}
	var started struct {
		UserMedia struct {
			UUID string `json:"uuid"`
		} `json:"user_media"`
		UploadURLs []uploadPart `json:"upload_urls"`
	}
	if err := t.call(ctx, "usermedia/initiate-upload/", map[string]any{"filename": filepath.Base(path), "file_size_bytes": len(data)}, &started); err != nil {
		return "", err
	}
	uuid := started.UserMedia.UUID
	parts := make([]finishedPart, 0, len(started.UploadURLs))
	for _, part := range started.UploadURLs {
		etag, err := t.putPart(ctx, data, part)
		if err != nil {
			_ = t.call(context.WithoutCancel(ctx), "usermedia/"+uuid+"/abort-upload/", map[string]any{}, nil)
			return "", err
		}
		parts = append(parts, finishedPart{ETag: etag, PartNumber: part.PartNumber})
	}
	return uuid, t.call(ctx, "usermedia/"+uuid+"/finish-upload/", map[string]any{"parts": parts}, nil)
}

func (t thunderstoreAPI) putPart(ctx context.Context, data []byte, part uploadPart) (string, error) {
	if part.Offset < 0 || part.Length < 0 || part.Offset+part.Length > int64(len(data)) {
		return "", fmt.Errorf("thunderstore asked for part %d outside the file", part.PartNumber)
	}
	chunk := data[part.Offset : part.Offset+part.Length]
	sum := md5.Sum(chunk) // #nosec G401 -- S3's Content-MD5 integrity header
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, part.URL, bytes.NewReader(chunk))
	if err != nil {
		return "", err
	}
	req.ContentLength = part.Length
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(sum[:]))
	res, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("uploading part %d: %s", part.PartNumber, res.Status)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		return "", fmt.Errorf("uploading part %d: no ETag", part.PartNumber)
	}
	return etag, nil
}

// submit publishes the uploaded package under team in the community and returns its page.
func (t thunderstoreAPI) submit(ctx context.Context, uuid, team, community string, categories []string) (string, error) {
	var done struct {
		AvailableCommunities []struct {
			URL string `json:"url"`
		} `json:"available_communities"`
	}
	body := map[string]any{
		"upload_uuid":          uuid,
		"author_name":          team,
		"communities":          []string{community},
		"categories":           categories,
		"community_categories": map[string][]string{community: categories},
		"has_nsfw_content":     false,
	}
	if err := t.call(ctx, "submission/submit/", body, &done); err != nil {
		return "", err
	}
	if len(done.AvailableCommunities) == 0 {
		return "", nil
	}
	return done.AvailableCommunities[0].URL, nil
}
