// Package doctor is the environment checks `mortar doctor` and Settings › About run.
package doctor

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/problems"
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
	for _, g := range in.Games {
		e := env[g.ID]
		st := Pass
		if !g.Installed {
			st = Warn
		}
		checks = append(checks, Check{
			ID:     "game:" + g.ID,
			Status: st,
			Detail: fmt.Sprintf("%s: installed %s, folder %q, store %s, game %s, SMAPI %s, %s",
				g.Name, yes(g.Installed), g.InstallDir, g.Store, e.GameVersion, e.APIVersion, e.Platform),
		})
	}
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

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
