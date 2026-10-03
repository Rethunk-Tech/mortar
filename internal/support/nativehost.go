package support

import (
	"os"

	"github.com/Rethunk-AI/mortar/internal/nxm"
)

// RepairNativeHosts rewrites native-messaging host manifests for installed browsers.
func (s *Service) RepairNativeHosts() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return nxm.RepairNativeHosts(exe)
}
