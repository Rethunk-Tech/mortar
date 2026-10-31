package devsvc

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-desktopenv/nowindow"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// SourceEvent is emitted with the game and the changed folder when a watched project's source changes, so the window
// rebuilds that project.
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

// requireTrust refuses to run the project's build files until the user has agreed to it.
func requireTrust(p settings.DevProject) error {
	if p.Untrusted {
		return usererr.Wrap(usererr.Unconfirmed, fmt.Errorf("building %s runs its repository's code on this computer", p.Name))
	}
	return nil
}

// Trust records the user's yes to building the project in dir; builds, packs and watched rebuilds refuse until then.
func (s *Service) Trust(gameID, dir string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if p.Kind != "" && p.Untrusted {
		next := p
		next.Untrusted, next.Output = false, p.Dir
		return s.Settings.ReplaceDevProject(gameID, p, next)
	}
	if p.Untrusted {
		// The build output was kept out of the profile while the repository's committed files were unvetted.
		next := p
		next.Untrusted, next.Output = false, filepath.Join(p.Dir, "artifacts", "bin", p.Name, "debug")
		if err := os.MkdirAll(next.Output, 0o750); err != nil {
			return err
		}
		return s.Settings.ReplaceDevProject(gameID, p, next)
	}
	return s.Settings.TrustDevProject(gameID, dir)
}

// Build runs a Debug build of the project in dir; its output folder installs into the development profile.
func (s *Service) Build(ctx context.Context, gameID, dir string) (BuildResult, error) {
	p, err := s.project(gameID, dir)
	if err != nil {
		return BuildResult{}, err
	}
	if err := requireTrust(p); err != nil {
		return BuildResult{}, err
	}
	if p.Kind != "" {
		// An asset project's folder is what installs; there is nothing to compile.
		return BuildResult{OK: true}, nil
	}
	defer s.lockedBuild(ctx, gameID, p)()
	out, err := s.run(ctx, p.Dir, "dotnet", buildArgs(p, "Debug")...)
	if _, missing := errors.AsType[*exec.Error](err); missing {
		return BuildResult{}, usererr.Wrap(usererr.NotFound, errors.New("install the .NET SDK to build mods"))
	}
	if err == nil {
		removeStaleBundles(p)
	}
	return BuildResult{OK: err == nil, Problems: ParseProblems(string(out)), Log: tail(string(out))}, nil
}

// buildArgs are dotnet's arguments for a build in config. An adopted project names its project file and builds into
// the folder that installs; the template's own layout needs neither.
func buildArgs(p settings.DevProject, config string) []string {
	csproj, adopted := recordedCsproj(p.Dir)
	if !adopted {
		return []string{"build", "-c", config, "-nologo"}
	}
	out := p.Output
	if config != "Debug" {
		out = releaseOutput(p)
	}
	return []string{"build", csproj, "-c", config, "-nologo", "-o", out}
}

// lockedBuild takes the project's build lock and declares the dependencies installed since they were added on the
// plugin, so it compiles with them; the returned func releases the lock.
func (s *Service) lockedBuild(ctx context.Context, gameID string, p settings.DevProject) func() {
	stored, _ := building.LoadOrStore(p.Dir, &sync.Mutex{})
	unlock := func() {}
	if mu, ok := stored.(*sync.Mutex); ok {
		mu.Lock()
		unlock = mu.Unlock
	}
	if err := s.wire(ctx, p, func() error { return s.syncDependencies(gameID, p) }); err != nil {
		log.Printf("devsvc: declaring %s's dependencies: %v", p.Name, err)
	}
	return unlock
}

// removeStaleBundles deletes bundles from the output that assets/ no longer holds: the template's CopyAssetBundles only
// adds, and the whole output folder is what installs.
func removeStaleBundles(p settings.DevProject) {
	assets := filepath.Join(p.Dir, "assets")
	if _, err := os.Stat(assets); err != nil {
		return
	}
	built, _ := filepath.Glob(filepath.Join(p.Output, "*.bundle"))
	for _, b := range built {
		if _, err := os.Stat(filepath.Join(assets, filepath.Base(b))); errors.Is(err, fs.ErrNotExist) {
			if err := safefs.Remove(b); err != nil {
				log.Printf("devsvc: removing stale bundle %s: %v", b, err)
			}
		}
	}
}

// SetWatch turns rebuilding on every source change on or off for the project in dir.
func (s *Service) SetWatch(gameID, dir string, on bool) error {
	return s.Settings.SetDevProjectWatch(gameID, dir, on)
}

// WatchedSources are every project's source folders, by game, for the folder watch: a change rebuilds a project with
// build on save on and marks the others changed since their last build. The folder watch is not recursive, so every
// folder below src/<Name> and assets/ (build output aside) is listed, and Compat and assets/ even before they exist.
func WatchedSources(cur settings.Settings, games []string) map[string][]string {
	out := map[string][]string{}
	for _, g := range games {
		for _, p := range cur.GamePrefs(g).DevProjects {
			if p.Kind != "" {
				continue
			}
			src := filepath.Join(p.Dir, "src", p.Name)
			if csproj, ok := recordedCsproj(p.Dir); ok {
				src = filepath.Dir(csproj)
			}
			assets := filepath.Join(p.Dir, "assets")
			// Compat and assets are listed even before they exist, so a folder made later is watched from the next refresh.
			out[g] = append(out[g], src, filepath.Join(src, "Compat"), assets)
			for _, root := range []string{src, assets} {
				_ = filepath.WalkDir(root, func(dir string, d fs.DirEntry, err error) error {
					if err != nil {
						// A folder not made yet, or one that cannot be read, is skipped like build output.
						return fs.SkipDir
					}
					if !d.IsDir() {
						return nil
					}
					if dir != root && slices.Contains([]string{"bin", "obj"}, d.Name()) {
						return fs.SkipDir
					}
					if dir != root && dir != filepath.Join(src, "Compat") {
						out[g] = append(out[g], dir)
					}
					return nil
				})
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
	if !safefs.UnderRoot(p.Dir, file) {
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
		cmd.Env = hostenv.Environ()
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
			p.Kind = migrateKind(gameID, p.Kind)
			return p, nil
		}
	}
	return settings.DevProject{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no mod project in %s", dir))
}
