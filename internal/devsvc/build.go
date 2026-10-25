package devsvc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/nowindow"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// SourceEvent is emitted with the game when a watched project's source changes, so the window rebuilds it.
const SourceEvent = "dev:source"

const buildLogLines = 200

// Problem is one compiler error or warning, at its file and line.
type Problem struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// BuildResult is a build's outcome: its problems, each once, and the end of its output.
type BuildResult struct {
	OK       bool      `json:"ok"`
	Problems []Problem `json:"problems"`
	Log      string    `json:"log"`
}

// MSBuild's canonical form: file(line,col): error CODE: text [project].
var msbuildLine = regexp.MustCompile(`^\s*(.+?)\((\d+),(\d+)(?:,\d+,\d+)?\): (error|warning) ([A-Za-z]+\d+): (.*?)(?: \[[^\]]+\])?$`)

// ParseProblems reads MSBuild's output for errors and warnings, each once and in order.
func ParseProblems(out string) []Problem {
	problems := []Problem{}
	for line := range strings.Lines(out) {
		m := msbuildLine.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
		if m == nil {
			continue
		}
		row, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		p := Problem{File: m[1], Line: row, Column: col, Severity: m[4], Code: m[5], Message: m[6]}
		if !slices.Contains(problems, p) {
			problems = append(problems, p)
		}
	}
	return problems
}

var building sync.Map // project dir → *sync.Mutex

// Build runs a Debug build of the project in dir; its output folder installs into the development profile.
func (s *Service) Build(ctx context.Context, gameID, dir string) (BuildResult, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return BuildResult{}, err
	}
	stored, _ := building.LoadOrStore(p.Dir, &sync.Mutex{})
	if mu, ok := stored.(*sync.Mutex); ok {
		mu.Lock()
		defer mu.Unlock()
	}
	// Dependencies installed since they were added are declared on the plugin before it compiles.
	if err := s.syncDependencies(gameID, p); err != nil {
		log.Printf("devsvc: declaring %s's dependencies: %v", p.Name, err)
	}
	out, err := s.run(ctx, p.Dir, "dotnet", "build", "-c", "Debug", "-nologo")
	if _, missing := errors.AsType[*exec.Error](err); missing {
		return BuildResult{}, usererr.Wrap(usererr.NotFound, errors.New("install the .NET SDK to build mods"))
	}
	return BuildResult{OK: err == nil, Problems: ParseProblems(string(out)), Log: tail(string(out))}, nil
}

// SetWatch turns rebuilding on every source change on or off for the project in dir.
func (s *Service) SetWatch(gameID, dir string, on bool) error {
	return s.Settings.SetDevProjectWatch(gameID, dir, on)
}

// WatchedSources are the source folders of the projects being watched, by game, for the folder watch.
// ponytail: only src/<Name> itself is watched, not its subfolders; walk them when a template nests code.
func WatchedSources(cur settings.Settings, games []string) map[string][]string {
	out := map[string][]string{}
	for _, g := range games {
		for _, p := range cur.GamePrefs(g).DevProjects {
			if p.Watch {
				out[g] = append(out[g], filepath.Join(p.Dir, "src", p.Name))
			}
		}
	}
	return out
}

// OpenProject shows the project's folder; OpenAt opens one of its files.
func (s *Service) OpenProject(gameID, dir string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	return datadir.Open(p.Dir)
}

// OpenAt opens file, which must be inside the project in dir, in the first code editor found (VS Code, VSCodium,
// Rider) at line and column, else in the system's editor for it.
func (s *Service) OpenAt(gameID, dir, file string, line, column int) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(p.Dir, file)
	}
	if !datadir.UnderRoot(p.Dir, file) {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not in the project", file))
	}
	at := fmt.Sprintf("%s:%d:%d", file, max(line, 1), max(column, 1))
	if startEditor([][]string{{"code", "--goto", at}, {"codium", "--goto", at}, {"rider", "--line", strconv.Itoa(max(line, 1)), file}}) {
		return nil
	}
	return datadir.Open(file)
}

// OpenEditor opens the project's folder in the first code editor found (VS Code, VSCodium, Rider), else in the file
// manager.
func (s *Service) OpenEditor(gameID, dir string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if startEditor([][]string{{"code", p.Dir}, {"codium", p.Dir}, {"rider", p.Dir}}) {
		return nil
	}
	return datadir.Open(p.Dir)
}

// startEditor runs the first of the editor commands that is installed and starts, and reports whether one did.
func startEditor(commands [][]string) bool {
	for _, editor := range commands {
		if _, err := exec.LookPath(editor[0]); err != nil {
			continue
		}
		cmd := exec.CommandContext(context.Background(), editor[0], editor[1:]...) // #nosec G204 -- fixed editors; paths are inside the project
		nowindow.Set(cmd)
		if err := cmd.Start(); err == nil {
			go func() { _ = cmd.Wait() }()
			return true
		}
	}
	return false
}

func (s *Service) project(gameID, dir string) (settings.DevProject, error) {
	for _, p := range s.Settings.Get().GamePrefs(gameID).DevProjects {
		if p.Dir == dir {
			return p, nil
		}
	}
	return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no mod project in %s", dir))
}
