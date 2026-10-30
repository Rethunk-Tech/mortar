package launchsvc

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-nettrace"
)

// charge attributes each interval ending at a sample to its innermost frame whose assembly is in assemblyToMod. The
// final sample has no following timestamp and therefore contributes no interval.
func charge(samples []nettrace.Sample, threadID uint64, assemblyToMod map[string]string) startupSamples {
	selected := make([]nettrace.Sample, 0, len(samples))
	for _, sample := range samples {
		if sample.ThreadID == threadID {
			selected = append(selected, sample)
		}
	}
	slices.SortStableFunc(selected, func(a, b nettrace.Sample) int { return cmp.Compare(a.Timestamp, b.Timestamp) })
	result := startupSamples{Schema: 1, Mods: make(map[string]int64), ThreadSamples: len(selected)}
	if len(selected) < 2 {
		return result
	}
	var otherNanos uint64
	modNanos := make(map[string]uint64)
	for i := 0; i+1 < len(selected); i++ {
		if selected[i+1].Timestamp <= selected[i].Timestamp {
			continue
		}
		weight := selected[i+1].Timestamp - selected[i].Timestamp
		if modID := innermostMod(selected[i].Frames, assemblyToMod); modID == "" {
			otherNanos += weight
		} else {
			modNanos[modID] += weight
		}
	}
	for modID, nanos := range modNanos {
		result.Mods[modID] = nanosToMilliseconds(nanos)
	}
	result.OtherMs = nanosToMilliseconds(otherNanos)
	return result
}

func innermostMod(frames []nettrace.Frame, assemblyToMod map[string]string) string {
	for _, frame := range frames {
		if modID := modOf(frame.Assembly, assemblyToMod); modID != "" {
			return modID
		}
	}
	return ""
}

// modOf maps a frame's assembly name, which may be a path or a full "Name, Version=..." identity, to its mod. The
// map's keys are bare assembly names.
func modOf(assembly string, assemblyToMod map[string]string) string {
	return assemblyToMod[assemblyBase(assembly)]
}

func assemblyBase(assembly string) string {
	assembly = strings.TrimSpace(strings.SplitN(assembly, ",", 2)[0])
	return strings.TrimSuffix(filepath.Base(strings.ReplaceAll(assembly, `\`, `/`)), ".dll")
}

func nanosToMilliseconds(nanos uint64) int64 {
	milliseconds := (nanos + 500_000) / 1_000_000
	const maxInt64 = uint64(1<<63 - 1)
	if milliseconds > maxInt64 {
		return int64(maxInt64)
	}
	return int64(milliseconds)
}
