package launch

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestPIDUint32RejectsNegative(t *testing.T) {
	t.Parallel()
	if _, err := pidUint32(-1); err == nil {
		t.Fatal("expected error for negative pid")
	}
}

func TestWaitMillisClampsLargeDuration(t *testing.T) {
	t.Parallel()
	d := time.Duration(int64(math.MaxUint32)+1) * time.Millisecond
	if got := waitMillis(d); got != math.MaxUint32 {
		t.Fatalf("waitMillis clamp: got %d", got)
	}
}

func TestStartedProcessIsTrackedAndKillable(t *testing.T) {
	done, err := StartHidden(context.Background(), nil, "", "cmd.exe", "/c", "ping -n 60 127.0.0.1 >nul")
	if err != nil {
		t.Fatal(err)
	}
	jobsMu.Lock()
	pids := make([]int, 0, len(jobs))
	for pid := range jobs {
		pids = append(pids, pid)
	}
	jobsMu.Unlock()
	if len(pids) == 0 {
		t.Fatal("started process is not in a job")
	}
	for _, pid := range pids {
		killTree(pid)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("process survived killTree")
	}
}
