package queue

import "github.com/Rethunk-AI/mortar/internal/archive"

// waitsForSameMod reports an earlier item for the same Nexus mod and profile still under way. A file without a
// manifest waits for it, so a mod's main file installs before the optional files that go on top of it; other files
// of the same mod download in parallel. Call with s.mu held.
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

// parkOverlay marks the item as an optional file when optional is true, and puts it back in the queue while an
// earlier file of its mod is still under way; downloaded says the archive is on disk, so it next installs from there.
// It reports whether the item was parked.
func (s *Service) parkOverlay(id string, optional, downloaded bool) bool {
	if !optional {
		return false
	}
	s.mu.Lock()
	cur := s.find(id)
	if cur == nil || cur.State != StateDownloading {
		s.mu.Unlock()
		return false
	}
	cur.overlay = true
	if !s.waitsForSameMod(cur) {
		s.mu.Unlock()
		return false
	}
	cur.State, cur.Progress, cur.Speed = StateQueued, 0, 0
	if downloaded {
		cur.Progress, cur.readyZip = 100, true
	}
	s.mu.Unlock()
	s.publish(true)
	return true
}
