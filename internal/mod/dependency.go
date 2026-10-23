package mod

import "cmp"

// Dependency is one mod another mod needs; Required defaults to true when the manifest omits IsRequired.
type Dependency struct {
	UniqueID       string
	MinimumVersion string
	Required       bool
	// Format is the ID format UniqueID belongs to; empty is SMAPI.
	Format string
}

// ModID is the needed mod as an ID.
func (d Dependency) ModID() ID { return NewID(cmp.Or(d.Format, FormatSMAPI), d.UniqueID) }

// NewDependency is a Dependency on id.
func NewDependency(id ID, minimumVersion string, required bool) Dependency {
	d := Dependency{UniqueID: id.Local(), MinimumVersion: minimumVersion, Required: required}
	if f := id.Format(); f != FormatSMAPI {
		d.Format = f
	}
	return d
}
