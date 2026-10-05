package profile

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// counted records every HealthEvent a service emits.
type counted struct {
	mu      sync.Mutex
	notices []HealthNotice
}

func (c *counted) emit(_ string, data any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := data.(HealthNotice); ok {
		c.notices = append(c.notices, n)
	}
}

func (c *counted) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.notices)
}

func TestHealthPassRunsWeeklyFlaggedAndNotWhileRunning(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	seen := &counted{}
	svc.HealthEmit = seen.emit
	now := time.Now()
	svc.healthPass(context.Background(), now)
	if seen.len() != 1 {
		t.Fatalf("first pass checked %d profiles", seen.len())
	}
	svc.healthPass(context.Background(), now.Add(time.Hour))
	if seen.len() != 1 {
		t.Fatal("checked again within the week")
	}
	svc.FlagHealth("stardew")
	svc.healthPass(context.Background(), now.Add(time.Hour))
	svc.healthPass(context.Background(), now.Add(2*time.Hour))
	if seen.len() != 2 {
		t.Fatalf("a flag runs exactly one more check, got %d", seen.len())
	}
	e.Running = func(string, string) bool { return true }
	svc.healthPass(context.Background(), now.Add(8*24*time.Hour))
	if seen.len() != 2 {
		t.Fatal("checked while the game runs")
	}
	e.Running = nil
	e.GameRunning = func(string) bool { return true }
	svc.healthPass(context.Background(), now.Add(8*24*time.Hour))
	if seen.len() != 2 {
		t.Fatal("checked while the game runs, however it was started")
	}
	e.GameRunning = nil
	svc.healthPass(context.Background(), now.Add(8*24*time.Hour))
	if seen.len() != 3 || seen.notices[2].Profile != p.ID {
		t.Fatalf("a week later = %#v", seen.notices)
	}
}

func TestHealthBadgeFollowsTheLatestCheck(t *testing.T) {
	t.Parallel()
	e, svc, p := healthEnv(t)
	if _, err := svc.ProfileHealth("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.HealthBadges("stardew"); len(got) != 0 {
		t.Fatalf("clean profile badge = %v", got)
	}
	folder := filepath.Join(e.mods(p.ID), p.Entries[0].Key)
	if err := os.RemoveAll(folder); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProfileHealth("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.HealthBadges("stardew"); got[p.ID] != 1 {
		t.Fatalf("badge = %v", got)
	}
	if _, err := svc.RestoreDriftEntry("stardew", p.ID, p.Entries[0].Key); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProfileHealth("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.HealthBadges("stardew"); len(got) != 0 {
		t.Fatalf("badge after a clean check = %v", got)
	}
}

func TestRunHealthChecksWaitsForStartupToSettle(t *testing.T) {
	t.Parallel()
	_, svc, _ := healthEnv(t)
	seen := &counted{}
	svc.HealthEmit = seen.emit
	ctx, cancel := context.WithCancel(context.Background())
	settled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		svc.RunHealthChecks(ctx, settled)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	if seen.len() != 0 {
		t.Fatal("checked before startup settled")
	}
	close(settled)
	deadline := time.Now().Add(5 * time.Second)
	for seen.len() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done
	if seen.len() != 1 {
		t.Fatalf("checks after settling = %d", seen.len())
	}
}
