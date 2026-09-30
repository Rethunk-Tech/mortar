package updatesvc

import (
	"context"
	"errors"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fake struct {
	cfg        *updater.Config
	rel        *updater.Release
	downloaded bool
}

func (f *fake) Init(cfg updater.Config) error { f.cfg = &cfg; return nil }
func (f *fake) Check(context.Context) (*updater.Release, error) {
	return f.rel, nil
}
func (f *fake) DownloadAndInstall(context.Context) error { f.downloaded = true; return nil }
func (f *fake) Restart(context.Context) error            { return nil }

func TestDevBuildsAndVersionsNeverCheck(t *testing.T) {
	for _, c := range []struct {
		version    string
		production bool
	}{{"1.0.0", false}, {"1.1.0-dev", true}} {
		f := &fake{}
		s := &Service{}
		err := configure(s, f, c.version, nil, c.production)
		if err != nil || s.Info().Off != "dev" || f.cfg != nil {
			t.Fatalf("%+v: off %q, configured %v, %v", c, s.Info().Off, f.cfg != nil, err)
		}
		if _, err := s.Check(context.Background()); !errors.Is(err, errOff) {
			t.Fatalf("%+v: check %v", c, err)
		}
	}
}

func TestOnlyASignedReleaseInstalls(t *testing.T) {
	f := &fake{rel: &updater.Release{Version: "1.1.0", Verification: &updater.Verification{Digest: []byte{1}}}}
	s := &Service{}
	err := configure(s, f, "1.0.0", []byte("key"), true)
	if err != nil || f.cfg == nil || f.cfg.CurrentVersion != "1.0.0" || string(f.cfg.PublicKey) != "key" {
		t.Fatalf("configured %+v, %v", f.cfg, err)
	}
	if _, err := s.Check(context.Background()); !errors.Is(err, errUnsigned) {
		t.Fatalf("unsigned: %v", err)
	}
	if err := s.Install(context.Background()); !errors.Is(err, errNone) || f.downloaded {
		t.Fatalf("install after unsigned: %v, downloaded %v", err, f.downloaded)
	}
	f.rel.Verification.Signature = []byte{2}
	if rel, err := s.Check(context.Background()); err != nil || rel.Version != "1.1.0" {
		t.Fatalf("signed: %+v, %v", rel, err)
	}
	if err := s.Install(context.Background()); err != nil || !f.downloaded {
		t.Fatalf("install: %v, downloaded %v", err, f.downloaded)
	}
}
