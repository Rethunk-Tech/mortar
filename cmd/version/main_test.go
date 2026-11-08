package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// releaseFiles are the five copies cmd/version rewrites, with the shapes the real files have. {version} stands in
// for the version they carry, which the test writes.
var releaseFiles = map[string]string{
	"build/config.yml": "version: \"3\"\n\ninfo:\n  productName: \"Mortar\"\n  version: \"{version}\" # the only place it is set\n",
	"build/windows/wails.exe.manifest": `<assembly manifestVersion="1.0">
    <assemblyIdentity type="win32" name="tech.rethunk.mortar" version="{version}" processorArchitecture="*"/>
</assembly>`,
	"build/windows/info.json": `{
  "fixed": {
    "file_version": "{version}"
  },
  "info": {
    "0000": {
      "ProductVersion": "{version}",
      "FileVersion": "{version}"
    }
  }
}
`,
	"build/linux/tech.rethunk.Mortar.metainfo.xml": `<component type="desktop-application">
  <releases>
    <release version="{version}" date="2026-11-04">
      <description><p>Current.</p></description>
    </release>
    <release version="0.5.0" date="2026-11-03">
      <description><p>Previous.</p></description>
    </release>
  </releases>
</component>
`,
	"build/linux/aur/PKGBUILD":           "pkgname=mortar\npkgver={version}\npkgrel=1\n",
	"build/windows/nsis/wails_tools.nsh": "!ifndef INFO_PRODUCTVERSION\n    !define INFO_PRODUCTVERSION \"{version}\"\n!endif\n",
}

// releaseTree writes every release file, each already carrying version.
func releaseTree(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range releaseFiles {
		writeFile(t, root, rel, withVersion(body, version))
	}
	return root
}

func withVersion(body, version string) string {
	return strings.ReplaceAll(body, "{version}", version)
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readAll(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// inDir runs fn with the process's working directory at root, which cmd/version reads its files from.
func inDir(t *testing.T, root string, fn func()) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	fn()
}

func TestCheckOnFilesCarryingTheVersionChangesNothing(t *testing.T) {
	root := releaseTree(t, "0.6.0")
	before := map[string]string{}
	for rel := range releaseFiles {
		before[rel] = readAll(t, root, rel)
	}
	inDir(t, root, func() {
		if err := run(true, false); err != nil {
			t.Fatalf("-check: %v", err)
		}
	})
	for rel, want := range before {
		if got := readAll(t, root, rel); got != want {
			t.Errorf("%s changed: %q", rel, got)
		}
	}
}

func TestCheckReportsAFileThatLagsTheVersion(t *testing.T) {
	root := releaseTree(t, "0.6.0")
	writeFile(t, root, "build/windows/info.json", withVersion(releaseFiles["build/windows/info.json"], "0.5.0"))
	inDir(t, root, func() {
		err := run(true, false)
		if err == nil || !strings.Contains(err.Error(), "build/windows/info.json") || !strings.Contains(err.Error(), "0.6.0") {
			t.Fatalf("-check = %v", err)
		}
	})
	if got := readAll(t, root, "build/windows/info.json"); !strings.Contains(got, "0.5.0") {
		t.Errorf("-check rewrote the file: %q", got)
	}
}

func TestANewVersionRewritesEveryCopyAndPrependsOneRelease(t *testing.T) {
	root := releaseTree(t, "0.5.0")
	writeFile(t, root, "build/config.yml", withVersion(releaseFiles["build/config.yml"], "0.6.0"))
	today := time.Now().Format(time.DateOnly)
	inDir(t, root, func() {
		if err := run(false, false); err != nil {
			t.Fatal(err)
		}
	})
	// The Windows files carry the version in every match.
	if got := readAll(t, root, "build/windows/wails.exe.manifest"); !strings.Contains(got, `version="0.6.0"`) || strings.Contains(got, "0.5.0") {
		t.Errorf("manifest %q", got)
	}
	if got := readAll(t, root, "build/windows/info.json"); strings.Contains(got, "0.5.0") || strings.Count(got, "0.6.0") != 3 {
		t.Errorf("info.json %q", got)
	}
	if got := readAll(t, root, "build/linux/aur/PKGBUILD"); !strings.Contains(got, "pkgver=0.6.0") {
		t.Errorf("PKGBUILD %q", got)
	}
	if got := readAll(t, root, "build/windows/nsis/wails_tools.nsh"); !strings.Contains(got, `INFO_PRODUCTVERSION "0.6.0"`) {
		t.Errorf("nsh %q", got)
	}

	meta := readAll(t, root, "build/linux/tech.rethunk.Mortar.metainfo.xml")
	if n := strings.Count(meta, "<release "); n != 3 {
		t.Fatalf("metainfo has %d releases, want the new one above two kept: %q", n, meta)
	}
	if want := `<release version="0.6.0" date="` + today + `"`; !strings.Contains(meta, want) {
		t.Errorf("metainfo lacks a dated %s: %q", want, meta)
	}
	if strings.Index(meta, `version="0.6.0"`) > strings.Index(meta, `version="0.5.0"`) {
		t.Errorf("the new release is not above the old one: %q", meta)
	}
	if !strings.Contains(meta, `<release version="0.5.0" date="2026-11-03">`) || !strings.Contains(meta, "Previous.") {
		t.Errorf("the previous release lost its date or notes: %q", meta)
	}
}

func TestRewritingTwiceIsIdempotent(t *testing.T) {
	root := releaseTree(t, "0.5.0")
	writeFile(t, root, "build/config.yml", withVersion(releaseFiles["build/config.yml"], "0.6.0"))
	inDir(t, root, func() {
		if err := run(false, false); err != nil {
			t.Fatal(err)
		}
		once := map[string]string{}
		for rel := range releaseFiles {
			once[rel] = readAll(t, root, rel)
		}
		if err := run(false, false); err != nil {
			t.Fatal(err)
		}
		for rel, want := range once {
			if got := readAll(t, root, rel); got != want {
				t.Errorf("%s changed on the second run: %q", rel, got)
			}
		}
		if n := strings.Count(once["build/linux/tech.rethunk.Mortar.metainfo.xml"], "<release "); n != 3 {
			t.Errorf("metainfo has %d releases after one run, want 3", n)
		}
	})
}
