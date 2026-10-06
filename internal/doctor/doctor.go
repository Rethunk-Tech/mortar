// Package doctor is the environment checks `mortar doctor` and Settings › About run.
package doctor

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
)

// Pass, Warn and Fail are a check's result.
const (
	Pass = "pass"
	Warn = "warn"
	Fail = "fail"
)

// Check is one environment finding.
type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

// Report is the checks CLI prints and the window shows.
type Report struct {
	Checks []Check `json:"checks"`
}

// Live is the running app's environment, the same fields `mortar doctor` asks for.
type Live struct {
	Version        string
	CommandVersion string
	DataDir        string
	Games          []game.GameInfo
	Environment    map[string]problems.Environment
	NxmHandled     bool
	NxmPrevious    string
}

// FromLive turns the running app's environment into the checks the CLI prints.
func FromLive(in Live) Report {
	env := in.Environment
	if env == nil {
		env = map[string]problems.Environment{}
	}
	checks := []Check{{
		ID:     "mortar",
		Status: Pass,
		Detail: fmt.Sprintf("Mortar %s (this command %s)", in.Version, in.CommandVersion),
	}, {
		ID:     "dataDir",
		Status: Pass,
		Detail: "Data folder: " + in.DataDir,
	}}
	if in.DataDir == "" {
		checks[1].Status = Fail
	}
	anyInstalled := false
	for _, g := range in.Games {
		anyInstalled = anyInstalled || g.Installed
	}
	for _, g := range in.Games {
		checks = append(checks, gameCheck(g, env[g.ID], anyInstalled))
		checks = append(checks, protonChecks(g)...)
	}
	checks = append(checks, deckChecks()...)
	checks = append(checks, flatpakLibraryChecks()...)
	handler := "off"
	nxmStatus := Warn
	if in.NxmHandled {
		handler = "Mortar"
		nxmStatus = Pass
		if in.NxmPrevious != "" {
			handler += " (other games go to " + in.NxmPrevious + ")"
		}
	}
	checks = append(checks, Check{
		ID:     "nxm",
		Status: nxmStatus,
		Detail: "nxm:// links: " + handler,
	})
	return Report{Checks: checks}
}

// PlainText is the human report `mortar doctor` prints (one detail line per check).
func PlainText(r Report) string {
	var b strings.Builder
	for _, c := range r.Checks {
		b.WriteString(c.Detail)
		b.WriteByte('\n')
	}
	return b.String()
}

// A game that is not installed only warns when no game is: Mortar supports several and most people own one.
func gameCheck(g game.GameInfo, e problems.Environment, anyInstalled bool) Check {
	c := Check{ID: "game:" + g.ID, Status: Pass}
	if !g.Installed {
		c.Detail = g.Name + ": not installed"
		if !anyInstalled {
			c.Status = Warn
		}
		return c
	}
	parts := []string{g.Name}
	if e.GameVersion != "" {
		parts = append(parts, e.GameVersion)
	}
	if e.APIVersion != "" {
		parts = append(parts, "with SMAPI "+e.APIVersion)
	}
	where := []string{}
	for _, v := range []string{g.Store, runtimeOf(g, e.Platform)} {
		if v != "" {
			where = append(where, v)
		}
	}
	c.Detail = strings.Join(parts, " ") + fmt.Sprintf(" in %q", g.InstallDir)
	if len(where) > 0 {
		c.Detail += " (" + strings.Join(where, ", ") + ")"
	}
	return c
}

// runtimeOf names what the selected install runs on: Proton and its version, or Wine, for a Windows build on this
// host; else host, the host's platform.
func runtimeOf(g game.GameInfo, host string) string {
	for _, in := range g.Installs {
		if in.Dir != g.InstallDir {
			continue
		}
		switch in.Runtime {
		case runtime.Proton:
			v := in.RuntimeVersion
			switch {
			case v == "":
				return "Proton"
			case strings.Contains(strings.ToLower(v), "proton"):
				return v
			}
			return "Proton " + v
		case runtime.WinePrefix:
			return "Wine"
		}
	}
	return host
}
