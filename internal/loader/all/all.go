// Package all registers every loader driver; a catalog game names its loaders by id, so the registry needs them all.
package all

import (
	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/smapi"
)
