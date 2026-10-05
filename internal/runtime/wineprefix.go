package runtime

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const storeBottles = "bottles"

type winePrefix struct{}

func (winePrefix) ID() string { return WinePrefix }

// Detect: a Windows build in a Bottles bottle on Linux runs through bottles-cli.
func (winePrefix) Detect(inst Install) bool {
	return inst.host() == "linux" && inst.Platform == "windows" && inst.Store == storeBottles && inst.Prefix != ""
}

func (w winePrefix) Resolve(inst Install, t components.PathTemplate) (string, error) {
	if !w.Detect(inst) {
		return "", errors.New("install does not run in a Bottles bottle")
	}
	if t.Windows == "" {
		return "", errors.New("no path for a windows build")
	}
	user := prefixUser(inst.Prefix)
	return expand(t.Windows, map[string]string{
		"home":         user,
		"install":      inst.Dir,
		"documents":    filepath.Join(user, "Documents"),
		"appData":      filepath.Join(user, "AppData", "Roaming"),
		"localAppData": filepath.Join(user, "AppData", "Local"),
		"localLow":     filepath.Join(user, "AppData", "LocalLow"),
	})
}

// prefixUser is the Windows profile folder inside a bottle: its one real user, else Wine's default steamuser.
func prefixUser(bottle string) string {
	users := filepath.Join(bottle, "drive_c", "users")
	if entries, err := os.ReadDir(users); err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != "Public" {
				return filepath.Join(users, e.Name())
			}
		}
	}
	return filepath.Join(users, "steamuser")
}

var reBottleName = regexp.MustCompile(`(?m)^Name:\s*(.+?)\s*$`)

// bottleName is the name bottles-cli knows the bottle by: bottle.yml's Name, else its folder's name.
func bottleName(bottle string) string {
	if b, err := fsx.ReadFile(filepath.Join(bottle, "bottle.yml")); err == nil {
		if m := reBottleName.FindSubmatch(b); m != nil {
			if n := strings.Trim(string(m[1]), `"'`); n != "" {
				return n
			}
		}
	}
	return filepath.Base(bottle)
}

// flatpakBottles reports whether the bottle belongs to Flatpak Bottles, whose CLI is not on the host's PATH.
func flatpakBottles(bottle string) bool {
	return strings.Contains(filepath.ToSlash(bottle), "/.var/app/com.usebottles.bottles/")
}

// Command is the argv that runs exe with args inside inst's bottle.
func Command(inst Install, exe string, args ...string) ([]string, error) {
	if !(winePrefix{}).Detect(inst) {
		return nil, errors.New("install does not run in a Bottles bottle")
	}
	argv := []string{"bottles-cli"}
	if flatpakBottles(inst.Prefix) {
		argv = []string{"flatpak", "run", "--command=bottles-cli", "com.usebottles.bottles"}
	}
	argv = append(argv, "run", "-b", bottleName(inst.Prefix), "-e", exe)
	if len(args) > 0 {
		argv = append(argv, "--")
		argv = append(argv, args...)
	}
	return argv, nil
}

// Runner starts argv and waits for it.
type Runner func(ctx context.Context, argv []string) error

// ExecRunner runs argv as a host process.
func ExecRunner(ctx context.Context, argv []string) error {
	return exec.CommandContext(ctx, argv[0], argv[1:]...).Run() // #nosec G204 -- argv is built by Command: bottles-cli or flatpak with fixed leading arguments
}

// Run runs exe with args in inst's bottle through run, which is how a Windows installer (SMAPI's) is run in it.
func Run(ctx context.Context, run Runner, inst Install, exe string, args ...string) error {
	argv, err := Command(inst, exe, args...)
	if err != nil {
		return err
	}
	return run(ctx, argv)
}
