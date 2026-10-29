//go:build windows

package gamestore

import (
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"golang.org/x/sys/windows/registry"
)

// appxPackages is the per-user AppModel repository: one subkey per installed package, named by its full name
// (Name_Version_Arch_ResourceId_PublisherId), whose PackageRootFolder is what Get-AppxPackage reports as InstallLocation.
// Reading it directly spares a PowerShell start (about a second each) for every game the catalog lists on Xbox.
const appxPackages = `Software\Classes\Local Settings\Software\Microsoft\Windows\CurrentVersion\AppModel\Repository\Packages`

func appxRoot(pkg string) string {
	k, err := registry.OpenKey(registry.CURRENT_USER, appxPackages, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return ""
	}
	for _, full := range names {
		if !strings.HasPrefix(full, pkg+"_") {
			continue
		}
		sub, err := registry.OpenKey(k, full, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		dir, _, err := sub.GetStringValue("PackageRootFolder")
		_ = sub.Close()
		if err == nil && safefs.IsDir(dir) {
			return dir
		}
	}
	return ""
}
