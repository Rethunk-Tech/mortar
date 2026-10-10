package nexus

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// MaxCollectionArchive caps the curator's collection archive Mortar reads into memory.
const MaxCollectionArchive = 64 << 20

// CollectionArchive downloads a collection revision's curator archive (collection.json plus bundled files).
// downloadLink is the API path Nexus reports for the revision. The archive is the curator's manifest, not a mod
// file; callers still use it only for accounts that asked for the import.
func (c *Client) CollectionArchive(ctx context.Context, downloadLink string) ([]byte, error) {
	if !strings.HasPrefix(downloadLink, "/v2/collections/") || strings.Contains(downloadLink, "..") {
		return nil, fmt.Errorf("unexpected collection download link %q", downloadLink)
	}
	var res struct {
		Links []struct {
			URI string `json:"URI"`
		} `json:"download_links"`
		Link struct {
			URI string `json:"URI"`
		} `json:"download_link"`
	}
	if err := c.get(ctx, downloadLink, true, &res); err != nil {
		return nil, err
	}
	uri := res.Link.URI
	if len(res.Links) > 0 {
		uri = res.Links[0].URI
	}
	if !strings.HasPrefix(uri, "https://") {
		return nil, errors.New("nexus gave no collection download link")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", source.UserAgent(c.version))
	hc := cmp.Or(c.HTTP, http.DefaultClient)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	b, err := fsx.ReadCapped(resp.Body, MaxCollectionArchive)
	if err != nil {
		return nil, fmt.Errorf("the collection archive: %w", err)
	}
	return b, nil
}
