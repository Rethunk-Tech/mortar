// Package smapi is the SMAPI mod loader of Stardew Valley: install, status, the launch arguments that point it at a
// profile, and the console and log it offers.
package smapi

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	gamert "github.com/Rethunk-Tech/mortar/internal/runtime"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

const (
	linuxOriginal = "StardewValley-original"
	linuxLauncher = "StardewValley"
	smapiMarker   = "StardewModdingAPI"
	// maxLauncher bounds the launcher read: SMAPI's is a shell script and the game's is a small native stub.
	maxLauncher = 16 << 20
)

// logHeader matches the first line of SMAPI-latest.txt.
var logHeader = regexp.MustCompile(`^SMAPI (\S+) with Stardew Valley (\S+)`)

func (l Loader) logDir() (string, error) {
	if l.LogDir != "" {
		return l.LogDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return gamert.Resolve(gamert.Install{Platform: runtime.GOOS, Home: home}, gameInfo().Paths["errorLogs"])
}

// logVersions returns the SMAPI and game versions from the first line of SMAPI-latest.txt, or empty strings.
func (l Loader) logVersions() (smapi, game string) {
	dir, err := l.logDir()
	if err != nil {
		return "", ""
	}
	f, err := fsx.Open(filepath.Join(dir, "SMAPI-latest.txt"))
	if err != nil {
		return "", ""
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return "", ""
	}
	if m := logHeader.FindStringSubmatch(sc.Text()); m != nil {
		return m[1], m[2]
	}
	return "", ""
}

func launcherHasSMAPI(dir string) bool {
	f, err := fsx.Open(filepath.Join(dir, linuxLauncher))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	b := make([]byte, maxLauncher)
	n, _ := f.Read(b)
	return bytes.Contains(b[:n], []byte(smapiMarker))
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

// buildOS is the OS of the game build in dir: a Windows build on a Linux host (one in a Bottles bottle) is
// recognised by its executable and is run and installed into the Windows way.
func buildOS(dir string) string {
	if runtime.GOOS == "linux" && isFile(filepath.Join(dir, "Stardew Valley.exe")) {
		return "windows"
	}
	return runtime.GOOS
}

// loaderState reports what is installed in dir for goos. On Linux SMAPI renames the game's launcher to
// StardewValley-original and installs its own in its place, which a game update overwrites.
func loaderState(dir, goos string) (installed, broken bool) {
	if goos == "windows" {
		return isFile(filepath.Join(dir, smapiMarker+".exe")), false
	}
	original := isFile(filepath.Join(dir, linuxOriginal))
	smapiFiles := isFile(filepath.Join(dir, smapiMarker+".dll"))
	if !original && !smapiFiles {
		return false, false
	}
	if launcherHasSMAPI(dir) && original {
		return true, false
	}
	return false, true
}

// Status reports SMAPI's state in the install. The version is the log's, else the bundled mods'; a version Mortar
// recorded when it installed SMAPI takes precedence over both and is the caller's to apply.
func (l Loader) Status(t loader.Target) (loader.Status, error) {
	installed, broken := loaderState(t.InstallDir, buildOS(t.InstallDir))
	st := loader.Status{Installed: installed, Broken: broken}
	if !installed && !broken {
		return st, nil
	}
	logSMAPI, logGame := l.logVersions()
	st.GameVersion = logGame
	st.Version = cmp.Or(logSMAPI, bundledVersion(t.InstallDir))
	return st, nil
}

// bundledMods are the mods the SMAPI installer places in Mods.
var bundledMods = []string{"ConsoleCommands", "SaveBackup"}

// bundledVersion reads the version from Console Commands' manifest, which the installer sets to SMAPI's own.
func bundledVersion(dir string) string {
	b, err := fsx.ReadFile(filepath.Join(dir, "Mods", bundledMods[0], manifest.FileName))
	if err != nil {
		return ""
	}
	m, err := manifest.Parse(b)
	if err != nil {
		return ""
	}
	return m.Version
}

// BundleSource is how a profile lists SMAPI's bundled mods.
func (Loader) BundleSource() (kind, name string) { return "smapi", "SMAPI" }

// CopyBundled copies SMAPI's Console Commands and Save Backup from dir/Mods into dst, for SMAPI installed outside Mortar.
func (Loader) CopyBundled(dir, dst string) error {
	for _, name := range bundledMods {
		if err := datadir.CopyTree(filepath.Join(dir, "Mods", name), filepath.Join(dst, name)); err != nil {
			return fmt.Errorf("copy %s: %w", name, err)
		}
	}
	return nil
}

// installer returns the folder and file name of the platform installer inside the extracted zip.
func installer(version, goos string) (dir, exe string, err error) {
	root := filepath.Join("SMAPI "+version+" installer", "internal")
	switch goos {
	case "linux":
		return filepath.Join(root, "linux"), "SMAPI.Installer", nil
	case "windows":
		return filepath.Join(root, "windows"), "SMAPI.Installer.exe", nil
	}
	return "", "", fmt.Errorf("installing SMAPI is not supported on %s", goos)
}

// Install runs the installer in pkg's archive, which is the same for a first install and an update, and hands t.Bundled
// SMAPI's own mods before anything is cleaned up. It returns the installed version.
func (Loader) Install(ctx context.Context, t loader.Target, pkg loader.Package, progress func(loader.Step)) (string, error) {
	dir, version := t.InstallDir, pkg.Version
	goos := buildOS(dir)
	instDir, exe, err := installer(version, goos)
	if err != nil {
		return "", err
	}
	work, err := os.MkdirTemp("", "mortar-smapi-")
	if err != nil {
		return "", err
	}
	defer func() { _ = fsx.RemoveAll(work) }()

	unpacked := filepath.Join(work, "installer")
	if err := os.Mkdir(unpacked, 0o700); err != nil {
		return "", err
	}
	if err := archive.Extract(pkg.Archive, unpacked); err != nil {
		return "", err
	}

	folder := filepath.Join(unpacked, instDir)
	if err := fsx.Chmod(filepath.Join(folder, exe), 0o700); err != nil {
		return "", fmt.Errorf("SMAPI %s installer is missing %s: %w", version, exe, err)
	}
	args := []string{"--install", "--no-prompt", "--game-path", dir}
	if goos != runtime.GOOS {
		// The Windows installer runs inside the install's own runtime, which the host can only reach through Exec.
		if t.Exec == nil {
			return "", fmt.Errorf("a %s build of the game needs a runtime to run the SMAPI installer", goos)
		}
		if err := t.Exec(ctx, launchplan.RuntimeReq{}, append([]string{filepath.Join(folder, exe)}, args...)); err != nil {
			return "", fmt.Errorf("SMAPI installer failed: %w", err)
		}
	} else {
		cmd := exec.CommandContext(ctx, filepath.Join(folder, exe), args...)
		cmd.Dir = folder
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("SMAPI installer failed (%w): %s", err, tail(out))
		}
	}
	if installed, _ := loaderState(dir, goos); !installed {
		return "", errors.New("the SMAPI installer finished but SMAPI is not in the game folder")
	}
	progress(loader.StepFiles)
	progress(loader.StepLauncher)

	mods := filepath.Join(work, "bundled")
	if err := os.Mkdir(mods, 0o700); err != nil {
		return "", err
	}
	if err := archive.Extract(filepath.Join(folder, "install.dat"), mods); err != nil {
		return "", err
	}
	if err := t.Bundled(version, filepath.Join(mods, "Mods")); err != nil {
		return "", err
	}
	progress(loader.StepBundled)
	return version, nil
}

// tail keeps the end of installer output, where its error is.
func tail(out []byte) string {
	const keep = 2000
	if len(out) > keep {
		out = out[len(out)-keep:]
	}
	return strings.TrimSpace(string(out))
}
