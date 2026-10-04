package queue

import "github.com/Rethunk-AI/mortar/internal/archive"

// waitsForSameMod holds a Nexus item while an earlier item for the same mod and profile is still under way, so a
// mod's main file installs before the optional files that go on top of it. Call with s.mu held.
func (s *Service) waitsForSameMod(it *Item) bool {
	if it.ModID <= 0 {
		return false
	}
	for _, other := range s.items {
		if other == it {
			return false
		}
		if other.ModID == it.ModID && other.Game == it.Game && other.Profile == it.Profile && !dismissable(other.State) {
			return true
		}
	}
	return false
}

// manifestLess reports an archive without a SMAPI manifest or installer: an optional file laid over its mod's main
// file, which the profile installs as such without asking whether to join or keep it apart.
func manifestLess(path string) bool {
	p, err := archive.PreviewArchive(path)
	return err == nil && !p.Truncated && !p.Fomod && len(p.Manifests) == 0
}
