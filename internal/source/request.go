package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// StatusError is a site answering with an HTTP status that is not 200. Auth is true for 401 and 403, which a caller
// may read as "not allowed" for one request or as a bad key.
type StatusError struct {
	Service string
	Code    int
	Status  string
}

func (e *StatusError) Error() string { return e.Service + " answered " + e.Status }

func (e *StatusError) Auth() bool {
	return e.Code == http.StatusUnauthorized || e.Code == http.StatusForbidden
}

// Request is one JSON API call. Header sets the driver's own authentication headers; Service names the site in errors.
type Request struct {
	Service   string
	Client    *http.Client
	Method    string
	URL       string
	Params    url.Values
	Body      any
	UserAgent string
	Header    func(http.Header)
}

// DoJSON sends r and decodes a 200 answer into out. A 429 is a *BusyError and any other status a *StatusError; the
// body of those is never parsed.
func DoJSON(ctx context.Context, r Request, out any) error {
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	u := r.URL
	if len(r.Params) > 0 {
		u += "?" + r.Params.Encode()
	}
	var payload io.Reader
	if r.Body != nil {
		raw, err := json.Marshal(r.Body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, u, payload)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", r.UserAgent)
	req.Header.Set("Accept", "application/json")
	if r.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.Header != nil {
		r.Header(req.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return Busy(r.Service, resp)
	default:
		return &StatusError{Service: r.Service, Code: resp.StatusCode, Status: resp.Status}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxBody))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: %w", strings.ToLower(r.Service), err)
	}
	return nil
}
