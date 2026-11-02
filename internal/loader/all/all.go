// Package all registers every loader driver; a catalog game names its loaders by id, so the registry needs them all.
package all

import (
	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/folder"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/larian"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/melonloader"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/smapi"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/vslog"
)
