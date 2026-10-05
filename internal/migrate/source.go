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
}

// ProfileInfo is the selectable identity of an external profile.
type ProfileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Mods counts the mods the profile would import.
	Mods int `json:"mods"`
}
