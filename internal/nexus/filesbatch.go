package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// filesBatch is how many mods one GraphQL request asks about.
const filesBatch = 50

// BatchFile is one file of a mod as Nexus's v2 API lists it.
type BatchFile struct {
	FileID   int    `json:"fileId"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Category string `json:"category"`
}

// FilesOf lists the files of many mods with one GraphQL request per 50 mods (aliased modFiles queries), instead
// of one request per mod.
func (c *Client) FilesOf(ctx context.Context, modIDs []int) (map[int][]BatchFile, error) {
	out := make(map[int][]BatchFile, len(modIDs))
	for start := 0; start < len(modIDs); start += filesBatch {
		batch := modIDs[start:min(start+filesBatch, len(modIDs))]
		if err := c.filesOf(ctx, batch, out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (c *Client) filesOf(ctx context.Context, modIDs []int, out map[int][]BatchFile) error {
	var q strings.Builder
	q.WriteString("{")
	for _, id := range modIDs {
		fmt.Fprintf(&q, " m%d: modFiles(modId: %d, gameId: %d) { fileId name version category }", id, id, GameID)
	}
	q.WriteString(" }")
	body, err := json.Marshal(map[string]string{"query": q.String()})
	if err != nil {
		return err
	}
	base := c.BaseURL
	if base == "" {
		base = BaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v2/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Apikey", c.key)
	req.Header.Set("Application-Name", "Mortar")
	req.Header.Set("Application-Version", c.version)
	req.Header.Set("User-Agent", "Mortar/"+c.version)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return &StatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	var raw struct {
		Data map[string][]BatchFile `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&raw); err != nil {
		return err
	}
	for _, id := range modIDs {
		if files, ok := raw.Data[fmt.Sprintf("m%d", id)]; ok {
			out[id] = files
		}
	}
	return nil
}
