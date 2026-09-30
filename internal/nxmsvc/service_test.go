package nxmsvc

import (
	"errors"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/nxm"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

type fakeHandler struct {
	owner    nxm.Owner
	registry []string
	failNext bool
}

func (f *fakeHandler) Owner() (nxm.Owner, error) { return f.owner, nil }

func (f *fakeHandler) Register() error {
	if f.failNext {
		return errors.New("boom")
	}
	f.owner = nxm.Owner{ID: "tech.rethunk.Mortar.desktop", Name: "Mortar", Mine: true}
	f.registry = append(f.registry, "register")
	return nil
}

func (f *fakeHandler) RegisterLinks() error {
	f.registry = append(f.registry, "links")
	return nil
}

func (f *fakeHandler) Restore(previous string) error {
	f.owner = nxm.Owner{ID: previous, Name: previous, Mine: false}
	f.registry = append(f.registry, "restore:"+previous)
	return nil
}

func newService(t *testing.T, h *fakeHandler) *Service {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(v *settings.Settings) { v.NexusUserID = 42 }); err != nil {
		t.Fatal(err)
	}
	s := NewService(store, h)
	s.now = func() time.Time { return time.Unix(1_000_000, 0) }
	return s
}

func TestEnableRecordsPreviousOwnerAndDisableRestoresIt(t *testing.T) {
	h := &fakeHandler{owner: nxm.Owner{ID: "vortex.desktop", Name: "Vortex"}}
	s := newService(t, h)
	if name, err := s.Owner(); err != nil || name != "Vortex" {
		t.Fatalf("owner %q, %v", name, err)
	}
	if err := s.Enable(); err != nil {
		t.Fatal(err)
	}
	if got := s.store.Get(); !got.NxmHandled || got.NxmPrevious != "vortex.desktop" || !got.NxmAsked {
		t.Fatalf("settings after Enable: %+v", got)
	}
	if name, _ := s.Owner(); name != "" {
		t.Errorf("Owner names %q once Mortar owns the scheme", name)
	}
	if err := s.Enable(); err != nil || s.store.Get().NxmPrevious != "vortex.desktop" {
		t.Fatalf("a second Enable lost the previous owner: %+v, %v", s.store.Get(), err)
	}
	if err := s.Disable(); err != nil {
		t.Fatal(err)
	}
	if got := s.store.Get(); got.NxmHandled || got.NxmPrevious != "" || h.owner.ID != "vortex.desktop" {
		t.Fatalf("after Disable: %+v, owner %+v", got, h.owner)
	}
}

func TestReleaseLinksRestoresTheRecordedHandler(t *testing.T) {
	h := &fakeHandler{owner: nxm.Owner{ID: "vortex.desktop", Name: "Vortex"}}
	s := newService(t, h)
	if err := s.Enable(); err != nil {
		t.Fatal(err)
	}
	if err := ReleaseLinks(s.store, h); err != nil {
		t.Fatal(err)
	}
	if got := s.store.Get(); got.NxmHandled || got.NxmPrevious != "" || h.owner.ID != "vortex.desktop" {
		t.Fatalf("after ReleaseLinks: %+v, owner %+v", got, h.owner)
	}
	if len(h.registry) < 2 || h.registry[len(h.registry)-1] != "restore:vortex.desktop" {
		t.Fatalf("registry calls %q", h.registry)
	}
}

func TestReleaseLinksRemovesTheKeyWhenNoneWasRecorded(t *testing.T) {
	h := &fakeHandler{}
	s := newService(t, h)
	if err := s.Enable(); err != nil {
		t.Fatal(err)
	}
	if err := ReleaseLinks(s.store, h); err != nil {
		t.Fatal(err)
	}
	if h.owner.ID != "" || s.store.Get().NxmHandled {
		t.Fatalf("after ReleaseLinks with no previous: owner %+v, settings %+v", h.owner, s.store.Get())
	}
	if h.registry[len(h.registry)-1] != "restore:" {
		t.Fatalf("registry calls %q", h.registry)
	}
}

func TestFailedRegisterChangesNothing(t *testing.T) {
	h := &fakeHandler{failNext: true}
	s := newService(t, h)
	if err := s.Enable(); err == nil || s.store.Get().NxmHandled {
		t.Fatalf("err %v, settings %+v", err, s.store.Get())
	}
}

func TestReceiveQueuesAcceptedAndReportsRefused(t *testing.T) {
	s := newService(t, &fakeHandler{})
	good := "nxm://stardewvalley/mods/5/files/9?key=k&expires=1000600&user_id=42"
	bad := "nxm://stardewvalley/mods/5/files/9?key=k&expires=1000600&user_id=1"
	if !s.Receive([]string{"--flag", good, bad}) || s.Receive([]string{"mortar://x"}) {
		t.Fatal("Receive misreported whether a link was present")
	}
	in := s.Inbox()
	if len(in.Arrivals) != 1 || in.Arrivals[0].Link.ModID != 5 || len(in.Rejections) != 1 || in.Rejections[0].Reason != nxm.ReasonUser {
		t.Fatalf("inbox %+v", in)
	}
	if again := s.Inbox(); len(again.Rejections) != 0 || len(again.Arrivals) != 1 {
		t.Fatalf("refusals are handed over once: %+v", again)
	}
	if err := s.Assign(in.Arrivals[0].ID, "stardew", "p1"); err != nil {
		t.Fatal(err)
	}
	got := <-s.Assigned
	if got.Profile != "p1" || got.Link.FileID != 9 {
		t.Fatalf("assigned %+v", got)
	}
	if err := s.Assign(in.Arrivals[0].ID, "stardew", "p1"); err == nil {
		t.Error("a link was assigned twice")
	}
	if len(s.Inbox().Arrivals) != 0 {
		t.Error("assigned arrival still waiting")
	}
}

func TestIgnoreDropsTheArrival(t *testing.T) {
	s := newService(t, &fakeHandler{})
	s.Receive([]string{"nxm://stardewvalley/mods/5/files/9?key=k&expires=1000600&user_id=42"})
	s.Ignore(s.Inbox().Arrivals[0].ID)
	if len(s.Inbox().Arrivals) != 0 || len(s.Assigned) != 0 {
		t.Error("ignored link survived")
	}
}

func TestReceiveRoutesALinkTheQueueWaitsFor(t *testing.T) {
	s := newService(t, &fakeHandler{})
	s.Route = func(l nxm.Link) bool { return l.FileID == 9 }
	s.Receive([]string{
		"nxm://stardewvalley/mods/5/files/9?key=k&expires=1000600&user_id=42",
		"nxm://stardewvalley/mods/5/files/8?key=k&expires=1000600&user_id=42",
	})
	in := s.Inbox()
	if len(in.Arrivals) != 1 || in.Arrivals[0].Link.FileID != 8 {
		t.Fatalf("a routed link became an arrival, or the other did not: %+v", in)
	}
}

func TestAFullQueueKeepsTheArrival(t *testing.T) {
	s := newService(t, &fakeHandler{})
	s.Assigned = make(chan Assignment)
	s.Receive([]string{"nxm://stardewvalley/mods/5/files/9?key=k&expires=1000600&user_id=42"})
	id := s.Inbox().Arrivals[0].ID
	if err := s.Assign(id, "stardew", "p1"); err == nil {
		t.Fatal("assigned into a full queue")
	}
	s.Assigned = make(chan Assignment, 1)
	if err := s.Assign(id, "stardew", "p1"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(s.Inbox().Arrivals) != 0 || len(s.Assigned) != 1 {
		t.Error("retried arrival not handed over")
	}
}
