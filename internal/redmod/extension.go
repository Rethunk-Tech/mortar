package redmod

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/folder"
)

// CodeMissing is the finding's code: REDmods are in the profile and the game has no REDmod tool to deploy them.
const CodeMissing = "redmodMissing"

var _ = folder.Extend(extension{})

// extension is the folder loader's extension for a game whose mods target is REDmod's. The deploy itself runs once
// the files are placed (launchsvc), so it only reports what it will skip.
type extension struct{}

func (extension) Applies(info components.GameInfo) bool {
	t, ok := info.Target(folder.TargetID)
	return ok && t.REDmod
}

func (extension) Role() string { return "" }

func (extension) Contribute(*launchplan.Plan, loader.ProfileView) {}

func (extension) Prelaunch(loader.ProfileView) error { return nil }

// Findings reports REDmods in the profile that the install cannot deploy.
func (extension) Findings(p loader.ProfileView) []loader.Finding {
	if p.InstallDir == "" || Installed(p.InstallDir) {
		return nil
	}
	root, err := folder.ProfileModsDir(p.Game, p.Dir)
	if err != nil {
		return nil
	}
	infos, _ := filepath.Glob(filepath.Join(root, "mods", "*", "info.json"))
	if len(infos) == 0 {
		return nil
	}
	return []loader.Finding{{
		Kind: "load-exception", Plugin: "REDmod", Code: CodeMissing,
		Message: "REDmod mods are installed, but the REDmod DLC is not: the game starts without them",
	}}
}
