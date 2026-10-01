package launch

import (
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
