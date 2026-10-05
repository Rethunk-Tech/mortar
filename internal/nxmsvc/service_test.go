package nxmsvc

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type fakeHandler struct {
	owner    nxm.Owner
	registry []string
	failNext bool
}

func (f *fakeHandler) Owner(string) (nxm.Owner, error) { return f.owner, nil }

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

func (f *fakeHandler) Restore(previous map[string]string) error {
	f.owner = nxm.Owner{ID: previous["nxm"], Name: previous["nxm"], Mine: false}
	f.registry = append(f.registry, "restore:"+previous["nxm"])
	return nil
}

func (f *fakeHandler) Release(schemes []string, _ map[string]string) error {
	f.registry = append(f.registry, "release:"+strings.Join(schemes, ","))
	return nil
}

func (f *fakeHandler) ForwardOther(link, previous string) error {
	f.registry = append(f.registry, "forward:"+previous+":"+link)
	return nil
}

func newService(t *testing.T, h *fakeHandler) *Service {
	t.Helper()
	testfs.DataHome(t)
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
	if got := s.store.Get(); !got.NxmHandled || got.NxmPreviousHandlers["nxm"] != "vortex.desktop" || got.NxmPreviousName != "Vortex" || !got.NxmAsked {
		t.Fatalf("settings after Enable: %+v", got)
	}
	if name, _ := s.Owner(); name != "" {
		t.Errorf("Owner names %q once Mortar owns the scheme", name)
	}
	if err := s.Enable(); err != nil || s.store.Get().NxmPreviousHandlers["nxm"] != "vortex.desktop" {
		t.Fatalf("a second Enable lost the previous owner: %+v, %v", s.store.Get(), err)
	}
	if err := s.Disable(); err != nil {
		t.Fatal(err)
	}
	if got := s.store.Get(); got.NxmHandled || len(got.NxmPreviousHandlers) != 0 || h.owner.ID != "vortex.desktop" {
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
	if got := s.store.Get(); got.NxmHandled || len(got.NxmPreviousHandlers) != 0 || h.owner.ID != "vortex.desktop" {
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

func TestReceiveForwardsOtherGameLinks(t *testing.T) {
	h := &fakeHandler{}
	s := newService(t, h)
	if _, err := s.store.Update(func(v *settings.Settings) {
		v.NxmPreviousHandlers = map[string]string{"nxm": "vortex.desktop"}
		on := true
		v.NxmRedirectOtherGames = &on
	}); err != nil {
		t.Fatal(err)
	}
	link := "nxm://skyrim/mods/5/files/9?key=k&expires=1000600&user_id=42"
	if !s.Receive([]string{link}) {
		t.Fatal("expected nxm link")
	}
	if len(s.Inbox().Rejections) != 0 || len(s.Inbox().Arrivals) != 0 {
		t.Fatalf("other-game link should forward, not inbox: %+v", s.Inbox())
	}
	if len(h.registry) != 1 || h.registry[0] != "forward:vortex.desktop:"+link {
		t.Fatalf("registry %q", h.registry)
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
	link := "nxm://stardewvalley/mods/5/files/9?key=k&expires=1000600&user_id=42"
	s.Receive([]string{link})
	s.Ignore(s.Inbox().Arrivals[0].ID)
	if len(s.Inbox().Arrivals) != 0 || len(s.Assigned) != 0 {
		t.Error("ignored link survived")
	}
	s.Receive([]string{link})
	if len(s.Inbox().Arrivals) != 1 {
		t.Error("ignored link was not accepted when it arrived again")
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

func TestReceiveTakesALinkOnceUntilItExpires(t *testing.T) {
	s := newService(t, &fakeHandler{})
	start := s.now()
	link := "nxm://stardewvalley/mods/5/files/9?key=k&expires=99999999999&user_id=42"
	s.Receive([]string{link})
	s.Receive([]string{link})
	if n := len(s.Inbox().Arrivals); n != 1 {
		t.Fatalf("%d arrivals from one link sent twice", n)
	}
	s.now = func() time.Time { return start.Add(duplicateWindow + time.Second) }
	s.Receive([]string{link})
	if n := len(s.Inbox().Arrivals); n != 1 {
		t.Fatalf("an unexpired link was taken again: %d arrivals", n)
	}
	s.now = func() time.Time { return time.Unix(99999999999, 0).Add(time.Second) }
	s.Receive([]string{link})
	if in := s.Inbox(); len(in.Rejections) != 1 || in.Rejections[0].Reason != nxm.ReasonExpired {
		t.Fatalf("an expired link sent again was not refused as expired: %+v", in)
	}
}

func TestReceiveQueuesAThunderstoreLink(t *testing.T) {
	source.SetHandleLinks(map[string]bool{"thunderstore": true})
	t.Cleanup(func() { source.SetHandleLinks(nil) })
	s := newService(t, &fakeHandler{})
	if !s.Receive([]string{"ror2mm://v1/install/thunderstore.io/Me/Mod/1.2.3/"}) {
		t.Fatal("the ror2mm link was not taken")
	}
	in := s.Inbox()
	if len(in.Arrivals) != 1 || in.Arrivals[0].Package != "Me-Mod" || in.Arrivals[0].Version != "1.2.3" || !slices.Contains(in.Arrivals[0].Games, "lethal-company") || len(in.Rejections) != 0 {
		t.Fatalf("inbox %+v", in)
	}
	if err := s.Assign(in.Arrivals[0].ID, "riskofrain2", "p1"); err != nil {
		t.Fatal(err)
	}
	if got := <-s.Assigned; got.Package != "Me-Mod" || got.Version != "1.2.3" || got.Game != "riskofrain2" {
		t.Fatalf("assigned %+v", got)
	}
}

func TestOptInSourceLinksAreEnabledAndReleasedPerSource(t *testing.T) {
	h := &fakeHandler{owner: nxm.Owner{ID: "other.desktop", Name: "Other"}}
	s := newService(t, h)
	t.Cleanup(func() { source.SetHandleLink("thunderstore", false) })
	if err := s.EnableSource("thunderstore"); err != nil {
		t.Fatal(err)
	}
	got := s.store.Get()
	if got.ThunderstoreHandleLinks == nil || !*got.ThunderstoreHandleLinks || got.NxmPreviousHandlers["ror2mm"] != "other.desktop" {
		t.Fatalf("after EnableSource: %+v", got)
	}
	if !slices.Contains(source.Schemes(), "ror2mm") {
		t.Fatal("ror2mm is not claimed after enabling it")
	}
	if err := s.DisableSource("thunderstore"); err != nil {
		t.Fatal(err)
	}
	got = s.store.Get()
	if *got.ThunderstoreHandleLinks || got.NxmPreviousHandlers["ror2mm"] != "" || got.NxmPreviousHandlers["nxm"] != "other.desktop" {
		t.Fatalf("after DisableSource: %+v", got)
	}
	if last := h.registry[len(h.registry)-1]; last != "release:ror2mm" || slices.Contains(source.Schemes(), "ror2mm") {
		t.Fatalf("registry %q, schemes %v", h.registry, source.Schemes())
	}
}
