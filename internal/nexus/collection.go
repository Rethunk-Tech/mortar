package nexus

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
)

// MaxCollectionArchive caps the curator's collection archive Mortar reads into memory.
const MaxCollectionArchive = 64 << 20

// mirrorTimeout is how long the mirror, which sits outside Nexus's API and can stall, may stay silent: before the
// headers and between body reads. A slow but steady link is never cut off.
var mirrorTimeout = 2 * time.Minute

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
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stall := time.AfterFunc(mirrorTimeout, func() { cancel(errMirrorStalled) })
	defer stall.Stop()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", appversion.UserAgent(c.version))
	hc := cmp.Or(c.HTTP, http.DefaultClient)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, cmp.Or(context.Cause(ctx), err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	b, err := safefs.ReadCapped(idleReader{resp.Body, stall}, MaxCollectionArchive)
	// A stall cancels the request, and the mirror may then end the body cleanly before the read sees the cancel.
	if err == nil {
		err = context.Cause(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("the collection archive: %w", cmp.Or(context.Cause(ctx), err))
	}
	return b, nil
}

var errMirrorStalled = errors.New("the collection mirror stopped sending data")

// idleReader restarts the stall timer on every read that returns bytes.
type idleReader struct {
	r     io.Reader
	timer *time.Timer
}

func (i idleReader) Read(p []byte) (int, error) {
	n, err := i.r.Read(p)
	if n > 0 {
		i.timer.Reset(mirrorTimeout)
	}
	return n, err
}
