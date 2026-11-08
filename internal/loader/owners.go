package loader

import "github.com/Rethunk-Tech/go-bepinex/plugins"

// PluginOwners maps every way a loader log names a plugin (see plugins.Owners) to the installed package whose DLLs
// declare it, so a loader log and the window's Problems tab attribute a line the same way. A package is the identity;
// the GUIDs are read from its files, since a package may hold several plugins or none and can change a GUID between
// versions. A disabled package owns nothing: a log never blames a mod the game does not load.
func PluginOwners[T any](mods []T, folder func(T) string, enabled func(T) bool) map[string]T {
	var with []T
	var dirs []string
	for _, m := range mods {
		if f := folder(m); enabled(m) && f != "" {
			with, dirs = append(with, m), append(dirs, f)
		}
	}
	owners := make(map[string]T, len(with))
	for name, i := range plugins.Owners(dirs) {
		owners[name] = with[i]
	}
	return owners
}
