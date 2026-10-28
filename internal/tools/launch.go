package tools

import (
	"cmp"
	"log"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

func launchTool(t Tool, ctx Context) error {
	dir := expand(t.WorkingDir, ctx)
	dir = cmp.Or(dir, ctx.Game)
	args := expandArgs(t.Arguments, ctx)
	exe := expand(t.Executable, ctx)
	log.Printf("tools: starting %q (%s)", t.Name, exe)
	if err := refuseWindowsProgram(exe); err != nil {
		return err
	}
	name := exe
	if sandbox.InFlatpak() {
		name, args = sandbox.HostArgv(dir, nil, exe, args...)
		dir = ""
	}
	exited, err := launch.Start(dir, name, args...)
	if err != nil {
		return err
	}
	go launch.ReleaseOnExit(exited)
	return nil
}

func absDir(p string) string {
	if p == "" {
		return ""
	}
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
