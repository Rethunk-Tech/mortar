package secret

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExplainMapsMissingProvider(t *testing.T) {
	raw := errors.New(`The name org.freedesktop.secrets was not provided by any .service files: org.freedesktop.DBus.Error.ServiceUnknown`)
	got := explain(raw).Error()
	if !strings.Contains(got, "No keyring service is running") || strings.Contains(got, "DBus") {
		t.Fatalf("got %q", got)
	}
	other := errors.New("boom")
	if !errors.Is(explain(other), other) || explain(nil) != nil {
		t.Fatal("unrelated errors must pass through")
	}
}

func TestGetOutlivesTimeoutThenCaches(t *testing.T) {
	release := make(chan struct{})
	var reads atomic.Int32
	keyringGet = func(string) (string, error) {
		reads.Add(1)
		<-release
		return "token", nil
	}
	unlocked := make(chan struct{}, 1)
	OnUnlocked = func() { unlocked <- struct{}{} }
	t.Cleanup(func() {
		keyringGet = func(name string) (string, error) { return "", errors.New("unset " + name) }
		OnUnlocked = func() {}
	})

	if _, err := Get("k"); !errors.Is(err, ErrWaitingForUnlock) {
		t.Fatalf("want ErrWaitingForUnlock, got %v", err)
	}
	start := time.Now()
	if _, err := Get("k"); !errors.Is(err, ErrWaitingForUnlock) || time.Since(start) > getTimeout/2 {
		t.Fatalf("a read behind a stuck prompt must fail at once, got %v after %v", err, time.Since(start))
	}
	close(release)
	select {
	case <-unlocked:
	case <-time.After(5 * time.Second):
		t.Fatal("no unlock notification")
	}
	if v, err := Get("k"); err != nil || v != "token" {
		t.Fatalf("late value: %q, %v", v, err)
	}
	if reads.Load() != 1 {
		t.Fatalf("the late result must be served without another backend read, got %d reads", reads.Load())
	}
}
