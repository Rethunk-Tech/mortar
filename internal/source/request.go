package source

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
)

// StatusError is a site answering with an HTTP status that is not 200. Auth is true for 401 and 403, which a caller
// may read as "not allowed" for one request or as a bad key.
type StatusError struct {
	Service string
	Code    int
	Status  string
}

func (e *StatusError) Error() string { return e.Service + " answered " + e.Status }

// Unwrap lets usererr and errors.Is(err, httpkit.ErrNotFound) classify the answer like any other HTTP status.
func (e *StatusError) Unwrap() error { return &httpkit.StatusError{Code: e.Code, Status: e.Status} }

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
	client := cmp.Or(r.Client, http.DefaultClient)
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
	method := cmp.Or(r.Method, http.MethodGet)
	req, err := http.NewRequestWithContext(ctx, method, u, payload)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", r.UserAgent)
	if r.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.Header != nil {
		r.Header(req.Header)
	}
	err = httpkit.DoJSON(client, req, out, httpkit.JSONOptions{Limit: MaxBody})
	if err == nil {
		return nil
	}
	if rl, ok := errors.AsType[*httpkit.RateLimitError](err); ok {
		return &BusyError{Source: r.Service, Reset: rl.Reset}
	}
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return &StatusError{Service: r.Service, Code: se.Code, Status: se.Status}
	}
	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	if errors.As(err, &syntax) || errors.As(err, &typed) {
		return fmt.Errorf("%s: %w", strings.ToLower(r.Service), errors.Unwrap(err))
	}
	return err
}
