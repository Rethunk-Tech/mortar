package bepinex5

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// unityLibsURL is where BepInEx 6 fetches a Unity version's base libraries on an IL2CPP game's first start.
var unityLibsURL = "https://unity.bepinex.dev/libraries/"

// unityVersionPattern is the engine version Unity writes near the start of globalgamemanagers, as 2022.3.21f1.
var unityVersionPattern = regexp.MustCompile(`(\d{4}\.\d+\.\d+)[abfp]\d+`)

// unityHeaderBytes is how far into globalgamemanagers the version is looked for; it sits in the file's header.
const unityHeaderBytes = 1 << 16

// unityVersion is the Unity version of the game in installDir (2022.3.21), from its data folder, or "".
func unityVersion(installDir string) string {
	dirs, _ := filepath.Glob(filepath.Join(installDir, "*_Data"))
	for _, dir := range dirs {
		f, err := fsx.Open(filepath.Join(dir, "globalgamemanagers"))
		if err != nil {
			continue
		}
		head, _ := io.ReadAll(io.LimitReader(f, unityHeaderBytes))
		_ = f.Close()
		if m := unityVersionPattern.FindSubmatch(head); m != nil {
			return string(m[1])
		}
	}
	return ""
}

// placeUnityLibs puts the game's Unity base libraries in the profile's BepInEx/unity-libs, from a copy Mortar keeps,
// so an IL2CPP game's first start needs no network. When they cannot be had, BepInEx fetches them itself at launch.
func placeUnityLibs(ctx context.Context, installDir, profileDir string) error {
	ver := unityVersion(installDir)
	if ver == "" {
		return nil
	}
	base, err := datadir.Dir()
	if err != nil {
		return err
	}
	kept := filepath.Join(base, "cache", "unity-libs", ver+".zip")
	if _, err := os.Stat(kept); err != nil {
		if err := downloadUnityLibs(ctx, ver, kept); err != nil {
			return err
		}
	}
	b, err := fsx.ReadFile(kept)
	if err != nil {
		return err
	}
	return writeIn(filepath.Join(profileDir, "BepInEx", "unity-libs", ver+".zip"), b)
}

func writeIn(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteFileAtomic(path, b, 0o600)
}

func downloadUnityLibs(ctx context.Context, ver, dst string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, unityLibsURL+ver+".zip", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unity libraries %s: %s", ver, resp.Status)
	}
	b, err := fsx.ReadCapped(resp.Body, 256<<20)
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(b, []byte("PK")) {
		return fmt.Errorf("unity libraries %s: not a zip", ver)
	}
	log.Printf("bepinex: kept the Unity %s base libraries (%d KB)", ver, len(b)>>10)
	return writeIn(dst, b)
}
