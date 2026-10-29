package game

import (
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/gamestore"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

// epicCommand starts an Epic Games copy through its launcher, as r2modman (src/r2mm/launching/runners/multiplatform/EgsGameRunner.ts) and Gale
// (src-tauri/src/profile/launch/platform.rs) do, so the launcher signs the game in
// to Epic's online services; the game still loads its loader from its own folder. The link is the launcher's fixed
// scheme around an AppName checked as a plain name, so the system handler is never handed anything else.
func epicCommand(inst Install) (launch.Command, error) {
	info, _ := catalogGame(inst.Game)
	if info.Stores.Epic == nil {
		return launch.Command{}, fmt.Errorf("%s is not sold on Epic Games", inst.Game)
	}
	app := gamestore.EpicAppName(inst.Dir, info.Stores.Epic.AppNames)
	if !components.PlainAppName(app) {
		return launch.Command{}, fmt.Errorf("the Epic AppName %q is not a plain name", app)
	}
	link := "com.epicgames.launcher://apps/" + app + "?action=launch&silent=true"
	return launch.Command{Name: "rundll32.exe", Args: []string{"url.dll,FileProtocolHandler", link}, Failure: launch.HintEpic, Relay: true, Via: "epic"}, nil
}
