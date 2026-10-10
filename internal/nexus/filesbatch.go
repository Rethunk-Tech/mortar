package nexus

import (
	"context"
	"encoding/json"
	"fmt"
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
	// ReplacedBy is the author's file_updates successor. The batched query does not carry it, so it comes from a
	// mod's cached full file list when there is one.
	ReplacedBy int `json:"replacedBy,omitempty"`
}

// FilesOf lists the files of many mods with one GraphQL request per 50 mods (aliased modFiles queries), instead
// of one request per mod.
func (c *Client) FilesOf(ctx context.Context, t Title, modIDs []int) (map[int][]BatchFile, error) {
	out := make(map[int][]BatchFile, len(modIDs))
	for start := 0; start < len(modIDs); start += filesBatch {
		batch := modIDs[start:min(start+filesBatch, len(modIDs))]
		if err := c.filesOf(ctx, t, batch, out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (c *Client) filesOf(ctx context.Context, t Title, modIDs []int, out map[int][]BatchFile) error {
	var q strings.Builder
	q.WriteString("{")
	for _, id := range modIDs {
		fmt.Fprintf(&q, " m%d: modFiles(modId: %d, gameId: %d) { fileId name version category }", id, id, t.ID)
	}
	q.WriteString(" }")
	body, err := c.graphql(ctx, map[string]string{"query": q.String()})
	if err != nil {
		return err
	}
	var raw struct {
		Data map[string][]BatchFile `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	for _, id := range modIDs {
		if files, ok := raw.Data[fmt.Sprintf("m%d", id)]; ok {
			out[id] = files
		}
	}
	return nil
}
