package game

import (
	"errors"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/dotnetroot"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ProfileServerStart is the program and arguments that start the dedicated server of the install in installDir with
// dataDir as its data folder: the server's own executable on a Windows build, the assembly under dotnet elsewhere.
func ProfileServerStart(spec components.ProfileServerSpec, platform, installDir, dataDir string) (exe string, args []string, err error) {
	if platform == "windows" {
		return filepath.Join(installDir, spec.Exe), []string{spec.DataFlag, dataDir}, nil
	}
	// Inside the Flatpak the server starts on the host (flatpak-spawn), which cannot see the sandbox's SDK extension.
	lookPath := defaultLookPath()
	dotnet, err := dotnetProgram(lookPath)
	if err != nil {
		return "", nil, err
	}
	return dotnet, []string{filepath.Join(installDir, spec.Assembly), spec.DataFlag, dataDir}, nil
}

// dotnetProgram is dotnet from the PATH lookPath searches, else the one Microsoft's install script puts in ~/.dotnet,
// which a desktop session does not have on its PATH.
func dotnetProgram(lookPath func(string) (string, error)) (string, error) {
	if p, err := lookPath("dotnet"); err == nil {
		return p, nil
	}
	if dir := dotnetroot.Home(lookPath); dir != "" {
		return filepath.Join(dir, "dotnet"), nil
	}
	return "", usererr.Wrap(usererr.Invalid, errors.New("the dedicated server needs the .NET runtime, and dotnet was not found"))
}
