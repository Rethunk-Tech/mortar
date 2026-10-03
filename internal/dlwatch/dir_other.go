//go:build !windows

package dlwatch

func windowsDownloads() string {
	return unixDownloads()
}
