package devsvc

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
)

// The Flatpak has no .NET SDK of its own. Flathub's SDK extension is mounted into the app when installed (the
// add-extensions block in build/linux/flatpak/tech.rethunk.Mortar.yml); its branch is the freedesktop release the
// manifest's runtime sits on. The extension's bin/dotnet is a symlink to /usr/lib/sdk, which is only mounted in a
// build, so we run lib/dotnet with DOTNET_ROOT pointing at lib.
const (
	flatpakDotnetExtension = "org.freedesktop.Sdk.Extension.dotnet10"
	flatpakDotnetBranch    = "26.08"
	flatpakDotnetMount     = "/app/extensions/dotnet10"
)

// FlatpakDotnetInstall is the command that installs the SDK extension.
const FlatpakDotnetInstall = "flatpak install flathub " + flatpakDotnetExtension + "//" + flatpakDotnetBranch

// extensionDotnetRoot is the extension's DOTNET_ROOT under mount, or "" outside a Flatpak or when it is not installed.
func extensionDotnetRoot(inside bool, mount string) string {
	if !inside {
		return ""
	}
	root := filepath.Join(mount, "lib")
	if st, err := os.Stat(filepath.Join(root, "dotnet")); err != nil || !st.Mode().IsRegular() {
		return ""
	}
	return root
}

func flatpakDotnetRoot() string {
	return extensionDotnetRoot(flatpak.Inside(), flatpakDotnetMount)
}

// dotnetProgram is the dotnet to run: the extension's when there is one, else whatever the PATH holds.
func dotnetProgram(root string) string {
	if root == "" {
		return "dotnet"
	}
	return filepath.Join(root, "dotnet")
}

// withDotnetRoot points env at root, and puts it first on the PATH so the dotnet that MSBuild starts is the same one.
func withDotnetRoot(env []string, root string) []string {
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
