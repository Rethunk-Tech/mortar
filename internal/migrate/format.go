// Package migrate reads external mod-manager profile layouts for import preview.
package migrate

// ModPreview is one mod exposed by an external profile import.
type ModPreview struct {
	UniqueID   string `json:"uniqueID"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Enabled    bool   `json:"enabled"`
	NexusModID int    `json:"nexusModID,omitempty"`
	SourcePath string `json:"sourcePath,omitempty"`
}

// ProfilePreview is one external profile and the mods it would import.
type ProfilePreview struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Source   string       `json:"source"`
	ModsPath string       `json:"modsPath"`
	Mods     []ModPreview `json:"mods"`
}
