package fonts

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const fontsKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

// List is the installed font families, from the machine's fonts registry key and the user's own (fonts installed for
// one user live under HKCU).
func List() ([]string, error) {
	var names []string
	var errs []error
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		got, err := valueNames(root)
		names = append(names, got...)
		errs = append(errs, err)
	}
	if len(names) == 0 {
		return nil, errors.Join(errs...)
	}
	return ParseWindowsNames(names), nil
}

func valueNames(root registry.Key) ([]string, error) {
	k, err := registry.OpenKey(root, fontsKey, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer func() { _ = k.Close() }()
	return k.ReadValueNames(-1)
}
