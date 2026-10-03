package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

func (c *cmd) gameCmd() error {
	if len(c.args) < 2 {
		return usageError{"game needs steam-launch-option or launch-presets"}
	}
	switch c.args[1] {
	case "steam-launch-option":
		return c.gameSteamLaunchOption()
	case "launch-presets":
		return c.gameLaunchPresets()
	default:
		return usageError{"unknown game command " + c.args[1]}
	}
}

func (c *cmd) gameLaunchPresets() error {
	if len(c.args) < 3 {
		return usageError{"game launch-presets needs a game"}
	}
	game := c.args[2]
	sub := ""
	name := ""
	if len(c.args) > 3 {
		sub = c.args[3]
	}
	if len(c.args) > 4 {
		name = c.args[4]
	}
	var options, prefix, env string
	if len(c.args) > 5 {
		options = c.args[5]
	}
	if len(c.args) > 6 {
		prefix = c.args[6]
	}
	if len(c.args) > 7 {
		env = strings.Join(c.args[7:], " ")
	}
	switch sub {
	case "", "list":
		sub = "list"
	case "add":
		if name == "" {
			return usageError{"game launch-presets add needs a name"}
		}
	case "remove":
		if name == "" {
			return usageError{"game launch-presets remove needs a name"}
		}
	default:
		return usageError{"game launch-presets is list, add, or remove"}
	}
	var presets []settings.LaunchPreset
	err := c.ask("game.launchPresets", control.Params{
		Game: game, Sub: sub, Name: name, Value: options, Path: prefix, Query: env,
	}, &presets, readTimeout)
	if err != nil {
		return err
	}
	return c.emit(presets, func() { printLaunchPresets(c, presets) })
}

func printLaunchPresets(c *cmd, presets []settings.LaunchPreset) {
	if len(presets) == 0 {
		fmt.Fprintln(c.out, "No launch presets.")
		return
	}
	for _, preset := range presets {
		fmt.Fprintf(c.out, "%s\t%s\t%s\t%s\n", preset.Name, preset.Options, preset.Prefix, preset.Env)
	}
}
