package control

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func echoHandler(context.Context, string, Params) (any, error) { return true, nil }

func waitLive(t *testing.T, dir string) {
	t.Helper()
	for range 100 {
		if _, ok := controlwire.Live(dir); ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the app never became reachable")
}

// A second process that takes the discovery file over and ends gives it back to the app that still runs.
func TestAnEndingSecondServerGivesTheControlFileBackToTheRunningApp(t *testing.T) {
	dir := t.TempDir()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() { _ = Serve(ctx, dir, "a", echoHandler) }()
	waitLive(t, dir)
	path := filepath.Join(dir, controlwire.FileName)
	first, _ := fsx.ReadFile(path)
	ctxB, stopB := context.WithCancel(t.Context())
	doneB := make(chan error, 1)
	go func() { doneB <- Serve(ctxB, dir, "b", echoHandler) }()
	for range 200 {
		if cur, _ := fsx.ReadFile(path); len(cur) > 0 && string(cur) != string(first) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopB()
	<-doneB
	time.Sleep(50 * time.Millisecond)
	var hello controlwire.Hello
	if err := controlwire.CallDir(dir, "hello", nil, &hello, time.Second); err != nil || hello.Version != "a" {
		t.Fatalf("the running app lost its control file: %+v, %v", hello, err)
	}
}

// A control file removed while the app runs is written back.
func TestARemovedControlFileIsWrittenBack(t *testing.T) {
	old := republishEvery
	republishEvery = 20 * time.Millisecond
	defer func() { republishEvery = old }()
	dir := t.TempDir()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() { _ = Serve(ctx, dir, "a", echoHandler) }()
	waitLive(t, dir)
	if err := os.Remove(filepath.Join(dir, controlwire.FileName)); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		if _, ok := controlwire.Live(dir); ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the control file was not written back")
}
