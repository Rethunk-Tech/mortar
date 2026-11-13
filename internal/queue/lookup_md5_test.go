package queue

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestLookupMD5FailsClosedWhenFilesAPIFails(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.d.VerifyNexusMD5 = func() bool { return true }
	f.files = func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "502 bad gateway", http.StatusBadGateway)
	}
	f.start()
	named := req(10)
	named.FileName = "a-1.0.zip"
	f.add(named)
	st := f.wait("files api failed", f.item(StateFailed))
	it := st.Items[0]
	if it.ErrorKind != FailMD5List {
		t.Fatalf("kind %q error %q", it.ErrorKind, it.Error)
	}
	msg := strings.ToLower(it.Error)
	if !strings.Contains(msg, "nexus") || !strings.Contains(msg, "file list") {
		t.Fatalf("error should name the Nexus file-list API failure, got %q", it.Error)
	}
}

func TestLookupMD5APIFailureIsATranslatableUserError(t *testing.T) {
	t.Parallel()
	apiErr := errors.New("502 bad gateway")
	_, err := lookupMD5(context.Background(), failingFiles{err: apiErr}, nexusmods.Title{Domain: "stardewvalley"}, Item{ModID: 10, FileID: 20})
	if err == nil {
		t.Fatal("lookupMD5: Files API failure must fail the download")
	}
	if !errors.Is(err, apiErr) {
		t.Fatalf("lookupMD5: %v", err)
	}
	if failureKind(err) != FailMD5List {
		t.Fatalf("lookupMD5 kind %q, want %q", failureKind(err), FailMD5List)
	}
	var ue *usererr.Error
	if !errors.As(err, &ue) {
		t.Fatalf("lookupMD5 should wrap a translatable usererr, got %T %v", err, err)
	}
}

func TestInstallAnywaySkipsVerificationForThatItem(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.s.d.VerifyNexusMD5 = func() bool { return true }
	f.files = func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "502 bad gateway", http.StatusBadGateway)
	}
	f.start()
	named := req(10)
	named.FileName = "a-1.0.zip"
	f.add(named)
	st := f.wait("files api failed", f.item(StateFailed))
	if err := f.s.InstallAnyway(st.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	done := f.wait("installed anyway", f.item(StateDone))
	if !done.Items[0].Unverified {
		t.Fatal("install anyway should mark the item unverified")
	}

	other := &Service{
		d:     Deps{VerifyNexusMD5: func() bool { return true }},
		items: []*Item{{ID: "b", State: StateQueued, ModID: 10, FileID: 20}},
	}
	if _, err := other.wantedMD5(context.Background(), failingFiles{err: errors.New("502")}, nexusmods.Title{Domain: "stardewvalley"}, Item{ID: "b"}); err == nil {
		t.Fatal("other items must still fail closed")
	}
}

type failingFiles struct{ err error }

func (f failingFiles) Files(context.Context, nexusmods.Title, int) ([]nexusmods.File, error) {
	return nil, f.err
}
