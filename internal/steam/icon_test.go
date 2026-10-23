package steam

import (
	"bytes"
	"image/png"
	"testing"
)

func TestClientIconTakesTheLargestPNGOfSteamsIcon(t *testing.T) {
	if got := (Steam{Root: "testdata/without-ico"}).ClientIcon("1966720"); string(got) != "small jpg" {
		t.Fatalf("without an .ico: %q", got)
	}
	s := Steam{Root: "testdata/with-ico"}
	got, err := png.DecodeConfig(bytes.NewReader(s.ClientIcon("1966720")))
	if err != nil || got.Width != 256 {
		t.Fatalf("got a %d-pixel image, %v; want the 256-pixel PNG", got.Width, err)
	}
	if s.ClientIcon("../1966720") != nil || s.ClientIcon("1") != nil {
		t.Fatal("an icon for an unknown or unsafe id")
	}
}
