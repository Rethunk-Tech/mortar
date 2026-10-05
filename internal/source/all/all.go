// Package all registers every source driver.
package all

import (
	_ "github.com/Rethunk-Tech/mortar/internal/source/github"
	_ "github.com/Rethunk-Tech/mortar/internal/source/moddrop"
	_ "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	_ "github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)
