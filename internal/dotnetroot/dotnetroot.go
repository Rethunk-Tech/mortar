// Package dotnetroot finds the dotnet that Mortar runs inside the Flatpak, which has no .NET of its own.
package dotnetroot

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-safefs"
)

// Flathub's SDK extension is mounted into the app when installed (the add-extensions block in
// build/linux/flatpak/tech.rethunk.Mortar.yml); its branch is the freedesktop release the manifest's runtime sits on.
// The extension's bin/dotnet is a symlink to /usr/lib/sdk, which is only mounted in a build, so we run lib/dotnet with
// DOTNET_ROOT pointing at lib.
const (
	Extension = "org.freedesktop.Sdk.Extension.dotnet10"
	Branch    = "26.08"
	Mount     = "/app/extensions/dotnet10"
)

// InstallCommand is the command that installs the SDK extension.
const InstallCommand = "flatpak install flathub " + Extension + "//" + Branch

// Root is the extension's DOTNET_ROOT under mount, or "" outside a Flatpak or when it is not installed.
func Root(inside bool, mount string) string {
	if !inside {
		return ""
	}
	root := filepath.Join(mount, "lib")
	if !safefs.IsFile(filepath.Join(root, "dotnet")) {
		return ""
	}
	return root
}

// FlatpakRoot is Root for this process.
func FlatpakRoot() string {
	return Root(flatpak.Inside(), Mount)
}

// Home is the folder Microsoft's install script puts dotnet in (~/.dotnet), or "" when its dotnet is not there. A desktop
// session does not have it on its PATH. lookPath decides whether the file is executable, so inside the Flatpak it is
// the host's lookup: the sandbox cannot see the host's home.
func Home(lookPath func(string) (string, error)) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".dotnet")
	if _, err := lookPath(filepath.Join(dir, "dotnet")); err != nil {
		return ""
	}
	return dir
}

// Program is the dotnet to run: the extension's when there is one, else whatever the PATH holds.
func Program(root string) string {
	if root == "" {
		return "dotnet"
	}
	return filepath.Join(root, "dotnet")
}

// WithRoot points env at root, and puts it first on the PATH so the dotnet that MSBuild starts is the same one.
func WithRoot(env []string, root string) []string {
	if root == "" {
		return env
	}
	path := root
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "DOTNET_ROOT="):
		case strings.HasPrefix(kv, "PATH="):
			path = root + string(os.PathListSeparator) + strings.TrimPrefix(kv, "PATH=")
		default:
			out = append(out, kv)
		}
	}
	return append(out, "DOTNET_ROOT="+root, "PATH="+path)
}
