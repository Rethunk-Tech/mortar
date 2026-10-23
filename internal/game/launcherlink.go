package game

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// onlyDLLOverrides is true when env holds nothing a launcher would lose: the DLL overrides are written into the prefix's
// registry before the launch (ensureRuntime), so the variable is not needed.
func onlyDLLOverrides(env map[string]string) bool {
	for k := range env {
		if k != "WINEDLLOVERRIDES" {
			return false
		}
	}
	return true
}

// launcherCommand starts a Heroic or Lutris copy through its launcher, so the game runs in the prefix and with the
// runner the launcher keeps for it. Heroic's link carries the plan's arguments (src/backend/protocol.ts reads each `arg`
// parameter); Lutris's rungame link carries nothing. A plan with other environment variables, a launch prefix or its own
// executable starts directly. ok is
// false when the launcher cannot take the plan.
func launcherCommand(inst Install, plan *launchplan.Plan, lookPath func(string) (string, error)) (cmd launch.Command, ok bool, err error) {
	kind, rest, found := strings.Cut(inst.Launcher, ":")
	if !found || plan.Exe != "" || len(plan.Prefix) > 0 || !onlyDLLOverrides(plan.Env) {
		return launch.Command{}, false, nil
	}
	switch kind {
	case "heroic":
		runner, app, _ := strings.Cut(rest, ":")
		if (runner != "gog" && runner != "legendary") || !components.PlainAppName(app) {
			return launch.Command{}, false, fmt.Errorf("the Heroic game %q is not a plain name", rest)
		}
		q := url.Values{"appName": {app}, "runner": {runner}, "arg": plan.Args}
		link := "heroic://launch?" + q.Encode()
		if opener, err := lookPath("xdg-open"); err == nil {
			return launch.Command{Name: opener, Args: []string{link}, Failure: launch.HintLauncher, Relay: true, Via: "heroic"}, true, nil
		}
		return launch.Command{}, false, nil
	case "lutris":
		if len(plan.Args) > 0 || !components.PlainAppName(rest) {
			return launch.Command{}, false, nil
		}
		if lutris, err := lookPath("lutris"); err == nil {
			return launch.Command{Name: lutris, Args: []string{"lutris:rungame/" + rest}, Failure: launch.HintLauncher, Relay: true, Via: "lutris"}, true, nil
		}
		return launch.Command{}, false, nil
	}
	return launch.Command{}, false, nil
}

// LauncherRelays reports whether inst starts through its own launcher for plan; a launcher copy whose plan the launcher
// cannot take starts directly instead.
func (st Starter) LauncherRelays(inst Install, plan *launchplan.Plan) bool {
	_, ok, err := launcherCommand(inst, plan, st.lookPathOrDefault())
	return ok && err == nil
}
