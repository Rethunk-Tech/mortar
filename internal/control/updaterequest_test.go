package control

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/queue"
)

func TestCurseForgeSwitchUpdatesQueueAsPackagesOnlyWhenNamedAndForbiddenOnesNever(t *testing.T) {
	t.Parallel()
	sw := problems.Update{
		Key: "nexus-20619-123175", ID: "smapi:Example.Bush", Installed: "1.0.0", Version: "1.0.1",
		Package: "998265", PackageSource: "curseforge", PackageVersion: "777", Switch: true, FromSource: "Nexus Mods",
	}
	if queueableUpdate(sw, false) || !queueableUpdate(sw, true) {
		t.Fatal("a switch must be queued only when named")
	}
	req := sw.Request("stardew", "p")
	if req.Kind != queue.KindUpdate || req.Package != "998265" || req.Source != "curseforge" || req.Version != "777" ||
		req.CurrentKey != "nexus-20619-123175" || req.ModID != 0 || req.FileID != 0 {
		t.Fatalf("request = %+v", req)
	}
	forbidden := problems.Update{Key: "nexus-20619-123175", ID: "smapi:Example.Bush", Version: "1.0.1", Source: "CurseForge", NotDistributable: true}
	if queueableUpdate(forbidden, true) || queueableUpdate(forbidden, false) {
		t.Fatal("a forbidden project's update must never be queued: it would become a Nexus request for mod 0")
	}
	if !queueableUpdate(problems.Update{NexusID: 7, FileID: 9}, false) {
		t.Fatal("an ordinary update must still queue")
	}
}

func TestUpdatesSettledWaitsForRunningOnesAndNamesAFailure(t *testing.T) {
	t.Parallel()
	items := []queue.Item{
		{ID: "a", State: queue.StateDone},
		{ID: "b", State: queue.StateWaitingClick},
		{ID: "c", State: queue.StateDownloading},
		{ID: "d", State: queue.StateFailed, Error: "no space"},
		{ID: "e", State: queue.StateNeedsChoice, Name: "Pick"},
	}
	if done, err := updatesSettled(items, []string{"a", "b", "gone"}); !done || err != nil {
		t.Fatalf("finished and handed-over updates: %v, %v", done, err)
	}
	if done, _ := updatesSettled(items, []string{"a", "c"}); done {
		t.Fatal("a downloading update must hold the launch")
	}
	if done, err := updatesSettled(items, []string{"d"}); !done || err == nil || err.Error() != "no space" {
		t.Fatalf("failed update: %v, %v", done, err)
	}
	if done, err := updatesSettled(items, []string{"e"}); !done || err == nil {
		t.Fatalf("an update that needs an answer nobody is there to give: %v, %v", done, err)
	}
}

func TestMissingRequestsFetchWhatTheInstalledModsLackOncePerPage(t *testing.T) {
	t.Parallel()
	nexus := &problems.Ref{Site: "Nexus", PageID: 1915, PageName: "Content Patcher", URL: "https://x", FileID: 9}
	missing := []problems.Missing{
		{DependentName: "New", ID: "smapi:cp", Reason: "absent", Where: nexus},
		{DependentName: "New", ID: "smapi:cp2", Reason: "absent", Where: nexus},
		{DependentName: "New", ID: "smapi:gh", Reason: "absent", Where: &problems.Ref{Site: "GitHub", GitHub: "o/r", URL: "https://y"}},
		{DependentName: "Old", ID: "smapi:other", Reason: "absent", Where: nexus},
		{DependentName: "New", ID: "smapi:off", Reason: "disabled", Where: nexus},
		{DependentName: "New", ID: "outside:tool", Reason: "absent", External: true},
		{DependentName: "New", ID: "smapi:opt", Reason: "absent", Listed: true, Optional: true, Where: nexus},
		{DependentName: "New", ID: "smapi:unknown", Reason: "absent"},
	}
	got := missingRequests("stardew", "p", []string{"New"}, missing)
	if len(got) != 2 || got[0].ModID != 1915 || got[0].FileID != 9 || !got[0].Latest || got[0].Kind != queue.KindDependency || got[1].Repo != "o/r" {
		t.Fatalf("requests = %+v", got)
	}
}
