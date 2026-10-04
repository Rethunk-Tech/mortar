//go:build updatetest

package updatesvc_test

import (
	"bytes"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests build two server-mode copies of testdata/fixture (0.1.1 and 0.1.2), serve a manifest from a local HTTP
// server and let the real Wails updater download, verify, swap and relaunch the binary on disk.
// Run: go test -tags updatetest -run TestUpdaterE2E ./internal/updatesvc

var (
	buildOnce sync.Once
	oldBin    string
	newBin    string
)

func goBuild(t *testing.T, version, out string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "go", "build", "-tags", "server production updatetest", "-ldflags", "-X main.version="+version, "-o", out, "./testdata/fixture")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", version, err, b)
	}
}

func binaries(t *testing.T) (string, string) {
	t.Helper()
	buildOnce.Do(func() {
		// Shared by every test, so not t.TempDir; the OS tmp cleaner owns it.
		dir, err := os.MkdirTemp("", "updatesvc-e2e-")
		if err != nil {
			t.Fatal(err)
		}
		oldBin, newBin = filepath.Join(dir, "old"), filepath.Join(dir, "new")
		goBuild(t, "0.1.1", oldBin)
		goBuild(t, "0.1.2", newBin)
	})
	if oldBin == "" {
		t.Skip("fixture build failed in an earlier test")
	}
	return oldBin, newBin
}

func run(t *testing.T, name string, args ...string) {
	t.Helper()
	if b, err := exec.CommandContext(t.Context(), name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, b)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type keypair struct{ priv, pubB64 string }

func genKey(t *testing.T, dir, name string) keypair {
	t.Helper()
	priv := filepath.Join(dir, name+".key")
	run(t, "wails3", "updater", "genkey", "-out", priv)
	pem := readFile(t, priv+".pub")
	// The fixture takes the PEM bytes base64-wrapped so they survive an environment variable.
	return keypair{priv: priv, pubB64: base64.StdEncoding.EncodeToString(pem)}
}

// site serves a release directory and writes its manifest: version, artifact bytes and signing key are the knobs.
type site struct {
	dir, url string
}

func newSite(t *testing.T) *site {
	t.Helper()
	s := &site{dir: t.TempDir()}
	srv := httptest.NewServer(http.FileServer(http.Dir(s.dir)))
	t.Cleanup(srv.Close)
	s.url = srv.URL
	return s
}

func (s *site) publish(t *testing.T, version string, artifact []byte, key string) {
	t.Helper()
	// The manifest is built from the genuine artifact and then served alongside whatever bytes the caller supplies.
	asset := filepath.Join(s.dir, "fixture-linux-"+runtime.GOARCH)
	if err := os.WriteFile(asset, artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"updater", "manifest", "-version", version, "-url-prefix", s.url, "-output", filepath.Join(s.dir, "manifest.json")}
	if key != "" {
		args = append(args, "-key", key)
	}
	run(t, "wails3", append(args, asset)...)
	// The fixture runs as a bare Linux binary, which updates from the portable entry, as release:manifest marks it.
	manifest := filepath.Join(s.dir, "manifest.json")
	b := bytes.ReplaceAll(readFile(t, manifest), []byte(`"platform": "linux"`), []byte(`"platform": "linux-portable"`))
	if err := os.WriteFile(manifest, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (s *site) tamper(t *testing.T) {
	t.Helper()
	asset := filepath.Join(s.dir, "fixture-linux-"+runtime.GOARCH)
	b := readFile(t, asset)
	b = append(b, 0)
	if err := os.WriteFile(asset, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return port
}

// launch runs a fresh copy of bin against site and returns the installed path and the fixture's log file.
func launch(t *testing.T, bin string, s *site, pubB64 string) (installed, logPath string) {
	t.Helper()
	root := t.TempDir()
	installed = filepath.Join(root, "mortar")
	// A hard link: the helper unlinks the path and renames the update in, so the pristine build is never written.
	if err := os.Link(bin, installed); err != nil {
		t.Fatal(err)
	}
	logPath = filepath.Join(root, "fixture.log")
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool { return strings.HasPrefix(kv, "WAILS_UPDATER_") })
	env = append(env,
		"MORTAR_UPDATE_MANIFEST_URL="+s.url+"/manifest.json",
		"FIXTURE_PUBKEY_B64="+pubB64,
		"FIXTURE_LOG="+logPath,
		"WAILS_SERVER_PORT="+freePort(t),
		"TMPDIR="+root,
	)
	cmd := exec.CommandContext(t.Context(), installed)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("fixture did not exit")
	}
	return installed, logPath
}

// waitLog polls until the log holds want, which the relaunched process writes after the helper's swap.
func waitLog(t *testing.T, logPath, want string) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(logPath); err == nil && strings.Contains(string(b), want) {
			return string(b)
		}
		time.Sleep(100 * time.Millisecond)
	}
	b, _ := os.ReadFile(logPath)
	t.Fatalf("log never contained %q:\n%s", want, b)
	return ""
}

func setup(t *testing.T) (oldB, newB []byte, s *site, good keypair) {
	t.Helper()
	if testing.Short() || runtime.GOOS != "linux" {
		t.Skip("updater e2e builds and runs binaries on linux")
	}
	if _, err := exec.LookPath("wails3"); err != nil {
		t.Skip("wails3 CLI not on PATH")
	}
	o, n := binaries(t)
	return readFile(t, o), readFile(t, n), newSite(t), genKey(t, t.TempDir(), "good")
}

func TestUpdaterE2EApplies(t *testing.T) {
	_, newB, s, key := setup(t)
	s.publish(t, "0.1.2", newB, key.priv)
	installed, logPath := launch(t, oldBin, s, key.pubB64)
	log := waitLog(t, logPath, "start version=0.1.2")
	if !strings.Contains(log, "check found=0.1.2") || !strings.Contains(log, "staged") || !strings.Contains(log, "restarting") {
		t.Fatalf("old process did not check, stage and restart:\n%s", log)
	}
	// The relaunched 0.1.2 asks the same manifest and must find nothing newer.
	waitLog(t, logPath, "check release=<nil> err=<nil>")
	if !bytes.Equal(readFile(t, installed), newB) {
		t.Fatal("binary on disk is not the 0.1.2 build")
	}
	if _, err := os.Stat(installed + ".bak"); err == nil {
		t.Fatal("backup left behind after a successful swap")
	}
}

// refused runs the old binary against a release that must not install and checks the disk is untouched.
func refused(t *testing.T, oldB []byte, s *site, pubB64, want string) {
	t.Helper()
	installed, logPath := launch(t, oldBin, s, pubB64)
	log := waitLog(t, logPath, want)
	if strings.Contains(log, "staged") || strings.Contains(log, "start version=0.1.2") {
		t.Fatalf("a refused release was staged or applied:\n%s", log)
	}
	if !bytes.Equal(readFile(t, installed), oldB) {
		t.Fatal("binary on disk changed")
	}
}

func TestUpdaterE2ETamperedBinaryRefused(t *testing.T) {
	oldB, newB, s, key := setup(t)
	s.publish(t, "0.1.2", newB, key.priv)
	s.tamper(t)
	refused(t, oldB, s, key.pubB64, "digest mismatch")
}

func TestUpdaterE2EBadSignatureRefused(t *testing.T) {
	oldB, newB, s, key := setup(t)
	other := genKey(t, t.TempDir(), "other")
	s.publish(t, "0.1.2", newB, other.priv)
	refused(t, oldB, s, key.pubB64, "did not verify")
}

func TestUpdaterE2EUnsignedRefused(t *testing.T) {
	oldB, newB, s, key := setup(t)
	s.publish(t, "0.1.2", newB, "")
	refused(t, oldB, s, key.pubB64, "not signed")
}

func TestUpdaterE2EDowngradeIgnored(t *testing.T) {
	oldB, newB, s, key := setup(t)
	s.publish(t, "0.1.0", newB, key.priv)
	refused(t, oldB, s, key.pubB64, "check release=<nil> err=<nil>")
}

func TestMain(m *testing.M) {
	code := m.Run()
	if oldBin != "" {
		_ = os.Remove(oldBin)
		_ = os.Remove(newBin)
		_ = os.Remove(filepath.Dir(oldBin))
	}
	os.Exit(code)
}
