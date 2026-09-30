package github

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type stallReader struct {
	started chan struct{}
	release <-chan struct{}
}

func (s *stallReader) Read(p []byte) (int, error) {
	close(s.started)
	<-s.release
	for i := range p {
		p[i] = 'y'
	}
	return len(p), io.EOF
}

func TestIdleReaderTimeoutDoesNotWriteCallerBuffer(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	r := &idleReader{
		done:  make(chan struct{}),
		cause: func() error { return nil },
		r:     &stallReader{started: started, release: release},
		idle:  20 * time.Millisecond,
	}
	p := make([]byte, 32)
	n, err := r.Read(p)
	if n != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Read = %d, %v", n, err)
	}
	<-started
	for i := range p {
		p[i] = 'x'
	}
	n, err = r.Read(p)
	if n != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Read = %d, %v", n, err)
	}
	close(release)
	time.Sleep(50 * time.Millisecond)
	for i, b := range p {
		if b != 'x' {
			t.Fatalf("caller buffer mutated at %d: %q", i, b)
		}
	}
}

func TestIdleReaderCancelDoesNotWriteCallerBuffer(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	r := &idleReader{
		done: done,
		cause: func() error {
			select {
			case <-done:
				return context.Canceled
			default:
				return nil
			}
		},
		r:    &stallReader{started: started, release: release},
		idle: time.Hour,
	}
	p := make([]byte, 32)
	go func() {
		<-started
		close(done)
	}()
	n, err := r.Read(p)
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("Read = %d, %v", n, err)
	}
	for i := range p {
		p[i] = 'x'
	}
	n, err = r.Read(p)
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("second Read = %d, %v", n, err)
	}
	close(release)
	time.Sleep(50 * time.Millisecond)
	for i, b := range p {
		if b != 'x' {
			t.Fatalf("caller buffer mutated at %d: %q", i, b)
		}
	}
}
