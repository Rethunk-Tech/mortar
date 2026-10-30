package fonts

import (
	"context"
	"os/exec"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/hostenv"
)

const listTimeout = 10 * time.Second

// List is the installed font families, from fontconfig.
func List() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "fc-list", ":", "family")
	cmd.Env = hostenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return ParseFcList(string(out)), nil
}
