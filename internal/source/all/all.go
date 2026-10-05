// Package all registers every source driver.
package all

import (
	_ "github.com/Rethunk-Tech/mortar/internal/source/github"
	_ "github.com/Rethunk-Tech/mortar/internal/source/itch"
	_ "github.com/Rethunk-Tech/mortar/internal/source/modrinth"
	_ "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	_ "github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)
