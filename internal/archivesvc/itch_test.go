package archivesvc

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestItchPageIsRebuiltFromTheNameAndAFileIsRecordedUnderIt(t *testing.T) {
	var got profile.Source
	s := NewService(Deps{Install: func(_ context.Context, _, _, _ string, src profile.Source) (profile.InstallResult, error) {
		got = src
		return profile.InstallResult{}, nil
	}})
	page, err := s.ItchPage(" https://Someone.itch.io/cool-mod?utm=x ")
	if err != nil || page.ID != "someone/cool-mod" || page.URL != "https://someone.itch.io/cool-mod" {
		t.Fatalf("page = %+v, %v", page, err)
	}
	if _, err := s.ItchPage("https://evil.test/someone.itch.io/x"); err == nil {
		t.Fatal("another site's address must be refused")
	}
	if _, err := s.InstallItchDownload(t.Context(), "sims4", "p", "a.zip", page.ID); err != nil {
		t.Fatal(err)
	}
	if got.Kind != profile.KindItch || got.Name != "someone/cool-mod" {
		t.Fatalf("source = %+v", got)
	}
	if _, err := s.InstallItchDownload(t.Context(), "sims4", "p", "a.zip", "../x"); err == nil {
		t.Fatal("a page name that is not user/game must be refused")
	}
}
