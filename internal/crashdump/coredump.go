package crashdump

import (
	"context"
	"os/exec"
	"time"
)

const coredumpctlTimeout = 5 * time.Second

// Coredump asks systemd-coredump about a crash of command comm since `since`; ok is false when coredumpctl is not
// installed, holds no dump, or refuses (reading another user's dump needs root, which is never asked for).
func Coredump(ctx context.Context, comm string, since time.Time) (Fault, bool) {
	path, err := exec.LookPath("coredumpctl")
	if err != nil {
		return Fault{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, coredumpctlTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "info", "--no-pager", "--since", since.Local().Format("2006-01-02 15:04:05"), "--", comm).Output() // #nosec G204 -- fixed binary; comm is a game executable name
	if err != nil {
		return Fault{}, false
	}
	return ParseCoredumpInfo(string(out))
}
