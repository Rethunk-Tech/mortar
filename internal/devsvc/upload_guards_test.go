package devsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/zalando/go-keyring"
)

func TestThunderstoreTokenIsKeptPerProjectTeamAndForgotten(t *testing.T) {
	keyring.MockInit()
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	const g = "lethal-company"
	if err := svc.SetThunderstoreToken(g, dir, ""); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("empty token: %v", err)
	}
	if svc.HasThunderstoreToken(g, dir) {
		t.Fatal("token before it was set")
	}
	if err := svc.SetThunderstoreToken(g, dir, "tok"); err != nil || !svc.HasThunderstoreToken(g, dir) {
		t.Fatalf("set: %v", err)
	}
	if err := svc.ForgetThunderstoreToken(g, dir); err != nil || svc.HasThunderstoreToken(g, dir) {
		t.Fatalf("forget: %v", err)
	}
	if svc.HasThunderstoreToken(g, filepath.Join(dir, "elsewhere")) {
		t.Error("an unknown project has a token")
	}
}

func TestUploadGuardsRefuseBeforeAnyRequest(t *testing.T) {
	keyring.MockInit()
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	ctx := t.Context()
	// A game with no Thunderstore community cannot be uploaded to Thunderstore.
	if _, err := svc.UploadThunderstore(ctx, "stardew-valley", dir); err == nil {
		t.Error("no community or project: want an error")
	}
	writePackedZip(t, dir)
	if _, err := svc.UploadNexus(ctx, "lethal-company", dir, "1.0.0"); !strings.Contains(err.Error(), "Nexus Mods file id") {
		t.Errorf("nexus: %v", err)
	}
	if err := writeSites(dir, siteIDs{Nexus: NexusIDs{ModID: "5", FileID: "7"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadNexus(ctx, "lethal-company", dir, "1.0"); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("a version of two numbers: %v", err)
	}
	if _, err := svc.UploadNexus(ctx, "lethal-company", dir, "1.0.0"); err == nil {
		t.Error("nexus without an API key: want an error")
	}
}

func TestSitePagesNameTheProjectsFilesPage(t *testing.T) {
	if got := curseForgePage(31); got != "https://authors.curseforge.com/#/projects/31/files" {
		t.Errorf("curseforge page = %q", got)
	}
	if got := nexusPage("lethal-company", NexusIDs{ModID: "x"}); got != "" {
		t.Errorf("nexus page without a numeric mod id = %q", got)
	}
	if got := nexusPage("lethal-company", NexusIDs{ModID: "55"}); !strings.HasSuffix(got, "/55?tab=files") {
		t.Errorf("nexus page = %q", got)
	}
}

func TestGitHubErrorMapsAMissingTokenToAPermissionError(t *testing.T) {
	if err := githubError(gh.ErrNoToken); usererr.KindOf(err) != usererr.Permission {
		t.Errorf("no token: %v", err)
	}
	other := errors.New("boom")
	if !errors.Is(githubError(other), other) {
		t.Error("another error is passed through")
	}
}

func TestIgnoreDependencyChecksThePackageAndTheProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	if err := svc.IgnoreDependency("lethal-company", dir, "not a package"); usererr.KindOf(err) != usererr.Invalid {
		t.Errorf("bad package: %v", err)
	}
	if err := svc.IgnoreDependency("lethal-company", filepath.Join(dir, "other"), "Ab-Cd"); err == nil {
		t.Error("unknown project: want an error")
	}
	if err := svc.IgnoreDependency("lethal-company", dir, "Ab-Cd"); err != nil {
		t.Fatal(err)
	}
	if got := svc.Settings.Get().GamePrefs("lethal-company").DevProjects[0].IgnoredDeps; len(got) != 1 || got[0] != "Ab-Cd" {
		t.Errorf("ignored = %v", got)
	}
}

func TestWriteTestWithWithoutARecordWritesNothingForNoIDs(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestWith(dir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".mortar")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a record was written: %v", err)
	}
	if err := writeTestWith(dir, []string{"Ab-Cd"}); err != nil {
		t.Fatal(err)
	}
	if got := readTestWith(dir); len(got) != 1 || got[0] != "Ab-Cd" {
		t.Errorf("test with = %v", got)
	}
}

func TestReplaceTreeKeepsOnlyGit(t *testing.T) {
	dest, gen := t.TempDir(), t.TempDir()
	for _, f := range []string{".git/HEAD", "old/file", "stale.txt"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dest, f)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := safefs.WriteFile(filepath.Join(dest, f), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := safefs.WriteFile(filepath.Join(gen, "new.txt"), []byte("n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceTree(dest, gen); err != nil {
		t.Fatal(err)
	}
	for f, want := range map[string]bool{".git/HEAD": true, "new.txt": true, "stale.txt": false, "old": false} {
		if _, err := os.Stat(filepath.Join(dest, f)); (err == nil) != want {
			t.Errorf("%s present = %v, want %v", f, err == nil, want)
		}
	}
}

func TestFromGitHubStopsRetryingWhenCancelled(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = func(_ context.Context, _, name string, args ...string) ([]byte, error) {
		if len(args) > 1 && args[1] == "clone" {
			return nil, errors.New("not ready")
		}
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	dest := filepath.Join(t.TempDir(), "r")
	start := time.Now()
	err := svc.fromGitHub(ctx, components.DevTemplate{Repo: "a/b"}, NewMod{Owner: "o", Repo: "r", Parent: t.TempDir()}, dest)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > cloneRetryDelay {
		t.Errorf("err = %v after %v", err, time.Since(start))
	}
}
