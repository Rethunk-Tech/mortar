package updatesvc

import (
	"os"
	"runtime"

	"github.com/Rethunk-AI/mortar/internal/selfexe"
)

// PortablePlatform is the manifest platform of the bare Linux program. The "linux" artifacts are the AppImages, so
// the bare program has its own entries, and each flavour replaces itself with its own kind of file.
const PortablePlatform = "linux-portable"

// updatePlatform is the manifest platform to update from: PortablePlatform for a Linux exe that is not running from an
// AppImage (launchable is the path selfexe.Launchable gives for exe), else "" for the updater's runtime.GOOS default.
func updatePlatform(goos, exe, launchable string) string {
	if goos == "linux" && launchable == exe {
		return PortablePlatform
	}
	return ""
}

func runningPlatform() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return updatePlatform(runtime.GOOS, exe, selfexe.Launchable(exe))
}
