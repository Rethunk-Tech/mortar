package stardew

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

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/loader"
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

func (g Game) logDir() (string, error) {
	if g.LogDir != "" {
		return g.LogDir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "StardewValley", "ErrorLogs"), nil
}

// logVersions returns the SMAPI and game versions from the first line of SMAPI-latest.txt, or empty strings.
func (g Game) logVersions() (smapi, game string) {
	dir, err := g.logDir()
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

// LoaderStatus reports SMAPI's state in dir. recorded is the version Mortar installed, preferred over the log's.
func (g Game) LoaderStatus(dir, recorded string) loader.Status {
	installed, broken := loaderState(dir, runtime.GOOS)
	st := loader.Status{Installed: installed, Broken: broken}
	if !installed && !broken {
		return st
	}
	logSMAPI, logGame := g.logVersions()
	st.GameVersion = logGame
	st.Version = cmp.Or(recorded, logSMAPI, bundledVersion(dir))
	return st
}

// bundledMods are the mods the SMAPI installer places in Mods.
var bundledMods = []string{"ConsoleCommands", "SaveBackup"}

// bundledVersion reads the version from Console Commands' manifest, which the installer sets to SMAPI's own.
func bundledVersion(dir string) string {
	b, err := fsx.ReadFile(filepath.Join(dir, "Mods", bundledMods[0], "manifest.json"))
	if err != nil {
		return ""
	}
	m, err := manifest.Parse(b)
	if err != nil {
		return ""
	}
	return m.Version
}

// CopyBundled copies SMAPI's Console Commands and Save Backup from dir/Mods into dst.
func (Game) CopyBundled(dir, dst string) error {
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

// InstallLoader installs the newest SMAPI into dir, or updates it: the installer is the same either way.
// It returns the installed version. bundled is handed SMAPI's own mods before anything is cleaned up.
func (g Game) InstallLoader(ctx context.Context, dir string, bundled loader.Bundled, progress func(loader.Step)) (string, error) {
	if err := g.ValidInstall(dir); err != nil {
		return "", err
	}
	version, err := g.LatestLoader(ctx)
	if err != nil {
		return "", err
	}
	return g.InstallLoaderAt(ctx, dir, version, bundled, progress)
}

// InstallLoaderAt installs the given SMAPI version into dir.
func (g Game) InstallLoaderAt(ctx context.Context, dir, version string, bundled loader.Bundled, progress func(loader.Step)) (string, error) {
	if err := g.ValidInstall(dir); err != nil {
		return "", err
	}
	instDir, exe, err := installer(version, runtime.GOOS)
	if err != nil {
		return "", err
	}
	work, err := os.MkdirTemp("", "mortar-smapi-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(work) }()

	zipPath := filepath.Join(work, "installer.zip")
	if err := g.download(ctx, version, zipPath); err != nil {
		return "", err
	}
	progress(loader.StepDownloaded)
	unpacked := filepath.Join(work, "installer")
	if err := os.Mkdir(unpacked, 0o700); err != nil {
		return "", err
	}
	if err := archive.Extract(zipPath, unpacked, archive.Options{}); err != nil {
		return "", err
	}

	folder := filepath.Join(unpacked, instDir)
	if err := fsx.Chmod(filepath.Join(folder, exe), 0o700); err != nil {
		return "", fmt.Errorf("SMAPI %s installer is missing %s: %w", version, exe, err)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(folder, exe), "--install", "--no-prompt", "--game-path", dir)
	cmd.Dir = folder
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("SMAPI installer failed (%w): %s", err, tail(out))
	}
	if installed, _ := loaderState(dir, runtime.GOOS); !installed {
		return "", errors.New("the SMAPI installer finished but SMAPI is not in the game folder")
	}
	progress(loader.StepFiles)
	progress(loader.StepLauncher)

	mods := filepath.Join(work, "bundled")
	if err := os.Mkdir(mods, 0o700); err != nil {
		return "", err
	}
	if err := archive.Extract(filepath.Join(folder, "install.dat"), mods, archive.Options{}); err != nil {
		return "", err
	}
	if err := bundled(version, filepath.Join(mods, "Mods")); err != nil {
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
