// Package hostedlink is the app's client for hosted share short codes on mortar.rethunk.tech.
package hostedlink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
)

// DefaultBaseURL is the hosted share service the app talks to.
const DefaultBaseURL = "https://mortar.rethunk.tech"

// Record is what the app keeps for a profile's hosted short link.
// Token is a secret and must be stored in the data dir, never in
// exported .mortar files or sync payloads.
type Record struct {
	Code    string `json:"code"`
	Token   string `json:"token"`
	Version int    `json:"version"`
}

// CreateResponse is POST /api/shares.
type CreateResponse struct {
	Code    string `json:"code"`
	Token   string `json:"token"`
	Version int    `json:"version"`
}

// UpdateResponse is PUT /api/shares/<code>.
type UpdateResponse struct {
	Version int `json:"version"`
}

// Client talks to the hosted share HTTP API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) base() string {
	if c != nil && c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

type payloadBody struct {
	Payload string `json:"payload"`
}

// Create POSTs a share payload and returns the short code, owner token, and version.
func (c *Client) Create(ctx context.Context, payload string) (CreateResponse, error) {
	var out CreateResponse
	err := c.do(ctx, http.MethodPost, "/api/shares", "", payloadBody{Payload: payload}, http.StatusCreated, &out)
	return out, err
}

// Update PUTs a new payload for an existing code using the owner token.
func (c *Client) Update(ctx context.Context, code, token, payload string) (UpdateResponse, error) {
	var out UpdateResponse
	path := "/api/shares/" + code
	err := c.do(ctx, http.MethodPut, path, token, payloadBody{Payload: payload}, http.StatusOK, &out)
	return out, err
}

// Ready reports whether the share API answers. The site alone answers its own 404 page there, so a missing
// service reads as not ready rather than as an error.
func (c *Client) Ready(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/api/shares/ready", nil)
	if err != nil {
		return false
	}
	res, err := c.http().Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = res.Body.Close() }()
	var body struct {
		Ready bool `json:"ready"`
	}
	return res.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(res.Body, 1024)).Decode(&body) == nil && body.Ready
}

// ShortURL is the public page for a hosted share.
func ShortURL(game, code string) string {
	return DefaultBaseURL + "/" + game + "/s/" + code
}

func (c *Client) do(ctx context.Context, method, path, token string, body any, want int, dest any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base()+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	slurp, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != want {
		msg := strings.TrimSpace(string(slurp))
		if msg == "" {
			msg = res.Status
		}
		return &httpkit.StatusError{Code: res.StatusCode, Status: fmt.Sprintf("%s: %s", res.Status, msg)}
	}
	if dest == nil || len(slurp) == 0 {
		return nil
	}
	return json.Unmarshal(slurp, dest)
}
