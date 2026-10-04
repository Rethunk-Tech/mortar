package cli

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func (c *cmd) gameCmd() error {
	if len(c.args) < 2 {
		return usageError{"game needs steam-launch-option or launch-preset-templates"}
	}
	switch c.args[1] {
	case "steam-launch-option":
		return c.gameSteamLaunchOption()
	case "launch-preset-templates":
		return c.gameLaunchPresetTemplates()
	default:
		return usageError{"unknown game command " + c.args[1]}
	}
}

func (c *cmd) gameLaunchPresetTemplates() error {
	if len(c.args) < 3 {
		return usageError{"game launch-preset-templates needs a game"}
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
			return usageError{"game launch-preset-templates add needs a name"}
		}
	case "use":
		if name == "" || options == "" {
			return usageError{"game launch-preset-templates use needs a template name and a profile"}
		}
	case "remove":
		if name == "" {
			return usageError{"game launch-preset-templates remove needs a name"}
		}
	default:
		return usageError{"game launch-preset-templates is list, add, use, or remove"}
	}
	var presets []settings.LaunchPresetTemplate
	err := c.call("game.launchPresetTemplates", control.Params{
		Game: game, Sub: sub, Name: name, Value: options, Path: prefix, Query: env, Profile: options,
	}, &presets, readTimeout)
	if err != nil {
		return err
	}
	return c.emit(presets, func() { printLaunchPresetTemplates(c, presets) })
}

func printLaunchPresetTemplates(c *cmd, presets []settings.LaunchPresetTemplate) {
	if len(presets) == 0 {
		fmt.Fprintln(c.out, "No launch presets.")
		return
	}
	for _, preset := range presets {
		console := "follow"
		if preset.ShowConsole != "" {
			console = "console=" + preset.ShowConsole
		}
		fmt.Fprintf(c.out, "%s\t%s\t%s\t%s\t%s\n", preset.Name, preset.Options, preset.Prefix, preset.Env, console)
	}
}
