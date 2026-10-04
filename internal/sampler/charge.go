package sampler

import (
	"path/filepath"
	"sort"
	"strings"
)

// ChargeResult is the sampled duration attributed to each mod and to all
// frames that do not belong to a mapped mod assembly.
type ChargeResult struct {
	Mods          map[string]int64 `json:"mods"`
	OtherMs       int64            `json:"otherMs"`
	ThreadSamples int              `json:"threadSamples"`
}

// Charge attributes each interval ending at a sample to its innermost frame
// whose assembly is present in assemblyToMod. The final sample has no
// following timestamp and therefore contributes no interval.
func Charge(samples []Sample, threadID uint32, assemblyToMod map[string]string) ChargeResult {
	selected := make([]Sample, 0, len(samples))
	for _, sample := range samples {
		if sample.ThreadID == threadID {
			selected = append(selected, sample)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].Timestamp < selected[j].Timestamp
	})
	result := ChargeResult{Mods: make(map[string]int64), ThreadSamples: len(selected)}
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
		modID := innermostMod(selected[i].Frames, assemblyToMod)
		if modID == "" {
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

func innermostMod(frames []Frame, assemblyToMod map[string]string) string {
	for _, frame := range frames {
		if modID := mappedAssembly(frame.Assembly, assemblyToMod); modID != "" {
			return modID
		}
	}
	return ""
}

func mappedAssembly(assembly string, assemblyToMod map[string]string) string {
	if modID := assemblyToMod[assembly]; modID != "" {
		return modID
	}
	base := assemblyBase(assembly)
	for name, modID := range assemblyToMod {
		if assemblyBase(name) == base {
			return modID
		}
	}
	return ""
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
