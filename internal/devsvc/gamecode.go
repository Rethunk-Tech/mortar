package devsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/msbuild"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// maxCodeHits bounds a game code search; a narrower query finds the rest.
const maxCodeHits = 300

// typeName is a type's full name as Members gives it, the only form Decompile passes to ILSpy.
var typeName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.+` + "`" + `]*$`)

type gameCode struct {
	modified time.Time
	members  []dotnet.Member
}

// Assembly is a DLL the Game code view reads: the game's own, a dependency's in the development profile, or the
// project's build.
type Assembly struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	// Group is "game", "dependency" or "project".
	Group string `json:"group"`
}

// noise names the runtime's own assemblies, which a mod never hooks into.
var noise = regexp.MustCompile(`^(System(\..*)?|mscorlib|netstandard|Microsoft\..*|Mono\.Security)\.dll$`)

// Assemblies lists what Game code can open, the game's main assembly first: a Unity game's Managed folder, else the
// DLLs beside the game's executable; the development profile's packages; the project's own build.
func (s *Service) Assemblies(gameID, dir string) ([]Assembly, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return nil, err
	}
	props, _ := fsx.ReadFile(filepath.Join(p.Dir, "Config.Build.user.props"))
	root := msbuild.Property(string(props), "GameRootDir")
	out := []Assembly{}
	if root != "" {
		game, _ := filepath.Glob(filepath.Join(root, "*_Data", "Managed", "*.dll"))
		if len(game) == 0 {
			game, _ = filepath.Glob(filepath.Join(root, "*.dll"))
		}
		slices.SortStableFunc(game, func(a, b string) int {
			return cmpBool(filepath.Base(b) == "Assembly-CSharp.dll", filepath.Base(a) == "Assembly-CSharp.dll")
		})
		for _, path := range game {
			if !noise.MatchString(filepath.Base(path)) {
				out = append(out, Assembly{Label: filepath.Base(path), Path: path, Group: "game"})
			}
		}
	}
	if installed, err := s.Profiles.Installed(gameID, p.Profile); err == nil {
		for _, im := range installed {
			if im.Source.Kind != profile.KindThunderstore || im.Folder == "" {
				continue
			}
			for _, path := range dotnet.DLLs(im.Folder) {
				out = append(out, Assembly{Label: im.Source.Name + "/" + filepath.Base(path), Path: path, Group: "dependency"})
			}
		}
	}
	built, _ := filepath.Glob(filepath.Join(p.Output, "*.dll"))
	for _, path := range built {
		out = append(out, Assembly{Label: filepath.Base(path), Path: path, Group: "project"})
	}
	return out, nil
}

func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	default:
		return -1
	}
}

// assembly is path when Assemblies lists it, else the game's main assembly when path is empty; anything else is
// refused, so the view never reads a file the player did not pick from the list.
func (s *Service) assembly(gameID, dir, path string) (string, error) {
	all, err := s.Assemblies(gameID, dir)
	if err != nil {
		return "", err
	}
	if path == "" && len(all) > 0 && all[0].Group == "game" {
		return all[0].Path, nil
	}
	for _, a := range all {
		if a.Path == path && path != "" {
			return path, nil
		}
	}
	return "", usererr.Wrap(usererr.NotFound, errors.New("that assembly is not one Game code lists"))
}

// GameCode searches an assembly's types, methods and fields: every word of query in "Type::Method", ignoring case. The game's
// members are read once per version of its assembly.
func (s *Service) GameCode(gameID, dir, assembly, query string) ([]dotnet.Member, error) {
	path, err := s.assembly(gameID, dir, assembly)
	if err != nil {
		return nil, err
	}
	members, err := s.members(path)
	if err != nil {
		return nil, err
	}
	words := strings.Fields(strings.ToLower(query))
	hits := []dotnet.Member{}
	for _, m := range members {
		name := strings.ToLower(m.Type + "::" + m.Method)
		if !matchesAll(name, words) {
			continue
		}
		if hits = append(hits, m); len(hits) == maxCodeHits {
			break
		}
	}
	return hits, nil
}

func matchesAll(name string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(name, w) {
			return false
		}
	}
	return true
}

func (s *Service) members(path string) ([]dotnet.Member, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	s.codeMu.Lock()
	defer s.codeMu.Unlock()
	if c, ok := s.code[path]; ok && c.modified.Equal(st.ModTime()) {
		return c.members, nil
	}
	members, err := dotnet.Members(path)
	if err != nil {
		return nil, err
	}
	if s.code == nil {
		s.code = map[string]gameCode{}
	}
	s.code[path] = gameCode{modified: st.ModTime(), members: members}
	return members, nil
}

// Usage is a method that uses a member: in which assembly, as "Type::Method".
type Usage struct {
	Assembly string `json:"assembly"`
	Label    string `json:"label"`
	Caller   string `json:"caller"`
}

// Usages finds the methods in every assembly Assemblies lists that call, construct, load or store target
// ("Namespace.Type::Member").
func (s *Service) Usages(gameID, dir, target string) ([]Usage, error) {
	all, err := s.Assemblies(gameID, dir)
	if err != nil {
		return nil, err
	}
	out := []Usage{}
	for _, a := range all {
		callers, err := dotnet.Usages(a.Path, target)
		if err != nil {
			log.Printf("devsvc: reading %s for usages: %v", a.Label, err)
			continue
		}
		for _, c := range callers {
			out = append(out, Usage{Assembly: a.Path, Label: a.Label, Caller: c})
		}
	}
	return out, nil
}

// Decompile returns one of an assembly's types as C#, or as IL, through ILSpy's command line (ilspycmd).
func (s *Service) Decompile(ctx context.Context, gameID, dir, assembly, typ string, il bool) (string, error) {
	if !typeName.MatchString(typ) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a type name", typ))
	}
	path, err := s.assembly(gameID, dir, assembly)
	if err != nil {
		return "", err
	}
	tool, ok := s.ilspy()
	if !ok {
		return "", usererr.Wrap(usererr.NotFound, errors.New("ILSpy's command line (ilspycmd) is not installed"))
	}
	args := []string{"-t", typ, path}
	if il {
		args = append([]string{"-il"}, args...)
	}
	out, err := s.run(ctx, filepath.Dir(path), tool, args...)
	if err != nil {
		return "", err
	}
	if il {
		return ilBlock(string(out), typ), nil
	}
	return string(out), nil
}

// ilBlock cuts typ's .class block out of a whole assembly's IL, since ILSpy disassembles every type with -il; a
// nested type's block names it without its outer types. The whole listing stands when the block is not found.
func ilBlock(il, typ string) string {
	name := typ
	if _, inner, nested := strings.CutLast(typ, "+"); nested {
		name = inner
	}
	lines := strings.Split(il, "\n")
	start := slices.IndexFunc(lines, func(l string) bool {
		t := strings.TrimSpace(l)
		return strings.HasPrefix(t, ".class ") && strings.HasSuffix(t, " "+name)
	})
	if start < 0 {
		return il
	}
	end := slices.IndexFunc(lines[start:], func(l string) bool { return strings.TrimSpace(l) == "} // end of class "+name })
	if end < 0 {
		return il
	}
	return strings.Join(lines[start:start+end+1], "\n")
}

// HasDecompiler reports whether ilspycmd is installed.
func (s *Service) HasDecompiler() bool {
	_, ok := s.ilspy()
	return ok
}

// InstallDecompiler installs ilspycmd as a .NET global tool for the player.
func (s *Service) InstallDecompiler(ctx context.Context) error {
	_, err := s.run(ctx, s.Home, "dotnet", "tool", "install", "--global", "ilspycmd")
	return err
}

// ilspy finds ilspycmd on the PATH or in .NET's global tools folder, which a fresh install has not added to it yet.
func (s *Service) ilspy() (string, bool) {
	if p, err := exec.LookPath("ilspycmd"); err == nil {
		return p, true
	}
	name := "ilspycmd"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(s.Home, ".dotnet", "tools", name)
	if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
		return p, true
	}
	return "", false
}
