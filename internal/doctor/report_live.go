package doctor

import (
	"os"

	"github.com/Rethunk-Tech/mortar/internal/nxm"
)

// LiveWithNativeHosts is FromLive plus native-messaging host checks for this executable.
func LiveWithNativeHosts(in Live) Report {
	rep := FromLive(in)
	exe, err := os.Executable()
	if err != nil {
		return rep
	}
	rep.Checks = AppendNativeHostChecks(rep.Checks, nxm.StatusForExecutable(exe))
	return rep
}
