package source

import "context"

// Standing is a site's own flag that a project is no longer a good thing to install: it was archived, taken down or
// marked abandoned. State is a short word and Reason what the site says, empty when it says nothing more.
type Standing struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// StandingChecker is a source that reports the flags of many installed projects in as few requests as it can. ids are
// the projects as the profile records them; the answer holds only the ones the site flags, keyed by that same id.
type StandingChecker interface {
	Standings(ctx context.Context, mortarVersion string, ids []string) (map[string]Standing, error)
}
