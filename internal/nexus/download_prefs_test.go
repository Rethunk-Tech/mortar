package nexus

import (
	"testing"

	"github.com/Rethunk-Tech/go-nexusmods"
)

func TestDownloadServersOrder(t *testing.T) {
	links := []nexusmods.Link{{ShortName: "cdn", URI: "u1"}, {ShortName: "paris", URI: "u2"}}
	var remembered []string
	s := DownloadServers{
		Remember:  func(names []string) { remembered = names },
		Preferred: func() string { return "paris" },
	}
	got := s.Order(links)
	if got[0].URI != "u2" || links[0].URI != "u1" {
		t.Fatalf("got %+v, want the preferred mirror first without touching the input", got)
	}
	if len(remembered) != 2 || remembered[0] != "cdn" || remembered[1] != "paris" {
		t.Fatalf("remembered = %v", remembered)
	}
	if got := (DownloadServers{}).Order(links); got[0].URI != "u1" {
		t.Fatalf("without settings the order stays: %+v", got)
	}
}
