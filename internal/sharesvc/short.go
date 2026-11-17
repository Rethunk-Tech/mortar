package sharesvc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const defaultShareAPIBase = "https://mortar.rethunk.tech"

// apiBaseURL is the hosted share API origin. Tests replace it with an httptest server.
var apiBaseURL = defaultShareAPIBase

const maxShareAPIBody = 64<<10 + 2048

type shareAPIResponse struct {
	Code    string `json:"code"`
	Version int    `json:"version"`
	Game    string `json:"game"`
	Payload string `json:"payload"`
}

func fetchSharePayload(ctx context.Context, route shortRoute) (string, error) {
	path := "/api/shares/" + route.code
	if route.version > 0 {
		path += "/" + strconv.Itoa(route.version)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+path, nil)
	if err != nil {
		return "", usererr.New(usererr.Network, "offline")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", usererr.New(usererr.Network, "offline")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return "", usererr.New(usererr.NotFound, "unknown or expired share code")
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxShareAPIBody))
		return "", usererr.New(usererr.Network, "offline")
	}
	var body shareAPIResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxShareAPIBody)).Decode(&body); err != nil || body.Payload == "" {
		return "", usererr.Wrap(usererr.Damaged, errors.New("share payload could not be read"))
	}
	return body.Payload, nil
}
