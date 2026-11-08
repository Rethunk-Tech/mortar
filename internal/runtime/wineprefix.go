package runtime

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-gamelocator/bottles"
	"github.com/Rethunk-Tech/mortar/internal/components"
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
	return expand(t.Windows, prefixTokens(inst))
}

func prefixTokens(inst Install) map[string]string {
	return userTokens(bottles.UserDir(inst.Prefix), inst.Dir)
}

// userTokens are the Windows known folders of the profile folder user, for a game installed in dir.
func userTokens(user, dir string) map[string]string {
	return map[string]string{
		"home":         user,
		"install":      dir,
		"documents":    filepath.Join(user, "Documents"),
		"appData":      filepath.Join(user, "AppData", "Roaming"),
		"localAppData": filepath.Join(user, "AppData", "Local"),
		"localLow":     filepath.Join(user, "AppData", "LocalLow"),
	}
}

// Command is the argv that runs exe with args inside inst's bottle.
func Command(inst Install, exe string, args ...string) ([]string, error) {
	if !(winePrefix{}).Detect(inst) {
		return nil, errors.New("install does not run in a Bottles bottle")
	}
	return bottles.RunArgv(inst.Prefix, exe, args...), nil
}

// Runner starts argv and waits for it.
type Runner func(ctx context.Context, argv []string) error

// ExecRunner runs argv as a host process, through flatpak-spawn from inside the Flatpak.
func ExecRunner(ctx context.Context, argv []string) error {
	return flatpak.Command(ctx, argv[0], argv[1:]...).Run()
}

// Run runs exe with args in inst's bottle through run, which is how a Windows installer (SMAPI's) is run in it.
func Run(ctx context.Context, run Runner, inst Install, exe string, args ...string) error {
	argv, err := Command(inst, exe, args...)
	if err != nil {
		return err
	}
	return run(ctx, argv)
}
