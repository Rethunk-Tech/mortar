package launch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

var fast = Timing{Timeout: 300 * time.Millisecond, Poll: 5 * time.Millisecond}

func TestRunStartsWhenLogRewritten(t *testing.T) {
	log := filepath.Join(t.TempDir(), "SMAPI-latest.txt")
	if err := os.WriteFile(log, []byte("old run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(log, old, old); err != nil {
		t.Fatal(err)
	}
	var got []string
	run := func(dir, name string, args ...string) error {
		got = append([]string{dir, name}, args...)
		go func() {
			time.Sleep(30 * time.Millisecond)
			_ = os.WriteFile(log, []byte("SMAPI 4.5.2 with Stardew Valley 1.6.15\nLoading mods\npartial"), 0o600)
		}()
		return nil
	}
	var mu sync.Mutex
	var lines []string
	err := Run(context.Background(), run, Command{Dir: "/g", Name: "steam", Args: []string{"-applaunch", "1"}, LogFile: log}, fast, func(l string) {
		mu.Lock()
		lines = append(lines, l)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"/g", "steam", "-applaunch", "1"}) {
		t.Fatalf("command = %v", got)
	}
	if !slices.Equal(lines, []string{"SMAPI 4.5.2 with Stardew Valley 1.6.15", "Loading mods"}) {
		t.Fatalf("lines = %q (the old run's line or a partial line leaked)", lines)
	}
}

func TestRunIgnoresStaleLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "SMAPI-latest.txt")
	if err := os.WriteFile(log, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(log, old, old); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), func(string, string, ...string) error { return nil }, Command{LogFile: log, Failure: HintSteam}, fast, func(string) {})
	var f *Failure
	if !errors.As(err, &f) || f.Hint != HintSteam {
		t.Fatalf("err = %v, want a steam-hint failure", err)
	}
}

func TestRunStartError(t *testing.T) {
	boom := errors.New("boom")
	err := Run(context.Background(), func(string, string, ...string) error { return boom }, Command{Failure: HintLaunchOptions}, fast, nil)
	var f *Failure
	if !errors.As(err, &f) || !errors.Is(err, boom) || f.Hint != HintLaunchOptions {
		t.Fatalf("err = %v", err)
	}
}
