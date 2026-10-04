package tools

import (
	"log"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func launchTool(t Tool, ctx Context) error {
	dir := expand(t.WorkingDir, ctx)
	if dir == "" {
		dir = ctx.Game
	}
	args := expandArgs(t.Arguments, ctx)
	exe := expand(t.Executable, ctx)
	log.Printf("tools: starting %q (%s)", t.Name, exe)
	_, err := launch.Start(dir, exe, args...)
	return err
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
