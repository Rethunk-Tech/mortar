//go:build windows

package steam

import "golang.org/x/sys/windows/registry"

func candidates(string) []string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer func() { _ = k.Close() }()
	p, _, err := k.GetStringValue("SteamPath")
	if err != nil {
		return nil
	}
	return []string{p}
}
