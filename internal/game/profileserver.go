package game

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
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
	dotnet, err := dotnetProgram(dotnetroot.FlatpakRoot())
	if err != nil {
		return "", nil, err
	}
	return dotnet, []string{filepath.Join(installDir, spec.Assembly), spec.DataFlag, dataDir}, nil
}

// dotnetProgram is the Flatpak's SDK extension dotnet when Mortar runs there with it, else dotnet from the PATH, else the one Microsoft's install script puts in ~/.dotnet, which a desktop
// session does not have on its PATH.
func dotnetProgram(extensionRoot string) (string, error) {
	if extensionRoot != "" {
		return dotnetroot.Program(extensionRoot), nil
	}
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
