//go:build !windows

package devsvc

// holderNames is Windows only: elsewhere an open file does not stop a folder from moving to the trash.
func holderNames(string) ([]string, bool) { return nil, false }
