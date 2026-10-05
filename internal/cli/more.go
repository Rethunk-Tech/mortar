package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// moreVerb is one verb that maps straight onto a control method: positional arguments in, the method's JSON out.
type moreVerb struct {
	method  string
	args    []string
	params  func(a []string, c *cmd) (control.Params, error)
	msg     string
	timeout time.Duration
	// method2 replaces method when --preview is given.
	method2 string
}

func rest(a []string, from int) []string {
	if len(a) <= from {
		return nil
	}
	return a[from:]
}

func readBody(path string) ([]byte, error) { return fsx.ReadFile(path) }

var moreVerbs = map[string]moreVerb{
	"bundles create": {
		method: "bundles.create", args: []string{"a game", "a bundle name", "a profile"}, msg: "Created the bundle.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], Profile: a[2], IDs: rest(a, 3)}, nil
		},
	},
	"bundles delete": {
		method: "bundles.delete", args: []string{"a game", "a bundle"}, msg: "Deleted the bundle.",
		params: func(a []string, _ *cmd) (control.Params, error) { return control.Params{Game: a[0], Name: a[1]}, nil },
	},
	"bundles rename": {
		method: "bundles.rename", args: []string{"a game", "a bundle", "a new name"}, msg: "Renamed the bundle.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], Value: strings.Join(a[2:], " ")}, nil
		},
	},
	"bundles add": {
		method: "bundles.add", args: []string{"a game", "a bundle", "a profile", "a mod id"}, msg: "Added the mods to the bundle.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], Profile: a[2], IDs: a[3:]}, nil
		},
	},
	"bundles remove": {
		method: "bundles.remove", args: []string{"a game", "a bundle", "a mod id"}, msg: "Removed the mods from the bundle.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], IDs: a[2:]}, nil
		},
	},
	"templates apply": {
		method: "templates.apply", method2: "templates.preview", args: []string{"a game", "a template", "a profile"}, timeout: installTimeout,
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], Profile: a[2]}, nil
		},
	},
	"templates undo": {
		method: "templates.undo", args: []string{"a game", "a profile", "an undo file (the undo object of templates apply --json)"}, msg: "Undid the template.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			b, err := readBody(a[2])
			return control.Params{Game: a[0], Profile: a[1], Body: b}, err
		},
	},
	"templates rename": {
		method: "templates.rename", args: []string{"a game", "a template", "a new name"}, msg: "Renamed the template.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], Value: strings.Join(a[2:], " ")}, nil
		},
	},
	"templates restore": {
		method: "templates.restore", args: []string{"a game", "a template file (one entry of templates list --json)"}, msg: "Restored the template.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			b, err := readBody(a[1])
			return control.Params{Game: a[0], Body: b}, err
		},
	},
	"tools add": {
		method: "tools.add", args: []string{"a game", "a name", "an executable"}, msg: "Added the tool.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Name: a[1], Path: absPath(a[2]), IDs: rest(a, 3)}, nil
		},
	},
	"tools update": {
		method: "tools.update", args: []string{"a game", "a tool id", "a name", "an executable"}, msg: "Updated the tool.",
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Key: a[1], Name: a[2], Path: absPath(a[3]), IDs: rest(a, 4)}, nil
		},
	},
	"tools remove": {
		method: "tools.remove", args: []string{"a game", "a tool id"}, msg: "Removed the tool.",
		params: func(a []string, _ *cmd) (control.Params, error) { return control.Params{Game: a[0], Key: a[1]}, nil },
	},
	"support diagnostics save": {
		method: "support.diagnostics", args: []string{"a .zip path"}, timeout: installTimeout,
		params: func(a []string, c *cmd) (control.Params, error) {
			return control.Params{Game: c.game, Profile: c.profileFlag, Path: absPath(a[0])}, nil
		},
	},
	"lan peers": {method: "lan.peers"},
	"lan send": {
		method: "lan.send", args: []string{"a game", "a profile", "a peer id"}, msg: "Sent the profile.", timeout: installTimeout,
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Profile: a[1], Name: a[2]}, nil
		},
	},
	"lan inbox":   {method: "lan.inbox"},
	"lan accept":  {method: "lan.accept", args: []string{"a transfer id"}, msg: "Accepted.", timeout: installTimeout, params: nameParam},
	"lan decline": {method: "lan.decline", args: []string{"a transfer id"}, msg: "Declined.", params: nameParam},
	"data move": {
		method: "data.move", args: []string{"a destination folder"}, msg: "Moved the data folder.", timeout: installTimeout,
		params: func(a []string, c *cmd) (control.Params, error) {
			return control.Params{Path: absPath(a[0]), Preview: c.previewFlag}, nil
		},
	},
	"data cleanup": {
		method: "data.cleanup", timeout: installTimeout,
		params: func(_ []string, c *cmd) (control.Params, error) { return control.Params{Preview: c.previewFlag}, nil },
	},
	"app update check":   {method: "app.update.check", timeout: installTimeout},
	"app update install": {method: "app.update.install", msg: "Installed the update; restart Mortar to use it.", timeout: installTimeout},
	"links register":     {method: "links.register", msg: "Registered Mortar for its own links."},
	"links enable":       {method: "links.enable", msg: "Mortar now handles that source's links.", params: linkParams},
	"links disable":      {method: "links.disable", msg: "That source's links went back to their previous handler.", params: linkParams},
	"game reset-install": {
		method: "game.resetInstall", args: []string{"a game"}, msg: "Reset the install.", timeout: installTimeout,
		params: func(a []string, c *cmd) (control.Params, error) {
			if !c.yesFlag {
				return control.Params{}, refusedError{"game reset-install deletes the game's install folder; pass --yes"}
			}
			return control.Params{Game: a[0]}, nil
		},
	},
	"game steam-status": {method: "game.steamStatus"},
	"problems check-updates": {
		method: "problems.checkUpdates", args: []string{"a game", "a profile"}, timeout: installTimeout,
		params: func(a []string, _ *cmd) (control.Params, error) {
			return control.Params{Game: a[0], Profile: a[1]}, nil
		},
	},
}

func nameParam(a []string, _ *cmd) (control.Params, error) { return control.Params{Name: a[0]}, nil }

// linkParams: links are toggled per source that has a link scheme.
func linkParams(_ []string, c *cmd) (control.Params, error) {
	if c.sourceFlag != "nexus" && c.sourceFlag != "thunderstore" {
		return control.Params{}, usageError{"links " + c.args[1] + " needs --source nexus or thunderstore"}
	}
	return control.Params{Source: c.sourceFlag}, nil
}

// more runs the verb when it is one of moreVerbs; handled is false otherwise.
func (c *cmd) more() (handled bool, err error) {
	var key string
	var spec moreVerb
	for n := min(3, len(c.args)); n >= 2; n-- {
		k := strings.Join(c.args[:n], " ")
		if v, ok := moreVerbs[k]; ok {
			key, spec = k, v
			break
		}
	}
	if key == "" {
		return false, nil
	}
	skip := len(strings.Fields(key))
	a, err := c.need(skip, spec.args...)
	if err != nil {
		return true, err
	}
	var p control.Params
	if spec.params != nil {
		if p, err = spec.params(a, c); err != nil {
			return true, err
		}
	}
	method := spec.method
	if c.previewFlag && spec.method2 != "" {
		method = spec.method2
	}
	timeout := spec.timeout
	if timeout == 0 {
		timeout = readTimeout
	}
	var raw json.RawMessage
	if err := c.call(method, p, &raw, timeout); err != nil {
		return true, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return true, c.emit(map[string]bool{"ok": true}, func() { fmt.Fprintln(c.out, spec.msg) })
	}
	return true, c.emit(raw, func() {
		var pretty []byte
		if pretty, err = json.MarshalIndent(raw, "", "  "); err == nil {
			fmt.Fprintln(c.out, string(pretty))
		}
	})
}
