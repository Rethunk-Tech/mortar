package redmod

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/installer"
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
	return ok && t.Layout == components.LayoutCyberpunk
}

func (extension) Role() string { return "" }

func (extension) Contribute(*launchplan.Plan, loader.ProfileView) {}

func (extension) Prelaunch(loader.ProfileView) error { return nil }

// Findings reports REDmods in the profile that the install cannot deploy, and what the Cyberpunk installer noted when
// it placed each mod (a layout it could not recognise, files it moved aside, an archive worth a second look).
func (extension) Findings(p loader.ProfileView) []loader.Finding {
	root, err := folder.ProfileModsDir(p.Game, p.Dir)
	if err != nil {
		return nil
	}
	var out []loader.Finding
	for _, n := range installer.ReadNotes(p.Dir) {
		if !datadir.Exists(filepath.Join(root, n.Folder)) {
			continue
		}
		out = append(out, loader.Finding{
			Kind: "load-exception", Plugin: n.Folder, Code: n.Code, Message: n.Message,
			Params: map[string]string{"name": n.Folder, "count": strconv.Itoa(len(n.Files)), "files": strings.Join(firstN(n.Files, 5), ", ")},
		})
	}
	if p.InstallDir == "" || Installed(p.InstallDir) {
		return out
	}
	infos, _ := filepath.Glob(filepath.Join(root, "*", "mods", "*", "info.json"))
	if len(infos) == 0 {
		return out
	}
	return append(out, loader.Finding{
		Kind: "load-exception", Plugin: "REDmod", Code: CodeMissing,
		Message: "REDmod mods are installed, but the REDmod DLC is not: the game starts without them",
	})
}

func firstN(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
