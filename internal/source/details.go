package source

import "context"

// Details is the fuller page of one mod, read when the player opens it in Browse.
type Details struct {
	// Description is the mod's long text as Markdown (its README); empty when the site has none.
	Description string `json:"description"`
	// Changelog is the site's changelog as Markdown, when it keeps one apart from its versions.
	Changelog string `json:"changelog,omitempty"`
	// Versions are the published version numbers, newest first.
	Versions []string `json:"versions"`
	// Dependencies are the ids the newest version needs, as the site names them.
	Dependencies []string `json:"dependencies"`
	Categories   []string `json:"categories"`
}

// Detailer is a source that reads one mod's Details; key is the source's name for the game.
type Detailer interface {
	Details(ctx context.Context, key, id, mortarVersion string) (Details, error)
}
