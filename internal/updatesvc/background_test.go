package updatesvc

import (
	"context"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

func TestBackgroundTick(t *testing.T) {
	type emitted struct {
		name string
		rel  Release
	}
	newTick := func(autoInstall func() bool) (*fake, *Service, func() []emitted) {
		f := &fake{rel: &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Signature: []byte{2}}}}
		s := signedService(t, f)
		s.AutoInstall = autoInstall
		var got []emitted
		emit := func(name string, data any) { got = append(got, emitted{name, data.(Release)}) }
		return f, s, func() []emitted {
			s.backgroundTick(context.Background(), emit)
			return got
		}
	}

	t.Run("auto install off", func(t *testing.T) {
		f, _, tick := newTick(func() bool { return false })
		if got := tick(); f.downloaded || len(got) != 0 {
			t.Fatalf("downloaded %v, emitted %+v", f.downloaded, got)
		}
	})
	t.Run("no update", func(t *testing.T) {
		f, _, tick := newTick(func() bool { return true })
		f.rel = nil
		if got := tick(); f.downloaded || len(got) != 0 {
			t.Fatalf("downloaded %v, emitted %+v", f.downloaded, got)
		}
	})
	t.Run("already staged", func(t *testing.T) {
		f, s, tick := newTick(func() bool { return true })
		if _, err := s.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := s.Install(context.Background()); err != nil {
			t.Fatal(err)
		}
		f.downloaded = false
		if got := tick(); f.downloaded || len(got) != 0 {
			t.Fatalf("downloaded %v, emitted %+v", f.downloaded, got)
		}
	})
	t.Run("auto install on", func(t *testing.T) {
		f, _, tick := newTick(func() bool { return true })
		got := tick()
		if !f.downloaded || len(got) != 1 || got[0].name != StagedEvent || got[0].rel.Version != "1.1.0" || !got[0].rel.Staged {
			t.Fatalf("downloaded %v, emitted %+v", f.downloaded, got)
		}
		if got = tick(); len(got) != 1 {
			t.Fatalf("a second tick emitted again: %+v", got)
		}
	})
}
