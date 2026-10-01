package nexus

import "testing"

func TestPickDownloadLink(t *testing.T) {
	links := []Link{
		{Name: "Nexus CDN", ShortName: "nexus_cdn", URI: "https://a.example/1"},
		{Name: "Amsterdam", ShortName: "amsterdam", URI: "https://b.example/2"},
	}
	got, ok := PickDownloadLink(links, "")
	if !ok || got.URI != links[0].URI {
		t.Fatalf("automatic = %+v, %v", got, ok)
	}
	got, ok = PickDownloadLink(links, "amsterdam")
	if !ok || got.URI != links[1].URI {
		t.Fatalf("preferred = %+v, %v", got, ok)
	}
	got, ok = PickDownloadLink(links, "missing")
	if !ok || got.URI != links[0].URI {
		t.Fatalf("unknown preferred falls back to first: %+v, %v", got, ok)
	}
	if _, ok := PickDownloadLink(nil, ""); ok {
		t.Fatal("empty list")
	}
}
