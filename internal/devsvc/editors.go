package devsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// Editor is one installed program a project can open in; each installed version is its own.
type Editor struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// systemEditor as the codeEditor setting opens a project with the system's default app for its solution. An empty
// setting is the first editor found, and anything else is the program at that path.
const systemEditor = "system"

// Where each editor installs, as globs under environment variables; the order is the order Automatic tries them.
var editorSites = []struct {
	name    string
	bin     string
	windows []string
	unix    []string
}{
	{
		name: "VS Code",
		bin:  "code",
		windows: []string{
			"${LOCALAPPDATA}/Programs/Microsoft VS Code/bin/code.cmd",
			"${ProgramFiles}/Microsoft VS Code/bin/code.cmd",
		},
		unix: []string{
			"/snap/bin/code",
			"/var/lib/flatpak/exports/bin/com.visualstudio.code",
			"${HOME}/.local/share/flatpak/exports/bin/com.visualstudio.code",
		},
	},
	{
		name: "VS Code Insiders",
		bin:  "code-insiders",
		windows: []string{
			"${LOCALAPPDATA}/Programs/Microsoft VS Code Insiders/bin/code-insiders.cmd",
			"${ProgramFiles}/Microsoft VS Code Insiders/bin/code-insiders.cmd",
		},
		unix: []string{"/snap/bin/code-insiders"},
	},
	{
		name: "VSCodium",
		bin:  "codium",
		windows: []string{
			"${LOCALAPPDATA}/Programs/VSCodium/bin/codium.cmd",
			"${ProgramFiles}/VSCodium/bin/codium.cmd",
		},
		unix: []string{
			"/snap/bin/codium",
			"/var/lib/flatpak/exports/bin/com.vscodium.codium",
			"${HOME}/.local/share/flatpak/exports/bin/com.vscodium.codium",
		},
	},
	{
		name: "Rider",
		bin:  "rider",
		windows: []string{
			"${LOCALAPPDATA}/JetBrains/Toolbox/apps/Rider/bin/rider64.exe",
			"${LOCALAPPDATA}/JetBrains/Toolbox/apps/Rider/ch-*/*/bin/rider64.exe",
			"${LOCALAPPDATA}/Programs/Rider*/bin/rider64.exe",
			"${ProgramFiles}/JetBrains/JetBrains Rider*/bin/rider64.exe",
		},
		unix: []string{
			"${HOME}/.local/share/JetBrains/Toolbox/apps/rider/bin/rider.sh",
			"${HOME}/.local/share/JetBrains/Toolbox/apps/rider/ch-*/*/bin/rider.sh",
			"${HOME}/.local/share/JetBrains/Toolbox/apps/Rider/ch-*/*/bin/rider.sh",
			"/opt/rider*/bin/rider.sh",
			"/opt/JetBrains Rider*/bin/rider.sh",
			"/snap/bin/rider",
			"/var/lib/flatpak/exports/bin/com.jetbrains.Rider",
			"${HOME}/.local/share/flatpak/exports/bin/com.jetbrains.Rider",
		},
	},
	{
		name:    "Cursor",
		bin:     "cursor",
		windows: []string{"${LOCALAPPDATA}/Programs/cursor/resources/app/bin/cursor.cmd"},
	},
}

// editorHost is what finding editors reads from the computer.
type editorHost struct {
	windows  bool
	getenv   func(string) string
	lookPath func(string) (string, error)
}

// find lists each known editor on PATH and in its install folders, the newest version of a folder pattern first.
func (h editorHost) find() []Editor {
	var out []Editor
	for _, site := range editorSites {
		if path, err := h.lookPath(site.bin); err == nil {
			out = h.add(out, Editor{Name: site.name, Path: path})
		}
		patterns := site.unix
		if h.windows {
			patterns = site.windows
		}
		for _, pattern := range patterns {
			found, err := filepath.Glob(filepath.FromSlash(os.Expand(pattern, h.getenv)))
			if err != nil {
				continue
			}
			slices.Reverse(found)
			for _, path := range found {
				out = h.add(out, Editor{Name: versioned(site.name, path), Path: path})
			}
		}
	}
	return out
}

// add keeps one entry per program: PATH often holds a link to a folder that is searched too.
func (h editorHost) add(list []Editor, e Editor) []Editor {
	key := func(path string) string {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		if h.windows {
			return strings.ToLower(path)
		}
		return path
	}
	if slices.ContainsFunc(list, func(have Editor) bool { return key(have.Path) == key(e.Path) }) {
		return list
	}
	return append(list, e)
}

// versioned adds the version a JetBrains install records, so two versions of an editor can be told apart.
func versioned(name, program string) string {
	data, err := safefs.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(program)), "product-info.json"))
	if err != nil {
		return name
	}
	var info struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &info) != nil || info.Version == "" {
		return name
	}
	return name + " " + info.Version
}

// parseVSWhere reads vswhere's JSON: one entry per Visual Studio installed, versions side by side.
func parseVSWhere(data []byte) []Editor {
	var installs []struct {
		DisplayName string `json:"displayName"`
		ProductPath string `json:"productPath"`
		Catalog     struct {
			Version string `json:"productDisplayVersion"`
		} `json:"catalog"`
	}
	if json.Unmarshal(data, &installs) != nil {
		return nil
	}
	out := make([]Editor, 0, len(installs))
	for _, install := range installs {
		if install.ProductPath == "" {
			continue
		}
		name := strings.TrimSpace(install.DisplayName + " " + install.Catalog.Version)
		out = append(out, Editor{Name: name, Path: install.ProductPath})
	}
	return out
}

// commandExe is the program a Windows shell\open\command value runs: quoted, or up to its ".exe" since an unquoted
// path may hold spaces.
func commandExe(command string) string {
	command = strings.TrimSpace(command)
	if rest, ok := strings.CutPrefix(command, `"`); ok {
		exe, _, _ := strings.Cut(rest, `"`)
		return exe
	}
	if at := strings.Index(strings.ToLower(command), ".exe"); at >= 0 {
		return command[:at+len(".exe")]
	}
	return ""
}

type editorKind int

const (
	editorOther editorKind = iota
	editorCodeFamily
	editorRider
	editorVisualStudio
)

func editorKindOf(program string) editorKind {
	base := strings.ToLower(filepath.Base(program))
	switch {
	case strings.Contains(base, "rider"):
		return editorRider
	case strings.HasPrefix(base, "devenv"):
		return editorVisualStudio
	case strings.HasPrefix(base, "code"), strings.HasPrefix(base, "codium"), strings.HasPrefix(base, "cursor"),
		strings.HasPrefix(base, "com.visualstudio.code"), strings.HasPrefix(base, "com.vscodium.codium"):
		return editorCodeFamily
	}
	return editorOther
}

// projectArgs opens a project: VS Code's family works on folders, and the rest open the solution when there is one.
func projectArgs(program, dir, solution string) []string {
	if editorKindOf(program) == editorCodeFamily || solution == "" {
		return []string{dir}
	}
	return []string{solution}
}

// fileArgs opens a file at a line where the program can be told one.
func fileArgs(program, file string, line, column int) []string {
	switch editorKindOf(program) {
	case editorCodeFamily:
		return []string{"--goto", fmt.Sprintf("%s:%d:%d", file, line, column)}
	case editorRider:
		return []string{"--line", fmt.Sprint(line), file}
	case editorVisualStudio:
		return []string{"/Edit", file}
	case editorOther:
	}
	return []string{file}
}

// solutionIn is the project's one solution file, or "" with none or a choice of several.
func solutionIn(dir string) string {
	var found []string
	for _, pattern := range []string{"*.sln", "*.slnx"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err == nil {
			found = append(found, matches...)
		}
	}
	if len(found) != 1 {
		return ""
	}
	return found[0]
}
