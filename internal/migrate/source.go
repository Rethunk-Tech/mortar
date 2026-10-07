package migrate

const (
	KindStardrop = "stardrop"
	KindVortex   = "vortex"
	KindMO2      = "mo2"
)

// SourceInfo is one detected external mod manager and its profiles.
type SourceInfo struct {
	Kind     string        `json:"kind"`
	Name     string        `json:"name"`
	Profiles []ProfileInfo `json:"profiles"`
	// Error says why the manager's profiles could not be read, when its data was found but is unreadable.
	Error string `json:"error,omitempty"`
}

// ProfileInfo is the selectable identity of an external profile.
type ProfileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Mods counts the mods the profile would import.
	Mods int `json:"mods"`
}
