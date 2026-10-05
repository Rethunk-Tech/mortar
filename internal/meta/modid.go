package meta

import "github.com/Rethunk-Tech/mortar/internal/mod"

// ModID is the mod as a mod.ID.
func (m Mod) ModID() mod.ID { return mod.SMAPI(m.UniqueID) }

// ModID is the needed mod as a mod.ID.
func (d Dependency) ModID() mod.ID { return mod.SMAPI(d.UniqueID) }
