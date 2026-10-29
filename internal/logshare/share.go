// Package logshare uploads a SMAPI log to smapi.io's parser through go-smapiweb.
package logshare

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-smapiweb"
)

// Upload posts text through c and returns the log's public link. The module's status error names no host, so it is
// worded here as the player has always read it.
func Upload(ctx context.Context, c *smapiweb.Client, text string, maxBytes int) (string, error) {
	link, err := c.UploadLog(ctx, text, maxBytes)
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok {
		return "", fmt.Errorf("smapi.io answered %s", se.Status)
	}
	return link, err
}
