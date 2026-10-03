package profile

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
)

const (
	ChannelMain     = "main"
	ChannelOptional = "optional"
	ChannelBeta     = "beta"
	historyChannel  = "channel"
)

// SetUpdateChannel records which Nexus file categories and prerelease versions a mod will accept as updates.
func (s *Service) SetUpdateChannel(game, id, key, channel string) (Profile, error) {
	return s.store.SetUpdateChannel(game, id, key, channel)
}

// SetUpdateChannel records which Nexus file categories and prerelease versions a mod will accept as updates.
func (s *Store) SetUpdateChannel(game, id, key, channel string) (Profile, error) {
	ch, err := parseUpdateChannel(channel)
	if err != nil {
		return Profile{}, err
	}
	stored := ch
	if stored == ChannelMain {
		stored = ""
	}
	return s.patchEntry(game, id, key, func(e *Entry) error {
		if e.UpdateChannel == stored {
			return nil
		}
		s.historyKind = historyChannel
		s.historyLabel = "Update channel " + ch
		e.UpdateChannel = stored
		return nil
	})
}

// NewestChannelFile is the newest file this channel will take that is newer than installed.
func NewestChannelFile(channel, installed string, files []nexus.File) (nexus.File, bool) {
	var best nexus.File
	found := false
	for _, f := range files {
		if !FileOnChannel(channel, f) {
			continue
		}
		if !versionNewer(f.Version, installed) {
			continue
		}
		if !found || versionNewer(f.Version, best.Version) || (sameVersion(f.Version, best.Version) && f.FileID > best.FileID) {
			best, found = f, true
		}
	}
	return best, found
}

// FileOnChannel is whether this Nexus file is an update this channel will show.
func FileOnChannel(channel string, f nexus.File) bool {
	cat := strings.ToUpper(strings.TrimSpace(f.Category))
	if cat == "OLD_VERSION" {
		return false
	}
	pre := prereleaseFile(f)
	switch NormalizeChannel(channel) {
	case ChannelOptional:
		return !pre && (cat == "MAIN" || cat == "OPTIONAL")
	case ChannelBeta:
		return cat == "MAIN" || pre
	default:
		return !pre && cat == "MAIN"
	}
}

// NormalizeChannel maps empty to main.
func NormalizeChannel(channel string) string {
	ch := strings.ToLower(strings.TrimSpace(channel))
	if ch == "" {
		return ChannelMain
	}
	return ch
}

func parseUpdateChannel(channel string) (string, error) {
	ch := NormalizeChannel(channel)
	switch ch {
	case ChannelMain, ChannelOptional, ChannelBeta:
		return ch, nil
	default:
		return "", fmt.Errorf("update channel must be main, optional, or beta")
	}
}

func prereleaseFile(f nexus.File) bool {
	return prereleaseText(f.Version) || prereleaseText(f.Name) || prereleaseText(f.FileName)
}

func prereleaseText(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "beta") || strings.Contains(low, "alpha") {
		return true
	}
	if strings.Contains(low, "-rc") || strings.Contains(low, ".rc") || strings.Contains(low, " rc") || strings.HasPrefix(low, "rc") {
		return true
	}
	v := strings.TrimSpace(s)
	i := strings.IndexByte(v, '-')
	if i <= 0 {
		return false
	}
	tag, _, _ := strings.Cut(strings.ToLower(v[i+1:]), ".")
	switch tag {
	case "stable", "release", "final", "ga", "rtm":
		return false
	}
	_, ok := meta.CompareVersions(v[:i], v[:i])
	return ok
}

func versionNewer(a, b string) bool {
	c, ok := meta.CompareVersions(a, b)
	return ok && c > 0
}

func sameVersion(a, b string) bool {
	c, ok := meta.CompareVersions(a, b)
	return ok && c == 0
}
