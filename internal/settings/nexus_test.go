package settings

import "testing"

func TestNexusDownloadServerValidation(t *testing.T) {
	s, _ := open(t)
	if _, err := s.Update(func(v *Settings) {
		v.NexusSeenDownloadServers = []string{"amsterdam", "nexus_cdn"}
		v.NexusPreferredDownloadServer = "amsterdam"
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(func(v *Settings) { v.NexusPreferredDownloadServer = "unknown" }); err == nil {
		t.Fatal("expected unknown server error")
	}
	if got := s.Get().NexusPreferredDownloadServer; got != "amsterdam" {
		t.Fatalf("unchanged on reject: %q", got)
	}
}

func TestRememberNexusDownloadServers(t *testing.T) {
	s, _ := open(t)
	if _, err := s.Update(func(v *Settings) {
		RememberNexusDownloadServers(v, []string{"a", "b", "a"})
	}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().NexusSeenDownloadServers; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("seen = %v", got)
	}
}

func TestNxmRedirectDefaultsWithPrevious(t *testing.T) {
	s, _ := open(t)
	if _, err := s.Update(func(v *Settings) { v.NxmPrevious = "vortex.desktop" }); err != nil {
		t.Fatal(err)
	}
	if !s.Get().RedirectOtherGames() {
		t.Fatal("expected redirect on when previous handler exists")
	}
}
