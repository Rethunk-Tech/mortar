package launchsvc

import (
	"strconv"
	"strings"
)

// PerformanceRow is one parsed row from SMAPI's performance report.
type PerformanceRow struct {
	Name      string  `json:"name"`
	AverageMs float64 `json:"averageMs"`
	PeakMs    float64 `json:"peakMs"`
	Calls     float64 `json:"calls"`
}

// ParsePerformanceReport parses the pipe-delimited tables emitted by SMAPI's performance commands.
func ParsePerformanceReport(lines []string) []PerformanceRow {
	columns := performanceColumns{name: -1, average: -1, peak: -1, calls: -1}
	rows := make([]PerformanceRow, 0)
	for _, line := range lines {
		fields := performanceFields(line)
		if len(fields) == 0 {
			continue
		}
		if next, ok := performanceHeader(fields); ok {
			columns = next
			continue
		}
		if columns.name < 0 || columns.average < 0 || columns.name >= len(fields) || columns.average >= len(fields) {
			continue
		}
		name := strings.TrimSpace(fields[columns.name])
		average, ok := performanceNumber(fields[columns.average])
		if !ok || name == "" {
			continue
		}
		row := PerformanceRow{Name: name, AverageMs: average}
		if columns.peak >= 0 && columns.peak < len(fields) {
			row.PeakMs, _ = performanceNumber(fields[columns.peak])
		}
		if columns.calls >= 0 && columns.calls < len(fields) {
			row.Calls, _ = performanceNumber(fields[columns.calls])
		}
		rows = append(rows, row)
	}
	return rows
}

// PerformanceReport exposes the pure parser to the console panel.
func (s *Service) PerformanceReport(lines []string) []PerformanceRow {
	return ParsePerformanceReport(lines)
}

type performanceColumns struct {
	name    int
	average int
	peak    int
	calls   int
}

func performanceFields(line string) []string {
	if !strings.Contains(line, "|") {
		return nil
	}
	fields := strings.Split(line, "|")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	if len(fields) == 0 || strings.Trim(fields[0], "-") == "" {
		return nil
	}
	return fields
}

func performanceHeader(fields []string) (performanceColumns, bool) {
	columns := performanceColumns{name: -1, average: -1, peak: -1, calls: -1}
	averageScore := -1
	for i, field := range fields {
		lower := strings.ToLower(field)
		switch {
		case columns.name < 0 && (lower == "mod" || lower == "event" || lower == "collection"):
			columns.name = i
		case columns.peak < 0 && strings.Contains(lower, "peak"):
			columns.peak = i
		case columns.calls < 0 && strings.Contains(lower, "call"):
			columns.calls = i
		}
		if strings.Contains(lower, "avg") || strings.Contains(lower, "average") {
			score := 1
			if strings.Contains(lower, "execution") || strings.Contains(lower, "time") || strings.Contains(lower, "ms") {
				score += 2
			}
			if strings.Contains(lower, "game+mods") {
				score += 4
			} else if strings.Contains(lower, "mods") {
				score += 3
			}
			if score > averageScore {
				averageScore = score
				columns.average = i
			}
		}
	}
	if columns.name < 0 || columns.average < 0 {
		return performanceColumns{}, false
	}
	return columns, true
}

func performanceNumber(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" || value == "—" {
		return 0, false
	}
	value = strings.TrimSpace(strings.TrimSuffix(value, "ms"))
	number, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	return number, err == nil
}
