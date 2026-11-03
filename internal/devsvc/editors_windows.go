//go:build windows

package devsvc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/nowindow"
	"golang.org/x/sys/windows/registry"
)

// The file types a C# project's editor registers for, which Windows records and desktop Linux types as plain text.
var projectFileTypes = []string{".sln", ".slnx", ".csproj"}

const fileExtsKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\`

// platformEditors adds what Windows itself knows: every Visual Studio from its installer's vswhere, and every program
// registered for a project's file types, as "Open with" lists them.
func platformEditors() []Editor {
	return append(visualStudios(), associatedEditors(projectFileTypes)...)
}

func visualStudios() []Editor {
	vswhere := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft Visual Studio", "Installer", "vswhere.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, vswhere, "-all", "-prerelease", "-products", "*", "-format", "json", "-utf8") //nolint:gosec // G204: the Visual Studio installer's own tool, at its documented path, with fixed arguments.
	nowindow.Set(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseVSWhere(out)
}

// associatedEditors reads the programs registered for the file types. VSLauncher only picks a Visual Studio for a
// solution, and vswhere names each one it could pick.
func associatedEditors(types []string) []Editor {
	var out []Editor
	seen := map[string]bool{}
	add := func(command string) {
		exe := commandExe(command)
		key := strings.ToLower(exe)
		if exe == "" || seen[key] || strings.EqualFold(filepath.Base(exe), "VSLauncher.exe") {
			return
		}
		seen[key] = true
		if _, err := os.Stat(exe); err == nil {
			out = append(out, Editor{Name: editorLabel(exe), Path: exe})
		}
	}
	for _, fileType := range types {
		for _, progID := range progIDs(fileType) {
			add(openCommand(progID))
		}
		for _, app := range openWithApps(fileType) {
			add(openCommand(`Applications\` + app))
		}
	}
	return out
}

func progIDs(fileType string) []string {
	ids := valueNames(registry.CLASSES_ROOT, fileType+`\OpenWithProgids`)
	ids = append(ids, valueNames(registry.CURRENT_USER, fileExtsKey+fileType+`\OpenWithProgids`)...)
	if id := stringValue(registry.CLASSES_ROOT, fileType, ""); id != "" {
		ids = append(ids, id)
	}
	if id := stringValue(registry.CURRENT_USER, fileExtsKey+fileType+`\UserChoice`, "ProgId"); id != "" {
		ids = append(ids, id)
	}
	return ids
}

func openWithApps(fileType string) []string {
	apps := subKeyNames(registry.CLASSES_ROOT, fileType+`\OpenWithList`)
	list := fileExtsKey + fileType + `\OpenWithList`
	for _, name := range valueNames(registry.CURRENT_USER, list) {
		if name != "MRUList" {
			apps = append(apps, stringValue(registry.CURRENT_USER, list, name))
		}
	}
	return apps
}

func openCommand(class string) string {
	command := stringValue(registry.CLASSES_ROOT, class+`\shell\open\command`, "")
	if expanded, err := registry.ExpandString(command); err == nil {
		return expanded
	}
	return command
}

func stringValue(root registry.Key, path, name string) string {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = key.Close() }()
	value, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return value
}

func valueNames(root registry.Key, path string) []string {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer func() { _ = key.Close() }()
	names, err := key.ReadValueNames(-1)
	if err != nil {
		return nil
	}
	return names
}

func subKeyNames(root registry.Key, path string) []string {
	key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer func() { _ = key.Close() }()
	names, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil
	}
	return names
}

// editorLabel names a program found only by its path.
func editorLabel(program string) string {
	base := filepath.Base(program)
	switch editorKindOf(program) {
	case editorRider:
		return "Rider"
	case editorVisualStudio:
		return "Visual Studio"
	case editorCodeFamily, editorOther:
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
