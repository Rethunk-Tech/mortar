package settings

import (
	"fmt"
	"slices"
)

// RememberNexusDownloadServers merges shortNames into NexusSeenDownloadServers in stable order.
func RememberNexusDownloadServers(v *Settings, shortNames []string) {
	if len(shortNames) == 0 {
		return
	}
	seen := slices.Clone(v.NexusSeenDownloadServers)
	for _, name := range shortNames {
		if name == "" || slices.Contains(seen, name) {
			continue
		}
		seen = append(seen, name)
	}
	v.NexusSeenDownloadServers = seen
}

func normalizeNexus(s *Settings) {
	if s.NexusSeenDownloadServers == nil {
		s.NexusSeenDownloadServers = []string{}
	}
	if s.NexusPreferredDownloadServer != "" && !slices.Contains(s.NexusSeenDownloadServers, s.NexusPreferredDownloadServer) {
		s.NexusPreferredDownloadServer = ""
	}
}

// RedirectOtherGames is whether nxm links for games Mortar does not take go to the handler Mortar replaced.
func (s Settings) RedirectOtherGames() bool {
	if s.NxmPreviousHandlers["nxm"] == "" {
		return false
	}
	if s.NxmRedirectOtherGames == nil {
		return true
	}
	return *s.NxmRedirectOtherGames
}

func validateNexus(next Settings) error {
	if next.NexusPreferredDownloadServer != "" && !slices.Contains(next.NexusSeenDownloadServers, next.NexusPreferredDownloadServer) {
		return fmt.Errorf("unknown Nexus download server %q", next.NexusPreferredDownloadServer)
	}
	return nil
}
