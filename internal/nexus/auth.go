package nexus

import (
	"context"
	"net/http"
)

// TokenSource returns the current OAuth access token, refreshing it when it is about to expire.
type TokenSource func(ctx context.Context) (string, error)

// WithBearer returns a client that sends an OAuth access token, which Nexus accepts in the Authorization header on
// both the REST and GraphQL APIs.
func (c *Client) WithBearer(token TokenSource) *Client { return c.with("", token) }

// WithKey returns a client that authenticates with a personal API key.
// Remove with the API-key path once OAuth ships.
func (c *Client) WithKey(key string) *Client { return c.with(key, nil) }

func (c *Client) authorize(ctx context.Context, req *http.Request) error {
	if c.bearer != nil {
		token, err := c.bearer(ctx)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	// Remove with the API-key path once OAuth ships.
	req.Header.Set("Apikey", c.key)
	return nil
}
