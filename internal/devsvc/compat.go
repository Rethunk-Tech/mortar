package devsvc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

var (
	// pluginGUID and csNamespace keep what goes into generated C# to names that cannot break out of a string or a
	// namespace declaration.
	pluginGUID  = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	csNamespace = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
	notIdent    = regexp.MustCompile(`[^A-Za-z0-9_]`)
)

// pluginSource is the source file below src that declares the mod's [BepInPlugin] class.
func pluginSource(src string) string {
	var found string
	_ = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		if d.IsDir() && (d.Name() == "obj" || d.Name() == "bin") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".cs") {
			if raw, err := fsx.ReadFile(path); err == nil && strings.Contains(string(raw), "[BepInPlugin(") {
				found = path
			}
		}
		return nil
	})
	return found
}

// declareDependency puts [BepInDependency] for guid above the [BepInPlugin] attribute, soft or hard, replacing an
// earlier declaration of the same plugin.
func declareDependency(src, guid string, soft bool) error {
	if !pluginGUID.MatchString(guid) {
		return nil
	}
	path := pluginSource(src)
	if path == "" {
		return nil
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return err
	}
	attr := fmt.Sprintf(`[BepInDependency("%s")]`, guid)
	if soft {
		attr = fmt.Sprintf(`[BepInDependency("%s", BepInDependency.DependencyFlags.SoftDependency)]`, guid)
	}
	body := string(raw)
	existing := regexp.MustCompile(`\[BepInDependency\("` + regexp.QuoteMeta(guid) + `"[^\]\n]*\)\]`)
	switch {
	case existing.MatchString(body):
		body = existing.ReplaceAllLiteralString(body, attr)
	default:
		at := strings.Index(body, "[BepInPlugin(")
		lineStart := strings.LastIndex(body[:at], "\n") + 1
		body = body[:lineStart] + body[lineStart:at] + attr + "\n" + body[lineStart:]
	}
	if body == string(raw) {
		return nil
	}
	return fsx.WriteFile(path, []byte(body), 0o644) // #nosec G306 -- a tracked source file
}

// compatTemplate keeps every use of a soft dependency's types in one class. The runtime resolves the types a method
// names when it compiles that method, so a method touching them must never be compiled while the dependency is absent:
// callers check Enabled first, and NoInlining keeps the body out of callers that the JIT would otherwise compile with it.
const compatTemplate = `using System.Runtime.CompilerServices;
using BepInEx.Bootstrap;

namespace %[1]s.Compat;

// %[2]s is a soft dependency: the mod runs without it. Only call into this class after checking Enabled, and keep
// every use of %[2]s's types inside it.
internal static class %[3]sCompat
{
    private static bool? enabled;

    internal static bool Enabled => enabled ??= Chainloader.PluginInfos.ContainsKey("%[4]s");

    [MethodImpl(MethodImplOptions.NoInlining | MethodImplOptions.NoOptimization)]
    internal static void Initialize()
    {
        // Calls into %[2]s go here.
    }
}
`

// writeCompat creates the compatibility class for a soft dependency once; the player's edits to it are kept.
func writeCompat(src, namespace, pkg, guid string) error {
	if !csNamespace.MatchString(namespace) || !pluginGUID.MatchString(guid) {
		return nil
	}
	_, name, _ := strings.Cut(pkg, "-")
	class := notIdent.ReplaceAllString(name, "")
	if class == "" {
		return nil
	}
	path := filepath.Join(src, "Compat", class+"Compat.cs")
	if _, err := os.Stat(path); err == nil || !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return fsx.WriteFile(path, []byte(fmt.Sprintf(compatTemplate, namespace, name, class, guid)), 0o644) // #nosec G306 -- a tracked source file
}
