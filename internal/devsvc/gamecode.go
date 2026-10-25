package devsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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

// gameAssembly is the game's own code, Assembly-CSharp.dll under the install the project builds against.
func (s *Service) gameAssembly(gameID, dir string) (string, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return "", err
	}
	props, _ := fsx.ReadFile(filepath.Join(p.Dir, "Config.Build.user.props"))
	root := readProp(string(props), "GameRootDir")
	found, _ := filepath.Glob(filepath.Join(root, "*_Data", "Managed", "Assembly-CSharp.dll"))
	if root == "" || len(found) == 0 {
		return "", usererr.Wrap(usererr.NotFound, errors.New("the game's Assembly-CSharp.dll was not found under the project's game folder"))
	}
	return found[0], nil
}

// GameCode searches the game's types and methods: every word of query in "Type::Method", ignoring case. The game's
// members are read once per version of its assembly.
func (s *Service) GameCode(gameID, dir, query string) ([]dotnet.Member, error) {
	path, err := s.gameAssembly(gameID, dir)
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

// Decompile returns one of the game's types as C#, through ILSpy's command line (ilspycmd).
func (s *Service) Decompile(ctx context.Context, gameID, dir, typ string) (string, error) {
	if !typeName.MatchString(typ) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a type name", typ))
	}
	path, err := s.gameAssembly(gameID, dir)
	if err != nil {
		return "", err
	}
	tool, ok := s.ilspy()
	if !ok {
		return "", usererr.Wrap(usererr.NotFound, errors.New("ILSpy's command line (ilspycmd) is not installed"))
	}
	out, err := s.run(ctx, filepath.Dir(path), tool, "-t", typ, path)
	if err != nil {
		return "", err
	}
	return string(out), nil
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
