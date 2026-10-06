package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestALaunchArmsTheBridgesIntroSkipFromTheSettingOrACrashCheck(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, profiles, lc, "A")
	dir, err := profiles.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), set, profiles)
	g := game.Find(lc)
	l, _ := svc.loaderOf(lc, p.ID)
	request := func() string {
		t.Helper()
		if err := svc.armIntroSkip(g, l, p.ID, dir); err != nil {
			t.Fatal(err)
		}
		b, err := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(bepinex5.IntroSkipFile)))
		if os.IsNotExist(err) {
			return ""
		}
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	if got := request(); got != "" {
		t.Fatalf("off by default, got %q", got)
	}
	if _, err := set.Update(func(s *settings.Settings) {
		gp := s.GamePrefs(lc)
		gp.SkipIntro = true
		s.Games = map[string]*settings.GameSettings{lc: &gp}
	}); err != nil {
		t.Fatal(err)
	}
	if got := request(); got != "intro" {
		t.Fatalf("the setting skips the animations only, got %q", got)
	}
	if _, err := profiles.SetOverride(lc, p.ID, "skipIntro", "false"); err != nil {
		t.Fatal(err)
	}
	if got := request(); got != "" {
		t.Fatalf("the profile's override wins, got %q", got)
	}
	withdraw, err := svc.launchToMenu(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer withdraw()
	if got := request(); got != "menu" {
		t.Fatalf("a crash check goes to the menu whatever the setting, got %q", got)
	}
	if got := request(); got != "" {
		t.Fatalf("the crash check's request is for one launch, got %q", got)
	}
}
