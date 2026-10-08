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
	"io"
	"io/fs"
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

// logHeader matches SMAPI's version line: the first line of SMAPI-latest.txt, which carries the "[time LEVEL SMAPI] "
// prefix of every log line, or the bare message a parsed log entry holds.
var logHeader = regexp.MustCompile(`^(?:\[[^\]]*\] )?SMAPI (\S+) with Stardew Valley (\S+)`)

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
	b, _ := io.ReadAll(io.LimitReader(f, maxLauncher))
	return bytes.Contains(b, []byte(smapiMarker))
}

// buildOS is the OS of the game build in dir: a Windows build on a Linux host (one in a Bottles bottle) is
// recognised by its executable and is run and installed into the Windows way.
func buildOS(dir string) string {
	if hostOS == "linux" && fsx.IsFile(filepath.Join(dir, "Stardew Valley.exe")) {
		return "windows"
	}
	return hostOS
}

// hostOS is the OS Mortar runs on; tests set it to exercise another host's branch.
var hostOS = runtime.GOOS

// loaderState reports what is installed in dir for goos. On Linux SMAPI renames the game's launcher to
// StardewValley-original and installs its own in its place, which a game update overwrites.
func loaderState(dir, goos string) (installed, broken bool) {
	if goos == "windows" {
		return fsx.IsFile(filepath.Join(dir, smapiMarker+".exe")), false
	}
	original := fsx.IsFile(filepath.Join(dir, linuxOriginal))
	smapiFiles := fsx.IsFile(filepath.Join(dir, smapiMarker+".dll"))
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
	st.Shared, st.LinkedFrom = sharedSMAPI(t.InstallDir)
	if !installed && !broken {
		return st, nil
	}
	logSMAPI, logGame := l.logVersions()
	st.GameVersion = logGame
	st.Version = cmp.Or(logSMAPI, bundledVersion(t.InstallDir))
	return st, nil
}

// sharedSMAPI reports whether SMAPI's main file in dir (the .exe or the .dll) is a link another mod manager deployed,
// and the folder a symlink points into.
func sharedSMAPI(dir string) (bool, string) {
	for _, name := range []string{smapiMarker + ".exe", smapiMarker + ".dll"} {
		path := filepath.Join(dir, name)
		if !fsx.Shared(path) {
			continue
		}
		target, err := fsx.EvalSymlinks(path)
		if err != nil || fsx.SamePath(filepath.Dir(target), dir) {
			return true, ""
		}
		return true, filepath.Dir(target)
	}
	return false, ""
}

// unshareTree replaces a folder that is itself a symlink with a real copy of what it links to.
func unshareTree(tree string) error {
	fi, err := os.Lstat(tree)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	target, err := fsx.EvalSymlinks(tree)
	if err != nil {
		return err
	}
	tmp := tree + ".mortar-unshare"
	if err := datadir.CopyTreeResolvingLinks(target, tmp); err != nil {
		_ = fsx.RemoveAll(tmp)
		return err
	}
	if err := os.Remove(tree); err != nil {
		_ = fsx.RemoveAll(tmp)
		return err
	}
	return fsx.Rename(tmp, tree)
}

// unshareSMAPI gives every file the installer rewrites (the game folder's own files, smapi-internal and SMAPI's
// bundled mods) its own copy first. The installer writes in place, so through a symlink or hard link it would
// change the other mod manager's copy too.
func unshareSMAPI(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			if err := fsx.Unshare(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	trees := []string{filepath.Join(dir, "smapi-internal")}
	for _, m := range bundledMods {
		trees = append(trees, filepath.Join(dir, "Mods", m))
	}
	for _, tree := range trees {
		if err := unshareTree(tree); err != nil {
			return err
		}
		err := filepath.WalkDir(tree, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			return fsx.Unshare(path)
		})
		if err != nil {
			return err
		}
	}
	return nil
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
		if err := datadir.CopyTreeResolvingLinks(filepath.Join(dir, "Mods", name), filepath.Join(dst, name)); err != nil {
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
	if err := unshareSMAPI(dir); err != nil {
		return "", fmt.Errorf("separate SMAPI from another mod manager's files: %w", err)
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
		if out, err := runInstaller(ctx, cmd); err != nil {
			return "", installerError(err, out)
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

// installerError names the failure with the installer's output, or, when it printed nothing (Windows cannot capture
// it), says what to check.
func installerError(err error, out []byte) error {
	if msg := tail(out); msg != "" {
		return fmt.Errorf("SMAPI installer failed (%w): %s", err, msg)
	}
	return fmt.Errorf("SMAPI installer failed (%w). Check that the game folder is complete, then retry", err)
}

// tail keeps the end of installer output, where its error is.
func tail(out []byte) string {
	const keep = 2000
	if len(out) > keep {
		out = out[len(out)-keep:]
	}
	return strings.TrimSpace(string(out))
}
