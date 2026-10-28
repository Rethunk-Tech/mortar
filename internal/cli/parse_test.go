package cli

import (
	"reflect"
	"testing"
)

func TestParseFlagForms(t *testing.T) {
	bools := map[string]func(*cmd) bool{
		"--json": func(c *cmd) bool { return c.json }, "-v": func(c *cmd) bool { return c.verbose },
		"--verbose": func(c *cmd) bool { return c.verbose }, "--all": func(c *cmd) bool { return c.all },
		"--unused": func(c *cmd) bool { return c.unused }, "--missing": func(c *cmd) bool { return c.missing },
		"--by-mod": func(c *cmd) bool { return c.byMod }, "--yes": func(c *cmd) bool { return c.yesFlag },
		"--force": func(c *cmd) bool { return c.force }, "--allow-unscanned": func(c *cmd) bool { return c.allowUnscanned },
		"--mark": func(c *cmd) bool { return c.mark }, "--restore": func(c *cmd) bool { return c.restore },
		"--check": func(c *cmd) bool { return c.check }, "--wait": func(c *cmd) bool { return c.wait },
		"--vanilla": func(c *cmd) bool { return c.vanilla }, "--update": func(c *cmd) bool { return c.updateFlag },
		"--unlink": func(c *cmd) bool { return c.unlinkFlag }, "--undo": func(c *cmd) bool { return c.undoFlag },
		"--changelog": func(c *cmd) bool { return c.changelogFlag }, "--everywhere": func(c *cmd) bool { return c.everywhere },
		"--remove": func(c *cmd) bool { return c.removeFlag }, "--clear": func(c *cmd) bool { return c.clearFlag },
		"--test": func(c *cmd) bool { return c.testFlag }, "--no-configs": func(c *cmd) bool { return c.noConfigs },
		"--no-mods": func(c *cmd) bool { return c.noMods }, "--preview": func(c *cmd) bool { return c.previewFlag },
	}
	for flag, get := range bools {
		var c cmd
		if err := c.parse([]string{flag, "x"}); err != nil || !get(&c) || !reflect.DeepEqual(c.args, []string{"x"}) {
			t.Errorf("%s: err=%v set=%v args=%v", flag, err, get(&c), c.args)
		}
		var d cmd
		if flag == "-v" {
			continue
		}
		if err := d.parse([]string{flag + "=1"}); err == nil || err.Error() != "unknown flag "+flag+"=1" {
			t.Errorf("%s=1: err=%v", flag, err)
		}
	}

	strs := map[string]struct {
		get func(*cmd) string
		msg string
	}{
		"--filter":  {func(c *cmd) string { return c.filter }, "--filter needs a value"},
		"--preset":  {func(c *cmd) string { return c.presetFlag }, "--preset needs a name"},
		"--item":    {func(c *cmd) string { return c.item }, "--item needs a mod"},
		"--reason":  {func(c *cmd) string { return c.reasonFlag }, "--reason needs a value"},
		"--format":  {func(c *cmd) string { return c.format }, "--format needs md or text"},
		"--run":     {func(c *cmd) string { return c.run }, "--run needs a run id"},
		"--profile": {func(c *cmd) string { return c.profileFlag }, "--profile needs a profile name"},
		"--name":    {func(c *cmd) string { return c.nameFlag }, "--name needs a name"},
		"--game":    {func(c *cmd) string { return c.game }, "--game needs a game id"},
		"--loader":  {func(c *cmd) string { return c.loaderFlag }, "--loader needs a loader id"},
		"--install": {func(c *cmd) string { return c.installFlag }, "--install needs an install id"},
		"--dir":     {func(c *cmd) string { return c.dirFlag }, "--dir needs a folder"},
		"--file":    {func(c *cmd) string { return c.fileFlag }, "--file needs a value"},
		"--version": {func(c *cmd) string { return c.versionFlag }, "--version needs a value"},
		"--from":    {func(c *cmd) string { return c.fromFlag }, "--from needs a mod manager (vortex or mo2)"},
		"--source":  {func(c *cmd) string { return c.sourceFlag }, "--source needs a source id"},
		"--code":    {func(c *cmd) string { return c.codeFlag }, "--code needs a value"},
		"--peer":    {func(c *cmd) string { return c.peerFlag }, "--peer needs a value"},
		"--keep":    {func(c *cmd) string { return c.keepFlag }, "--keep needs a value"},
		"--delete":  {func(c *cmd) string { return c.deleteFlag }, "--delete needs a value"},
		"--dismiss": {func(c *cmd) string { return c.dismissFlag }, "--dismiss needs a value"},
	}
	for flag, s := range strs {
		// A leading --version is the version command, so give it a verb first.
		pre := []string{"verb"}
		for _, form := range [][]string{{flag, "v"}, {flag + "=v"}} {
			var c cmd
			if err := c.parse(append(append([]string{}, pre...), form...)); err != nil || s.get(&c) != "v" || !reflect.DeepEqual(c.args, pre) {
				t.Errorf("%v: err=%v got=%q args=%v", form, err, s.get(&c), c.args)
			}
		}
		var c cmd
		if err := c.parse(append(append([]string{}, pre...), flag)); err == nil || err.Error() != s.msg {
			t.Errorf("%s bare at end: err=%v", flag, err)
		}
	}

	var c cmd
	if err := c.parse([]string{"--page", "3", "--move", "a", "--move=b", "--set", "-h", "--version", "x"}); err != nil ||
		c.pageFlag != 3 || !reflect.DeepEqual(c.moveFlag, []string{"a", "b"}) || !c.setFlag || c.setEquals ||
		!reflect.DeepEqual(c.args, []string{"help"}) || c.versionFlag != "x" {
		t.Errorf("mixed: %+v err", c)
	}
	var e cmd
	if err := e.parse([]string{"--set=a=b", "--page=4", "--help", "--version"}); err == nil || err.Error() != "--version needs a value" ||
		!e.setFlag || !e.setEquals || e.setSpec != "a=b" || e.pageFlag != 4 {
		t.Errorf("set/page: %+v err=%v", e, err)
	}
	var f cmd
	if err := f.parse([]string{"--version"}); err != nil || f.args[0] != "version" {
		t.Errorf("leading version: %v %v", f.args, err)
	}
	for _, args := range [][]string{{"--page"}, {"--page=z"}, {"--page", "z"}, {"--move"}, {"--nope"}} {
		var g cmd
		if err := g.parse(args); err == nil {
			t.Errorf("%v: want error", args)
		}
	}
	var h cmd
	if err := h.parse([]string{"__complete", "--json", "-h"}); err != nil || h.json || len(h.args) != 3 {
		t.Errorf("complete: %+v err=%v", h, err)
	}
}
