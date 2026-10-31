package devsvc

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	nexusAPI = "https://api.nexusmods.com/v3"
	// nexusSingleMax is the largest file Nexus takes in one PUT; a bigger one goes up in parts.
	nexusSingleMax = 100 << 20
	// nexusMaxParts is S3's limit on the parts of one upload.
	nexusMaxParts = 10000
	// nexusPolls bounds the wait for Nexus to process an upload, at a growing interval.
	nexusPolls = 40
	nexusName  = "Nexus Mods"
)

// nexusAPIKey is the personal API key kept for Nexus Mods, an error that asks for one when there is none.
func nexusAPIKey() (string, error) {
	key, err := nexussvc.StoredAPIKey()
	if err != nil || key == "" {
		return "", usererr.Wrap(usererr.Permission, errors.New("add your Nexus Mods API key in Settings, under Accounts"))
	}
	return key, nil
}

// nexusUploader uploads a file to Nexus Mods through its v3 API, as Nexus's own upload action does.
type nexusUploader struct {
	client  *http.Client
	base    string
	key     string
	version string
	// singleMax is the largest file sent in one PUT; zero is nexusSingleMax.
	singleMax int64
	// poll is the first wait between checks of an upload being processed; each wait is one longer.
	poll time.Duration
}

func (n nexusUploader) call(ctx context.Context, method, path string, body, out any) error {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, n.base+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", appversion.UserAgent(n.version))
	req.Header.Set("Apikey", n.key)
	nexus.SetAppHeaders(req.Header, n.version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(res.Body, maxSiteReply))
	if err := siteError(nexusName, method+" "+path, res, reply); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(reply, out)
}

// nexusUpload is the upload session Nexus answers with; Parts and Complete are set for a multipart one.
type nexusUpload struct {
	Data struct {
		ID           string   `json:"id"`
		State        string   `json:"state"`
		PresignedURL string   `json:"presigned_url"`
		PartSize     int64    `json:"part_size_bytes"`
		PartURLs     []string `json:"part_presigned_urls"`
		CompleteURL  string   `json:"complete_presigned_url"`
	} `json:"data"`
}

// send PUTs or POSTs r's bytes to a storage URL Nexus gave and returns its ETag header.
func (n nexusUploader) send(ctx context.Context, method, rawURL, contentType string, body io.Reader, size int64) (string, error) {
	target, err := storageURL(rawURL, n.base)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return "", err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)
	res, err := n.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(res.Body, maxSiteReply))
	if err := siteError(nexusName, "storage", res, reply); err != nil {
		return "", err
	}
	return strings.Trim(res.Header.Get("ETag"), `"`), nil
}

// store creates an upload session for the file and sends its bytes: in one PUT up to nexusSingleMax, else in the parts
// Nexus asks for, completed with the ETag of each. It returns the upload's id.
func (n nexusUploader) store(ctx context.Context, path string) (string, error) {
	f, err := safefs.OpenRegular(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := info.Size()
	req := map[string]any{"filename": filepath.Base(path), "size_bytes": size}
	var session nexusUpload
	if size <= cmp.Or(n.singleMax, nexusSingleMax) {
		if err := n.call(ctx, http.MethodPost, "/uploads", req, &session); err != nil {
			return "", err
		}
		d := session.Data
		if d.ID == "" || d.PresignedURL == "" {
			return "", errors.New("nexus mods answered without an upload address")
		}
		_, err := n.send(ctx, http.MethodPut, d.PresignedURL, "application/octet-stream", io.NewSectionReader(f, 0, size), size)
		return d.ID, err
	}
	if err := n.call(ctx, http.MethodPost, "/uploads/multipart", req, &session); err != nil {
		return "", err
	}
	d := session.Data
	if d.ID == "" || d.CompleteURL == "" || d.PartSize <= 0 || len(d.PartURLs) == 0 || len(d.PartURLs) > nexusMaxParts ||
		int64(len(d.PartURLs))*d.PartSize < size {
		return "", errors.New("nexus mods asked for parts that do not cover the file")
	}
	var done strings.Builder
	done.WriteString("<CompleteMultipartUpload>\n")
	for i, partURL := range d.PartURLs {
		offset := int64(i) * d.PartSize
		length := min(d.PartSize, size-offset)
		if length <= 0 {
			return "", fmt.Errorf("nexus mods asked for part %d beyond the file", i+1)
		}
		etag, err := n.send(ctx, http.MethodPut, partURL, "application/octet-stream", io.NewSectionReader(f, offset, length), length)
		if err != nil {
			return "", err
		}
		if etag == "" {
			return "", fmt.Errorf("uploading part %d: no ETag", i+1)
		}
		fmt.Fprintf(&done, "  <Part>\n    <PartNumber>%d</PartNumber>\n    <ETag>%s</ETag>\n  </Part>\n", i+1, etag)
	}
	done.WriteString("</CompleteMultipartUpload>")
	_, err = n.send(ctx, http.MethodPost, d.CompleteURL, "application/xml", strings.NewReader(done.String()), int64(done.Len()))
	return d.ID, err
}

// wait polls the upload until Nexus has processed it.
func (n nexusUploader) wait(ctx context.Context, id string) error {
	for attempt := 1; attempt <= nexusPolls; attempt++ {
		var up nexusUpload
		if err := n.call(ctx, http.MethodGet, "/uploads/"+id, nil, &up); err != nil {
			return err
		}
		if up.Data.State == "available" {
			return nil
		}
		t := time.NewTimer(min(n.poll*time.Duration(attempt), 15*time.Second))
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return errors.New("nexus mods is still processing the upload; try again in a few minutes")
}

// upload sends the package as the next version of the mod file ids.FileID, archiving the one it replaces, and adds
// changelog under the mod page when there is one and the mod id is known.
func (n nexusUploader) upload(ctx context.Context, path, name string, ids NexusIDs, version, changelog string) error {
	id, err := n.store(ctx, path)
	if err != nil {
		return err
	}
	if err := n.call(ctx, http.MethodPost, "/uploads/"+id+"/finalise", nil, nil); err != nil {
		return err
	}
	if err := n.wait(ctx, id); err != nil {
		return err
	}
	err = n.call(ctx, http.MethodPost, "/mod-files/"+strconv.Itoa(ids.FileID)+"/versions", map[string]any{
		"upload_id": id, "name": name, "version": version, "file_category": "main", "archive_existing_file": true,
	}, nil)
	if err != nil || changelog == "" || ids.ModID == 0 {
		return err
	}
	if err := n.call(ctx, http.MethodPost, "/mods/"+strconv.Itoa(ids.ModID)+"/changelogs", map[string]any{"version": version, "changelog": changelog}, nil); err != nil {
		return fmt.Errorf("the file is up, but its changelog was not added: %w", err)
	}
	return nil
}

func nexusDomain(gameID string) string {
	g, _ := components.Game(gameID)
	return g.NexusDomain()
}

func nexusPage(gameID string, ids NexusIDs) string {
	if ids.ModID == 0 {
		return ""
	}
	return nexusmods.ModURL(nexusDomain(gameID), ids.ModID) + "?tab=files"
}
