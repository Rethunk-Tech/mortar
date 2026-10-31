package devsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	// curseForgeKeyring is the item an author's CurseForge upload token is kept under, one per user.
	curseForgeKeyring = "curseforge-upload"
	curseForgeName    = "CurseForge"
)

// curseForgeUploader uploads a file to a CurseForge project through the author upload API.
type curseForgeUploader struct {
	client  *http.Client
	base    string
	token   string
	version string
}

// curseForgeBase is the host a game's upload API answers on, empty when the catalog names none.
func curseForgeBase(gameID string) string {
	g, _ := components.Game(gameID)
	if src, ok := g.Source("curseforge"); ok && src.UploadSubdomain != "" {
		return "https://" + src.UploadSubdomain + ".curseforge.com"
	}
	return ""
}

func curseForgePage(projectID int) string {
	return "https://authors.curseforge.com/#/projects/" + strconv.Itoa(projectID) + "/files"
}

// upload sends the package to the project as a release, with the changelog as markdown, and returns the new file's id.
func (c curseForgeUploader) upload(ctx context.Context, path string, projectID int, name, changelog string) (int, error) {
	f, err := safefs.OpenRegular(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	meta, err := json.Marshal(map[string]any{
		"changelog": changelog, "changelogType": "markdown", "releaseType": "release", "displayName": name,
	})
	if err != nil {
		return 0, err
	}
	// The file streams from disk through a pipe rather than being held in memory beside its encoding.
	pr, pw := io.Pipe()
	form := multipart.NewWriter(pw)
	go func() {
		err := form.WriteField("metadata", string(meta))
		if err == nil {
			h := textproto.MIMEHeader{}
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filepath.Base(path)))
			h.Set("Content-Type", "application/zip")
			var part io.Writer
			if part, err = form.CreatePart(h); err == nil {
				_, err = io.Copy(part, f)
			}
		}
		if err == nil {
			err = form.Close()
		}
		_ = pw.CloseWithError(err)
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/projects/"+strconv.Itoa(projectID)+"/upload-file", pr)
	if err != nil {
		_ = pr.Close()
		return 0, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", appversion.UserAgent(c.version))
	req.Header.Set("X-Api-Token", c.token)
	res, err := c.client.Do(req)
	if err != nil {
		_ = pr.CloseWithError(err)
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(res.Body, maxSiteReply))
	if err := siteError(curseForgeName, "upload", res, reply); err != nil {
		return 0, err
	}
	var done struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(bytes.NewReader(reply)).Decode(&done); err != nil || done.ID <= 0 {
		return 0, errors.New("curseforge accepted the file but named no file id")
	}
	return done.ID, nil
}

// SetCurseForgeToken keeps the author's CurseForge upload token in the keyring.
func (s *Service) SetCurseForgeToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return usererr.Wrap(usererr.Invalid, errors.New("the token is empty"))
	}
	return secret.Set(curseForgeKeyring, token)
}

// HasCurseForgeToken reports whether an upload token is kept.
func (s *Service) HasCurseForgeToken() bool {
	token, err := secret.Get(curseForgeKeyring)
	return err == nil && token != ""
}

// ForgetCurseForgeToken removes the kept upload token.
func (s *Service) ForgetCurseForgeToken() error { return secret.Delete(curseForgeKeyring) }
