package launchsvc

import (
	"reflect"
	"testing"
)

func TestParsePerformanceReportDetail(t *testing.T) {
	lines := []string{
		"[Console Commands] Performance Counter for GameLoop.UpdateTicked:",
		"Mod | Avg Execution Time (last 60s) | Last Execution Time | Peak Execution Time",
		"-------------------------------- | ----------------------------- | ------------------- | -------------------",
		"Drachenkatze.AdvancedKeyBindings | 0.00 | 0.00 | 0.64",
		"Pathoschild.Automate | 0.03 | 0.00 | 28.48",
	}

	got := ParsePerformanceReport(lines)
	want := []PerformanceRow{
		{Name: "Drachenkatze.AdvancedKeyBindings", AverageMs: 0, PeakMs: 0.64},
		{Name: "Pathoschild.Automate", AverageMs: 0.03, PeakMs: 28.48},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParsePerformanceReport() = %#v, want %#v", got, want)
	}
}

func TestParsePerformanceReportSummary(t *testing.T) {
	lines := []string{
		"Event | Average (ms) | Peak (ms) | Calls",
		"------ | ------------ | --------- | -----",
		"GameLoop.UpdateTicked | 0.44 | 2.10 | 98",
	}

	got := ParsePerformanceReport(lines)
	want := []PerformanceRow{
		{Name: "GameLoop.UpdateTicked", AverageMs: 0.44, PeakMs: 2.1, Calls: 98},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParsePerformanceReport() = %#v, want %#v", got, want)
	}
}
