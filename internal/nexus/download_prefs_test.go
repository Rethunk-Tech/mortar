package nexus

import "testing"

func TestApplyDownloadPreferences(t *testing.T) {
	links := []Link{{ShortName: "cdn", URI: "u1"}, {ShortName: "paris", URI: "u2"}}
	var remembered []string
	c := New("1")
	c.Servers = DownloadServers{
		Remember:  func(names []string) { remembered = names },
		Preferred: func() string { return "paris" },
	}
	got := c.WithKey("k").applyDownloadPreferences(links)
	if got[0].URI != "u2" || links[0].URI != "u1" {
		t.Fatalf("got %+v, want the preferred mirror first without touching the input", got)
	}
	if len(remembered) != 2 || remembered[0] != "cdn" || remembered[1] != "paris" {
		t.Fatalf("remembered = %v", remembered)
	}
	if got := New("1").applyDownloadPreferences(links); got[0].URI != "u1" {
		t.Fatalf("without settings the order stays: %+v", got)
	}
}
