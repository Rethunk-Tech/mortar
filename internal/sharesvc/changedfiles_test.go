package sharesvc

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

const changedGame = "sims4"

// changedSender is a Sims 4 profile holding a mod's saved settings and a file it created, which no record names, and a
// copy held under changed/.
func changedSender(t *testing.T, s *Service) (id string) {
	t.Helper()
	p, err := s.d.Profiles.Create(changedGame, "Sender")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.d.Profiles.ProfileDir(changedGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rel, body := range map[string]string{
		"Mods/mc/mc_settings.cfg":           "player=3",
		"Mods/mc/mc_state.dat":              "created during play",
		"changed/Mods/old/old_settings.cfg": "held",
	} {
		to := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return p.ID
}

func readIn(t *testing.T, s *Service, id, rel string) (string, bool) {
	t.Helper()
	dir, err := s.d.Profiles.ProfileDir(changedGame, id)
	if err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	return string(b), err == nil
}

// A paired LAN send and a sync carry the changed copies like configs, and the receiving profile keeps them through
// every later sync of its packages.
func TestAPairedSendCarriesChangedCopiesAndTheReceiverKeepsThem(t *testing.T) {
	s, _ := newService(t, true)
	src := changedSender(t, s)
	payload, _, err := s.ExportBytes(changedGame, src, share.OwnInclude())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ImportData(t.Context(), changedGame, base64.RawStdEncoding.EncodeToString(payload))
	if err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{
		"Mods/mc/mc_settings.cfg": "player=3", "Mods/mc/mc_state.dat": "created during play", "changed/Mods/old/old_settings.cfg": "held",
	} {
		if b, ok := readIn(t, s, got.Profile.ID, rel); !ok || b != want {
			t.Errorf("%s on the receiver = %q (%v), want %q", rel, b, ok, want)
		}
	}
	for range 2 {
		if err := s.d.Profiles.SyncPackages(changedGame, got.Profile.ID); err != nil {
			t.Fatal(err)
		}
		if b, _ := readIn(t, s, got.Profile.ID, "Mods/mc/mc_settings.cfg"); b != "player=3" {
			t.Fatalf("a sync on the receiver changed the copy to %q", b)
		}
	}
}

// A share with other people carries them exactly when it carries configs.
func TestAShareCarriesChangedCopiesOnlyWhenItCarriesConfigs(t *testing.T) {
	s, _ := newService(t, true)
	src := changedSender(t, s)
	for name, c := range map[string]struct {
		inc  share.Include
		want bool
	}{
		"with configs":    {share.Include{ConfigFiles: true}, true},
		"without configs": {share.Include{}, false},
		"default":         {share.DefaultInclude(), true},
	} {
		payload, _, err := s.ExportBytes(changedGame, src, c.inc)
		if err != nil {
			t.Fatal(err)
		}
		pv, err := share.ReadBytes(payload)
		if err != nil {
			t.Fatal(err)
		}
		if (len(pv.ChangedFiles) == 3) != c.want || (!c.want && len(pv.ChangedFiles) != 0) {
			t.Errorf("%s: %d changed files", name, len(pv.ChangedFiles))
		}
	}
}

// Changed copies from another computer are untrusted: a path outside the profile's content folder and changed/, or
// one that climbs out, is refused whole, and nothing is written.
func TestChangedCopiesFromAnotherComputerCannotLeaveTheProfile(t *testing.T) {
	s, _ := newService(t, true)
	p, err := s.d.Profiles.Create(changedGame, "Receiver")
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../x", "Mods/../../x", "/abs", "C:/x", `Mods\x`, "other/x", "Mods", "profile.json", "Mods//x"} {
		err := s.d.Profiles.WriteChangedFiles(changedGame, p.ID, map[string][]byte{"Mods/ok.cfg": []byte("ok"), rel: []byte("bad")}, true)
		if err == nil {
			t.Errorf("%q was accepted", rel)
		}
	}
	if _, ok := readIn(t, s, p.ID, "Mods/ok.cfg"); ok {
		t.Error("a refused set was partly written")
	}
}

func TestAChangedCopyCannotBeWrittenThroughALink(t *testing.T) {
	s, _ := newService(t, true)
	p, err := s.d.Profiles.Create(changedGame, "Receiver")
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := s.d.Profiles.ProfileDir(changedGame, p.ID)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Mods"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "Mods", "mc")); err != nil {
		t.Skip("links are not available")
	}
	if err := s.d.Profiles.WriteChangedFiles(changedGame, p.ID, map[string][]byte{"Mods/mc/x.cfg": []byte("bad")}, true); err == nil {
		t.Fatal("a write through a link was accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "x.cfg")); err == nil {
		t.Fatal("a file landed outside the profile")
	}
}
