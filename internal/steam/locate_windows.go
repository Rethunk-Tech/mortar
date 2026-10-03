//go:build windows

package steam

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func candidates(string) []string {
	var out []string
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE); err == nil {
		if p, _, err := k.GetStringValue("SteamPath"); err == nil && p != "" {
			out = append(out, filepath.Clean(p))
		}
		_ = k.Close()
	}
	for _, key := range []string{`SOFTWARE\Wow6432Node\Valve\Steam`, `SOFTWARE\Valve\Steam`} {
		if k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.QUERY_VALUE); err == nil {
			if p, _, err := k.GetStringValue("InstallPath"); err == nil && p != "" {
				out = append(out, filepath.Clean(p))
			}
			_ = k.Close()
		}
	}
	if pf := os.Getenv("ProgramFiles(x86)"); pf != "" {
		out = append(out, filepath.Join(pf, "Steam"))
	}
	return out
}
