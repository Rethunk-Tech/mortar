package game

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// HasProfileServer reports whether a profile of game id can run the game's dedicated server.
func HasProfileServer(id string) bool {
	info, ok := catalogGame(id)
	return ok && info.ProfileServer != nil
}

// ProfileServerStart is the program and arguments that start the dedicated server of the install in installDir with
// dataDir as its data folder: the server's own executable on a Windows build, the assembly under dotnet elsewhere.
func ProfileServerStart(spec components.ProfileServerSpec, platform, installDir, dataDir string) (exe string, args []string, err error) {
	if platform == "windows" {
		return filepath.Join(installDir, spec.Exe), []string{spec.DataFlag, dataDir}, nil
	}
	dotnet, err := dotnetProgram()
	if err != nil {
		return "", nil, err
	}
	return dotnet, []string{filepath.Join(installDir, spec.Assembly), spec.DataFlag, dataDir}, nil
}

// dotnetProgram is dotnet from the PATH, else the one Microsoft's install script puts in ~/.dotnet, which a desktop
// session does not have on its PATH.
func dotnetProgram() (string, error) {
	if p, err := exec.LookPath("dotnet"); err == nil {
		return p, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := filepath.Join(home, ".dotnet", "dotnet"); safefs.IsFile(p) {
			return p, nil
		}
	}
	return "", usererr.Wrap(usererr.Invalid, errors.New("the dedicated server needs the .NET runtime, and dotnet was not found"))
}
