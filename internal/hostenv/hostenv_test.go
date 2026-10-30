package hostenv

import (
	"slices"
	"testing"
)

func TestScrubAppImage(t *testing.T) {
	in := []string{
		"APPDIR=/tmp/.mount_x", "APPIMAGE=/home/u/Mortar.AppImage", "OWD=/home/u",
		"LD_LIBRARY_PATH=/tmp/.mount_x/usr/lib:/tmp/.mount_x/usr/lib64:/opt/lib",
		"PATH=/tmp/.mount_x/usr/bin:/usr/bin",
		"GTK_PATH=/tmp/.mount_x//usr/lib/gtk-3.0",
		"HOME=/home/u",
	}
	want := []string{"LD_LIBRARY_PATH=/opt/lib", "PATH=/usr/bin", "HOME=/home/u"}
	if got := scrub(in, "/home/u/Mortar.AppImage", "/tmp/.mount_x"); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := scrub(in, "", "/tmp/.mount_x"); !slices.Equal(got, in) {
		t.Fatalf("outside an AppImage the environment is untouched, got %v", got)
	}
}
