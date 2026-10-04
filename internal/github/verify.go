package github

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// ErrUnknown means SMAPI's update API could not be reached, so the source is neither verified nor contradicted.
var ErrUnknown = errors.New("SMAPI's update API is unreachable")

// Verify reports whether SMAPI's update API maps uniqueID to owner/repo. A GitHub install is trusted only then;
// (false, nil) is a known mismatch and ErrUnknown is not a verdict.
func Verify(ctx context.Context, m *meta.Client, uniqueID, owner, repo string) (bool, error) {
	got := m.CheckUpdates(ctx, meta.UpdateRequest{Mods: []meta.InstalledMod{{ID: uniqueID}}})
	if len(got) != 1 || !got[0].Known {
		return false, ErrUnknown
	}
	return strings.EqualFold(got[0].GitHubRepo, owner+"/"+repo), nil
}
