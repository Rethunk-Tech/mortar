// Command version copies the app version from build/config.yml into the files that cannot read it themselves
// (the Windows resource manifest, info and NSIS installer defines, the Linux metainfo, the AUR PKGBUILD). With -check it
// changes nothing and fails when any copy differs, so the gate catches a version set in only one place. With
// -print it only prints the version, for scripts and workflows.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

type copyOf struct {
	path    string
	pattern *regexp.Regexp // group 1 is the version; every match is rewritten
}

var copies = []copyOf{
	{"build/windows/wails.exe.manifest", regexp.MustCompile(`name="tech\.rethunk\.mortar" version="([^"]*)"`)},
	{"build/windows/info.json", regexp.MustCompile(`"(?:file_version|FileVersion|ProductVersion)": "([^"]*)"`)},
	{"build/linux/tech.rethunk.Mortar.metainfo.xml", regexp.MustCompile(`<release version="([^"]*)" date="[^"]*"`)},
	{"build/linux/aur/PKGBUILD", regexp.MustCompile(`(?m)^pkgver=(.*)$`)},
	{"build/windows/nsis/wails_tools.nsh", regexp.MustCompile(`!define INFO_PRODUCTVERSION "([^"]*)"`)},
}

func main() {
	check := flag.Bool("check", false, "fail when a copy differs instead of rewriting it")
	show := flag.Bool("print", false, "print build/config.yml's version and exit")
	flag.Parse()
	if err := run(*check, *show); err != nil {
		fmt.Fprintln(os.Stderr, "version:", err)
		os.Exit(1)
	}
}

func run(check, show bool) error {
	config, err := fsx.ReadFile("build/config.yml")
	if err != nil {
		return err
	}
	want, err := appversion.FromConfig(config)
	if err != nil {
		return err
	}
	if show {
		fmt.Println(want)
		return nil
	}
	var stale []string
	for _, c := range copies {
		b, err := fsx.ReadFile(c.path)
		if err != nil {
			return err
		}
		if !c.pattern.Match(b) {
			return fmt.Errorf("%s has no version to set", c.path)
		}
		out := setVersion(c, b, want)
		if bytes.Equal(out, b) {
			continue
		}
		if check {
			stale = append(stale, c.path)
			continue
		}
		info, err := os.Stat(c.path)
		if err != nil {
			return err
		}
		if err := fsx.WriteFile(c.path, out, info.Mode().Perm()); err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", c.path, want)
	}
	if len(stale) > 0 {
		return fmt.Errorf("these do not carry build/config.yml's version %s (run `go run ./cmd/version`): %v", want, stale)
	}
	return nil
}

// setVersion rewrites the version in every match, except in the metainfo, where only the first <release> is the
// current one: a new version gets its own <release> dated today above it, so the previous one keeps its date and
// description (build/release/notes.sh fills the new one's).
func setVersion(c copyOf, b []byte, want string) []byte {
	metainfo := c.path == "build/linux/tech.rethunk.Mortar.metainfo.xml"
	return firstOnly(c.pattern, b, metainfo, func(m []byte) []byte {
		sub := c.pattern.FindSubmatchIndex(m)
		if string(m[sub[2]:sub[3]]) == want {
			return m
		}
		if metainfo {
			return fmt.Appendf(nil, "<release version=%q date=%q />\n    %s", want, time.Now().Format(time.DateOnly), m)
		}
		return append(append(append([]byte{}, m[:sub[2]]...), want...), m[sub[3]:]...)
	})
}

func firstOnly(re *regexp.Regexp, b []byte, first bool, fn func([]byte) []byte) []byte {
	done := false
	return re.ReplaceAllFunc(b, func(m []byte) []byte {
		if done {
			return m
		}
		done = first
		return fn(m)
	})
}
