// Package redmod deploys a profile's REDmods for Cyberpunk 2077: the game reads them only from the files the REDmod
// DLC's own tool compiles into r6/cache/modded, which the game then loads when it starts with -modded. The behaviour
// is Vortex's (cyberpunk2077_ext_redux: redmod.deploy.ts, redmodding.metadata.ts, load_order.ts).
package redmod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// Paths below the install root.
const (
	toolPath      = "tools/redmod/bin/redMod.exe"
	launcherPath  = "REDprelauncher.exe"
	metadataPath  = "tools/redmod/metadata.json"
	modListPath   = "V2077/modlist.txt"
	generatedPath = "r6/cache/modded"
	redmodDir     = "mods"
)

// Arg is the argument that makes the game load what Deploy compiled.
const Arg = "-modded"

const (
	failedLine  = "Commandlet deploy has failed."
	stagePrefix = "[DEPLOY]"
	// interrupted is STATUS_CONTROL_C_EXIT: the console was closed or interrupted.
	interrupted = 0xC000013A
	// interruptedSigned is the same code as a 32-bit signed exit status reads it.
	interruptedSigned = -1073741510
)

// Installed reports whether the install has the REDmod DLC's tool and launcher, the one answer to "can this install
// load REDmods".
func Installed(installDir string) bool {
	return safefs.IsFile(Tool(installDir)) && safefs.IsFile(filepath.Join(installDir, launcherPath))
}

// Tool is the path of REDmod's command-line tool.
func Tool(installDir string) string { return filepath.Join(installDir, filepath.FromSlash(toolPath)) }

// Args are the arguments of the deploy: the root is a separate argument, every other flag takes its value with "=".
// path turns a host path into the form the tool is given (Wine's Z: form on Linux).
func Args(installDir string, path func(string) string) []string {
	return []string{
		"deploy", "-reportProgress", "-root", path(installDir),
		"-rttiSchemaPath=" + path(filepath.Join(installDir, filepath.FromSlash(metadataPath))),
		"-modlist=" + path(filepath.Join(installDir, filepath.FromSlash(modListPath))),
	}
}

// FailureReason is why a failed deploy failed: the last line the tool printed before it reported the failure.
func FailureReason(exitCode int, output string) string {
	if exitCode == interrupted || exitCode == interruptedSigned {
		return "REDmod was interrupted"
	}
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	failedAt := max(slices.Index(lines, failedLine), 0)
	var reason string
	for _, l := range lines[:failedAt] {
		if l != "" {
			reason = l
		}
	}
	if reason == "" || strings.HasPrefix(reason, stagePrefix) {
		return fmt.Sprintf("REDmod exited with code %d", exitCode)
	}
	return reason
}

// Deployed reports whether an earlier deploy left REDmods behind: its modlist names some. Deploying again with none
// left is what clears them, which Vortex does on every start.
func Deployed(installDir string) bool {
	previous, err := readModList(filepath.Join(installDir, filepath.FromSlash(modListPath)))
	return err == nil && len(previous) > 0
}

// WriteModList writes names, one folder per line with Windows line endings (the only ones the tool reads), and clears
// the files an earlier deploy generated when the same mods now come in another order, which the tool does not notice.
func WriteModList(installDir string, names []string) error {
	file := filepath.Join(installDir, filepath.FromSlash(modListPath))
	previous, readErr := readModList(file)
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		return err
	}
	if err := safefs.WriteFileAtomic(file, []byte(strings.Join(names, "\r\n")), 0o644); err != nil {
		return fmt.Errorf("write the REDmod modlist: %w", err)
	}
	// An unreadable list leaves the deployed order unknown.
	if readErr != nil || reordered(previous, names) {
		dir := filepath.Join(installDir, filepath.FromSlash(generatedPath))
		if err := safefs.RemoveAll(dir); err != nil {
			return fmt.Errorf("clear the deployed REDmod files: %w", err)
		}
		return os.MkdirAll(dir, 0o750)
	}
	return nil
}

func readModList(file string) ([]string, error) {
	b, _, err := safefs.ReadFileIfExists(file)
	if err != nil {
		return nil, err
	}
	var out []string
	for l := range strings.SplitSeq(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

func reordered(previous, next []string) bool {
	a, b := slices.Clone(previous), slices.Clone(next)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b) && !slices.Equal(previous, next)
}

// Runner runs the tool with args to its end: its exit code and what it printed. An error is a tool that could not
// run at all.
type Runner func(ctx context.Context, exe string, args []string) (exitCode int, output string, err error)

// FailedError is a deploy the tool reported as failed.
type FailedError struct{ Reason string }

func (e *FailedError) Error() string { return "REDmod could not deploy the mods: " + e.Reason }

// Deploy writes the modlist for names (the profile's REDmod folders, first winning) and runs the tool on it. path is
// Args's.
func Deploy(ctx context.Context, installDir string, names []string, path func(string) string, run Runner) error {
	if err := WriteModList(installDir, names); err != nil {
		return err
	}
	// The tool compiles into these and reads the mods from the first, which a profile with none has not created.
	for _, dir := range []string{redmodDir, generatedPath} {
		if err := os.MkdirAll(filepath.Join(installDir, filepath.FromSlash(dir)), 0o750); err != nil {
			return err
		}
	}
	code, out, err := run(ctx, Tool(installDir), Args(installDir, path))
	if err != nil {
		return fmt.Errorf("run REDmod: %w", err)
	}
	if code != 0 {
		return &FailedError{Reason: FailureReason(code, out)}
	}
	return nil
}
