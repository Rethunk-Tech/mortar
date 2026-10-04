package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const trackedCacheTTL = 30 * time.Second

// MessageError is a non-200 API answer that carried a JSON message (endorse-without-download uses this).
type MessageError struct {
	Code    int
	Status  string
	Message string
}

func (e *MessageError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "nexus answered " + e.Status
}

// EndorseStatus is the signed-in user's endorsement of one mod.
type EndorseStatus string

const (
	EndorseUndecided EndorseStatus = "Undecided"
	EndorseEndorsed  EndorseStatus = "Endorsed"
	EndorseAbstained EndorseStatus = "Abstained"
)

// TrackedMod is one entry from GET /v1/user/tracked_mods.json.
type TrackedMod struct {
	ModID      int    `json:"modId"`
	DomainName string `json:"domainName"`
}

type trackedCache struct {
	mu      sync.Mutex
	at      time.Time
	mods    []TrackedMod
	hasList bool
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	code, status, raw, err := c.roundTrip(ctx, method, path, body)
	if err != nil {
		return err
	}
	switch code {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return c.rateLimited()
	default:
		var msg struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &msg)
		if msg.Message != "" {
			return &MessageError{Code: code, Status: status, Message: msg.Message}
		}
		return &StatusError{Code: code, Status: status}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) endorse(ctx context.Context, path, version string) (EndorseStatus, error) {
	var raw struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	body := struct {
		Version string `json:"version"`
	}{Version: version}
	if err := c.do(ctx, http.MethodPost, path, body, &raw); err != nil {
		return "", err
	}
	return EndorseStatus(raw.Status), nil
}

// Endorse records an endorsement of modID at version for the signed-in user.
func (c *Client) Endorse(ctx context.Context, modID int, version string) (EndorseStatus, error) {
	return c.endorse(ctx, fmt.Sprintf("/v1/games/%s/mods/%d/endorse.json", Game, modID), version)
}

// Abstain withdraws an endorsement of modID at version for the signed-in user.
func (c *Client) Abstain(ctx context.Context, modID int, version string) (EndorseStatus, error) {
	return c.endorse(ctx, fmt.Sprintf("/v1/games/%s/mods/%d/abstain.json", Game, modID), version)
}

func (c *Client) cache() *trackedCache {
	if c.track != nil {
		return c.track
	}
	c.track = &trackedCache{}
	return c.track
}

// TrackedMods lists mods the signed-in user tracks. The list is reused for trackedCacheTTL.
func (c *Client) TrackedMods(ctx context.Context) ([]TrackedMod, error) {
	cache := c.cache()
	cache.mu.Lock()
	if cache.hasList && c.now().Sub(cache.at) < trackedCacheTTL {
		out := append([]TrackedMod(nil), cache.mods...)
		cache.mu.Unlock()
		return out, nil
	}
	cache.mu.Unlock()
	var raw []struct {
		ModID      int    `json:"mod_id"`
		DomainName string `json:"domain_name"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/user/tracked_mods.json", nil, &raw); err != nil {
		return nil, err
	}
	mods := make([]TrackedMod, 0, len(raw))
	for _, m := range raw {
		mods = append(mods, TrackedMod{ModID: m.ModID, DomainName: m.DomainName})
	}
	cache.mu.Lock()
	cache.mods = mods
	cache.at = c.now()
	cache.hasList = true
	cache.mu.Unlock()
	return append([]TrackedMod(nil), mods...), nil
}

type trackBody struct {
	DomainName string `json:"domain_name"`
	ModID      int    `json:"mod_id"`
}

func (c *Client) invalidateTracked() {
	cache := c.cache()
	cache.mu.Lock()
	cache.hasList = false
	cache.mods = nil
	cache.mu.Unlock()
}

func (c *Client) setTracked(ctx context.Context, method string, modID int) error {
	body := trackBody{DomainName: Game, ModID: modID}
	if err := c.do(ctx, method, "/v1/user/tracked_mods.json", body, nil); err != nil {
		var msg *MessageError
		if method == http.MethodPost && errors.As(err, &msg) && msg.Code == http.StatusUnprocessableEntity {
			c.invalidateTracked()
			return nil
		}
		return err
	}
	c.invalidateTracked()
	return nil
}

// Track starts tracking modID for the signed-in user.
func (c *Client) Track(ctx context.Context, modID int) error {
	return c.setTracked(ctx, http.MethodPost, modID)
}

// Untrack stops tracking modID for the signed-in user.
func (c *Client) Untrack(ctx context.Context, modID int) error {
	return c.setTracked(ctx, http.MethodDelete, modID)
}
