package syncsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
)

type fakeShares struct {
	previewErr error
	replaced   []string
	exported   share.Include
}

func (f *fakeShares) ExportBytes(_, _ string, inc share.Include) ([]byte, []string, error) {
	f.exported = inc
	return nil, nil, nil
}

func (f *fakeShares) PreviewPayload(context.Context, string, string, []byte) (sharesvc.Preview, error) {
	return sharesvc.Preview{Session: "s1"}, f.previewErr
}

func (f *fakeShares) Replace(_ context.Context, _, session, _ string, _ []string) (sharesvc.Result, error) {
	f.replaced = append(f.replaced, session)
	return sharesvc.Result{}, nil
}

type fakeProfiles struct{ all []profile.Profile }

func (f fakeProfiles) List(string) ([]profile.Profile, error)       { return f.all, nil }
func (fakeProfiles) Create(string, string) (profile.Profile, error) { return profile.Profile{}, nil }
func (fakeProfiles) Delete(string, string) error                    { return nil }

func TestAppSourceAppliesOnlyAPayloadItCouldRead(t *testing.T) {
	sh := &fakeShares{previewErr: errors.New("damaged payload")}
	a := &AppSource{profiles: fakeProfiles{}, shares: sh}
	if err := a.Apply(t.Context(), "stardew", "p1", []byte("x")); err == nil || len(sh.replaced) != 0 {
		t.Fatalf("a payload that failed its preview replaced the profile: %v %v", err, sh.replaced)
	}
	sh.previewErr = nil
	if err := a.Apply(t.Context(), "stardew", "p1", []byte("x")); err != nil || len(sh.replaced) != 1 || sh.replaced[0] != "s1" {
		t.Fatalf("apply %v, replaced %v", err, sh.replaced)
	}
}

func TestAppSourceNeverOffersADamagedProfile(t *testing.T) {
	a := &AppSource{profiles: fakeProfiles{all: []profile.Profile{{ID: "ok", Name: "Ok"}, {ID: "bad", Error: "unreadable"}}}}
	refs, err := a.Profiles("stardew")
	if err != nil || len(refs) != 1 || refs[0].ID != "ok" {
		t.Fatalf("refs %+v %v", refs, err)
	}
}

func TestAppSourceExportsSwitchedOffMods(t *testing.T) {
	sh := &fakeShares{}
	if _, err := (&AppSource{profiles: fakeProfiles{}, shares: sh}).Export("lethal-company", "p1"); err != nil {
		t.Fatal(err)
	}
	if !sh.exported.DisabledMods || !sh.exported.ConfigFiles {
		t.Fatalf("sync export include %+v leaves mods or configs out", sh.exported)
	}
}
