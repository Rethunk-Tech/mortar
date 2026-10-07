package control

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/archivesvc"
	"github.com/Rethunk-Tech/mortar/internal/avscan"
	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/bundles"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/loadersvc"
	"github.com/Rethunk-Tech/mortar/internal/logshare"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/storecheck"
	"github.com/Rethunk-Tech/mortar/internal/support"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/tools"
	"github.com/Rethunk-Tech/mortar/internal/updatesvc"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ChangedEvent tells the window a profile changed outside it, so it reloads that game's profiles.
const ChangedEvent = "profiles:changed"

// InstallAskEvent hands a CLI install that needs a choice (installer options or the mod's folder) to the window,
// which asks it the same way it asks for an archive dropped on it.
const InstallAskEvent = "install:ask"

// InstallAsk is the question an InstallAskEvent carries.
type InstallAsk struct {
	Game    string            `json:"game"`
	Profile string            `json:"profile"`
	Fomod   *profile.FomodAsk `json:"fomod,omitempty"`
	Remap   *profile.RemapAsk `json:"remap,omitempty"`
}

// OpenRequestMethod hands a share link or .mortar file to the running Mortar's import dialog, as a second launch
// would. A server build has no single-instance lock to carry that launch, so it forwards over this method.
const OpenRequestMethod = "open.request"

// Services are the running app's services the methods use.
type Services struct {
	Version  string
	Settings *settings.Store
	// SettingsSvc validates and saves settings changes, as the window's Settings does.
	SettingsSvc *settings.Service
	Games       *game.Service
	Store       *profile.Store
	Profiles    *profile.Service
	Problems    *problems.Service
	Launches    *launchsvc.Service
	Saves       *savessvc.Service
	Queue       *queue.Service
	Tools       *tools.Service
	Bundles     *bundles.Service
	Nexus       *nexussvc.Service
	Shares      *sharesvc.Service
	Data        *datasvc.Service
	Plays       *shortcut.Service
	Loaders     *loadersvc.Service
	Templates   *templates.Service
	Archives    *archivesvc.Service
	// Bisect and StoreCheck are the window's services; nil in tests that do not need them.
	Bisect     *bisect.Service
	StoreCheck *storecheck.Service
	Lan        *lan.Service
	Updates    *updatesvc.Service
	Nxm        *nxmsvc.Service
	Support    *support.Service
	Packs      *packsvc.Service
	// Emit is nil in tests that do not watch events.
	Emit func(name string, data any)
	// Quit closes the app without asking; nil in tests.
	Quit func()
	// QuitBlocker names what a quit would interrupt (downloads, a running game), or is empty; nil in tests.
	QuitBlocker func() string
	// Handoff takes command-line arguments as if a second instance had sent them; nil in tests.
	Handoff func(args []string)
}

// GameRow is one supported game for `mortar games`.
type GameRow struct {
	game.GameInfo
	Configured bool `json:"configured"`
	Profiles   int  `json:"profiles"`
}

// ModRow is one mod of a profile.
type ModRow struct {
	ID        mod.ID `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Author    string `json:"author"`
	Enabled   bool   `json:"enabled"`
	Pinned    bool   `json:"pinned"`
	PinReason string `json:"pinReason,omitempty"`
	Key       string `json:"key"`
	Source    string `json:"source"`
	// Note and Tags belong to the row's entry, so every mod of one download shows the same ones.
	Note string   `json:"note,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// ModInfo is one mod with what relates to it.
type ModInfo struct {
	ModRow
	Needs      []mod.ID `json:"needs"`
	Optional   []mod.ID `json:"optional"`
	Dependents []mod.ID `json:"dependents"`
	// OptionalFor is the mods that list this one as an optional dependency; Dependents require it.
	OptionalFor []mod.ID                  `json:"optionalFor"`
	Missing     []problems.Missing        `json:"missing"`
	Conflicts   []framework.AssetConflict `json:"conflicts"`
	Settings    []framework.SettingHint   `json:"settings"`
}

// ModProblem is a short, read-only problem shown beside a Nexus mod page.
type ModProblem struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// InstallOutcome is install's result without the whole profile.
type InstallOutcome struct {
	Added          []string `json:"added"`
	Updated        bool     `json:"updated"`
	VersionChanged bool     `json:"versionChanged"`
	// Needs names the choice the app must finish: "fomod" or "folder".
	Needs string `json:"needs,omitempty"`
}

// Removed lists the mods a remove took out, entry by entry.
type Removed struct {
	Mods []string `json:"mods"`
}

// ShareLink is a profile's share link.
type ShareLink struct {
	Web      string `json:"web"`
	App      string `json:"app"`
	TooLarge bool   `json:"tooLarge"`
}

// Exported is where export wrote and which settings files it left out.
type Exported struct {
	Path    string   `json:"path"`
	Skipped []string `json:"skipped"`
}

// RunLog is a stored SMAPI log.
type RunLog struct {
	Run  string `json:"run"`
	Text string `json:"text"`
}

// Doctor is the environment the app runs in.
type Doctor struct {
	Version     string                          `json:"version"`
	DataDir     string                          `json:"dataDir"`
	Games       []game.GameInfo                 `json:"games"`
	Environment map[string]problems.Environment `json:"environment"`
	NxmHandled  bool                            `json:"nxmHandled"`
	NxmPrevious string                          `json:"nxmPrevious"`
}

type BundleRow struct {
	bundles.Bundle
	Profiles []string `json:"profiles"`
}

type BundleApply struct {
	Added   int      `json:"added"`
	Missing []string `json:"missing"`
}

type NexusUntrack struct {
	nexussvc.UntrackAllResult
	Count int `json:"count"`
}

type ProfileMatch struct {
	Already   int      `json:"already"`
	Missing   []string `json:"missing"`
	Different []string `json:"different"`
	OnlyYours []string `json:"onlyYours"`
}

func modProblems(p profile.Profile, result problems.Result, source, id string) []ModProblem {
	var ids []mod.ID
	for _, entry := range p.Entries {
		if sourceID(entry.Source, source) == id {
			for _, im := range entry.Mods {
				ids = append(ids, im.ID)
			}
		}
	}
	involves := func(id mod.ID) bool {
		return slices.ContainsFunc(ids, func(want mod.ID) bool { return mod.Equal(want, id) })
	}
	out := []ModProblem{}
	for _, missing := range result.Missing {
		if !involves(missing.DependentID) {
			continue
		}
		need := missing.ID.Local()
		if missing.Where != nil && missing.Where.PageName != "" {
			need = missing.Where.PageName
		}
		kind, verb := "missing", "is missing"
		if missing.Reason == "outdated" {
			kind, verb = "outdated", "needs a newer"
		}
		out = append(out, ModProblem{Kind: kind, Text: fmt.Sprintf("%s %s %s", missing.DependentName, verb, need)})
	}
	for _, duplicate := range result.Duplicates {
		if involves(duplicate.ID) {
			out = append(out, ModProblem{Kind: "duplicate", Text: fmt.Sprintf("%s has duplicate copies", duplicate.Name)})
		}
	}
	for _, broken := range result.Broken {
		switch {
		case broken.Status == "cycle":
			if slices.ContainsFunc(broken.Cycle, func(d problems.Dependent) bool { return involves(d.ID) }) {
				out = append(out, ModProblem{Kind: "broken", Text: broken.Summary})
			}
		case involves(broken.ID):
			out = append(out, ModProblem{Kind: "broken", Text: fmt.Sprintf("%s is broken for this game version", broken.Name)})
		}
	}
	for _, conflict := range result.AssetConflicts {
		if conflict.Cosmetic || !slices.ContainsFunc(conflict.PackIDs, involves) {
			continue
		}
		var names []string
		for i, id := range conflict.PackIDs {
			if !involves(id) && i < len(conflict.Names) {
				names = append(names, conflict.Names[i])
			}
		}
		if len(names) == 0 {
			names = []string{"another mod"}
		}
		out = append(out, ModProblem{Kind: "conflict", Text: fmt.Sprintf("conflicts with %s", strings.Join(names, ", "))})
	}
	for _, run := range result.RunErrors {
		if involves(run.ID) {
			out = append(out, ModProblem{Kind: "error", Text: fmt.Sprintf("%s reported errors in the last run", run.Name)})
		}
	}
	return out
}

func resolveBundle(list []bundles.Bundle, name string) (bundles.Bundle, error) {
	for _, bundle := range list {
		if bundle.ID == name || bundle.Name == name {
			return bundle, nil
		}
	}
	return bundles.Bundle{}, fmt.Errorf("bundle %q not found", name)
}

// Handle runs one method against the live services.
func (s *Services) Handle(ctx context.Context, method string, p Params) (any, error) {
	switch method {
	case "app.quit":
		if s.Quit == nil {
			return nil, errors.New("quit is not available")
		}
		// No window may be open to ask, so a busy app refuses rather than reporting a quit it has not done.
		if s.QuitBlocker != nil && !p.Force {
			if why := s.QuitBlocker(); why != "" {
				return nil, fmt.Errorf("did not quit: %s; run mortar quit --force to quit anyway", strings.ToLower(why[:1])+why[1:])
			}
		}
		// After the reply, so the caller hears back before the app goes away. The pid lets it wait for the exit.
		go s.Quit()
		return os.Getpid(), nil
	case "loader.versions":
		return s.loaderVersions(ctx, p)
	case "loader.install":
		return s.loaderInstall(ctx, p)
	case "loader.pin":
		return nil, s.loaderPin(p)
	case "games":
		return s.games()
	case "game.steamLaunchOption":
		return s.gameSteamLaunchOption(p.Game, p.Set, p.Clear)
	case "game.launchPresetTemplates":
		return s.gameLaunchPresetTemplates(p)
	case "antivirus.status":
		v := s.Settings.Get()
		return avscan.StatusOf(ctx, avscan.Config{Mode: v.Antivirus, Socket: v.AntivirusSocket, Command: v.AntivirusCommand}), nil
	case "settings.get":
		cur := s.Settings.Get()
		if p.Key == "" {
			return cur.AllPrefsGame(p.Game), nil
		}
		v, err := cur.LookupGame(p.Key, p.Game)
		if err != nil {
			return nil, err
		}
		return [][2]string{{p.Key, v}}, nil
	case "settings.set":
		if p.Key == "" {
			return nil, fmt.Errorf("settings set needs a key")
		}
		return nil, s.SettingsSvc.SetByKey(p.Key, p.Value, p.Game)
	case "settings.export":
		if p.Path == "" {
			return nil, fmt.Errorf("settings export needs a file")
		}
		body, err := settings.MarshalExport(s.Settings.Get())
		if err != nil {
			return nil, err
		}
		if err := datadir.WriteFile(p.Path, body, 0o600); err != nil {
			return nil, err
		}
		return map[string]string{"path": p.Path}, nil
	case "settings.import":
		if p.Path == "" {
			return nil, fmt.Errorf("settings import needs a file")
		}
		return nil, s.SettingsSvc.ApplyImport(p.Path, settings.ImportSections())
	case "settings.reset":
		return nil, resetSettings(s.SettingsSvc, p.Key, p.Game)
	case "profiles":
		return s.Profiles.List(p.Game)
	case "trash.list":
		return s.Profiles.ListTrash(p.Game)
	case "trash.restore":
		item, err := s.resolveTrash(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Restore(p.Game, item.ID) })
	case "trash.delete":
		item, err := s.resolveTrash(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			return Removed{Mods: []string{item.Name}}, s.Profiles.Purge(p.Game, item.ID)
		})
	case "trash.empty":
		return s.changed(p.Game, func() (any, error) { return nil, s.Profiles.PurgeTrash(p.Game) })
	case "tools":
		if s.Tools == nil {
			return nil, errors.New("tools are unavailable")
		}
		return s.Tools.List(p.Game)
	case "tools.run":
		if s.Tools == nil {
			return nil, errors.New("tools are unavailable")
		}
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return nil, s.Tools.Launch(p.Game, prof.ID, p.Name)
	case "profile.create":
		if strings.TrimSpace(p.Name) == "" {
			return nil, errors.New("a profile needs a name")
		}
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Create(p.Game, p.Name) })
	case "profile.fromSave":
		if s.Saves == nil {
			return nil, errors.New("saves are unavailable")
		}
		return s.changed(p.Game, func() (any, error) { return s.Saves.FromSave(ctx, p.Game, p.Name) })
	case "saves.check":
		if s.Saves == nil {
			return nil, errors.New("saves are unavailable")
		}
		id := ""
		if p.Profile != "" {
			prof, err := s.resolve(p.Game, p.Profile)
			if err != nil {
				return nil, err
			}
			id = prof.ID
		}
		return s.Saves.Check(ctx, p.Game, p.Name, id)
	case "doctor":
		return s.doctor()
	case "launchers":
		return s.Games.Launchers()
	case "launchers.add":
		if err := s.SettingsSvc.AddLauncherRoot(p.Name, p.Path); err != nil {
			return nil, err
		}
		return s.Games.Launchers()
	case "launchers.remove":
		if err := s.SettingsSvc.RemoveLauncherRoot(p.Name, p.Path); err != nil {
			return nil, err
		}
		return s.Games.Launchers()
	case "queue":
		return s.Queue.State(), nil
	case "queue.retry":
		if p.Name == "" {
			s.Queue.RetryFailed()
		} else {
			s.Queue.Retry(p.Name)
		}
		return s.Queue.State(), nil
	case "queue.skip":
		if p.Name == "" {
			s.Queue.SkipAll()
		} else {
			s.Queue.Skip(p.Name)
		}
		return s.Queue.State(), nil
	case "queue.add":
		return s.queueAdd(ctx, p)
	case "queue.pause":
		s.Queue.Pause()
		return s.Queue.State(), nil
	case "queue.resume":
		s.Queue.Resume()
		return s.Queue.State(), nil
	case "queue.clear":
		s.Queue.ClearFinished()
		return s.Queue.State(), nil
	case "backups":
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		return s.Saves.ListBackups(p.Game, p.Profile)
	case "backups.restore":
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		id, err := s.optionalProfile(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return nil, s.Saves.RestoreBackup(p.Game, id, p.Name, p.IDs)
	case "backups.keep":
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		return nil, s.Saves.SetBackupPinned(p.Game, p.Name, true)
	case "backups.unkeep":
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		return nil, s.Saves.SetBackupPinned(p.Game, p.Name, false)
	case "backups.create":
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		id, err := s.optionalProfile(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		made, err := s.Saves.CreateBackup(p.Game, id, p.Name)
		if err == nil && !made {
			err = backup.ErrNoSaves
		}
		return nil, err
	case "bundles":
		if s.Bundles == nil {
			return nil, errors.New("bundles are unavailable")
		}
		list, err := s.Bundles.List(p.Game)
		if err != nil {
			return nil, err
		}
		profiles, err := s.Profiles.List(p.Game)
		if err != nil {
			return nil, err
		}
		out := make([]BundleRow, 0, len(list))
		for _, b := range list {
			ids := make(map[mod.ID]bool)
			for _, m := range b.Mods {
				ids[m.ID] = true
			}
			var names []string
			for _, prof := range profiles {
				if prof.Error != "" {
					continue
				}
				have := make(map[mod.ID]bool)
				for _, entry := range prof.Entries {
					for _, im := range entry.Mods {
						have[im.ID] = true
					}
				}
				ok := true
				for id := range ids {
					if !have[id] {
						ok = false
						break
					}
				}
				if ok {
					names = append(names, prof.Name)
				}
			}
			out = append(out, BundleRow{Bundle: b, Profiles: names})
		}
		return out, nil
	case "bundles.apply":
		if s.Bundles == nil {
			return nil, errors.New("bundles are unavailable")
		}
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		list, err := s.Bundles.List(p.Game)
		if err != nil {
			return nil, err
		}
		bundle, err := resolveBundle(list, p.Name)
		if err != nil {
			return nil, err
		}
		result, err := s.Bundles.Apply(p.Game, bundle.ID, prof.ID)
		if err != nil {
			return nil, err
		}
		return BundleApply{Added: result.Added, Missing: result.Missing}, nil
	case "source.untrack":
		if err := trackingSource(p.Source); err != nil {
			return nil, err
		}
		if s.Nexus == nil {
			return nil, errors.New("nexus is unavailable")
		}
		result, err := s.Nexus.UntrackAll(ctx, p.Game, p.Unused)
		if err != nil {
			return nil, err
		}
		return NexusUntrack{UntrackAllResult: result, Count: result.Untracked + result.Remaining}, nil
	case "source.tracked":
		if err := trackingSource(p.Source); err != nil {
			return nil, err
		}
		if s.Nexus == nil {
			return nil, errors.New("nexus is unavailable")
		}
		return s.Nexus.TrackedCount(ctx, p.Game)
	case "source.trackedMissing":
		return s.trackedMissing(ctx, p)
	case "changelog":
		if s.Nexus == nil {
			return nil, errors.New("nexus is unavailable")
		}
		return s.updateChangelog(ctx, p)
	case "cache.size":
		if s.Data == nil {
			return nil, errors.New("data is unavailable")
		}
		return s.Data.CacheInfo()
	case "cache.clear":
		if s.Data == nil {
			return nil, errors.New("data is unavailable")
		}
		return nil, s.Data.ClearCache()
	case "data.usageByMod":
		if s.Data == nil {
			return nil, errors.New("data is unavailable")
		}
		return s.Data.ModUsage()
	case "store.check":
		return s.storeCheck(ctx, p.Game)
	case "bisect.start":
		return s.bisectStart(ctx, p)
	case "bisect.status":
		return s.bisectStatus(p.Name)
	case "bisect.stop":
		return s.bisectStop(p.Name)
	case "store.report":
		return s.storeReport(p)
	case "store.remove":
		return s.storeRemove(p)
	case "history.all":
		return s.Profiles.RecentHistory(p.Game)
	case shortcut.PlayRequestMethod:
		if s.Handoff == nil {
			return nil, errors.New("play requests are not available")
		}
		r := shortcut.Request{Game: p.Game, Profile: p.Profile}
		s.Handoff([]string{shortcut.Arg(r.Game, r.Profile)})
		return r, nil
	case OpenRequestMethod:
		if s.Handoff == nil {
			return nil, errors.New("open requests are not available")
		}
		if p.Path == "" {
			return nil, errors.New("open needs a share link or .mortar file")
		}
		s.Handoff([]string{p.Path})
		return map[string]string{"path": p.Path}, nil
	case "status":
		if p.Install != "" {
			return s.Launches.StatusInstall(p.Game, p.Install)
		}
		return s.Launches.Status(p.Game)
	case "sweep":
		return s.Launches.Sweep(ctx, p.Game, p.Install)
	case "launch.vanilla":
		if err := s.Launches.StartVanilla(ctx, p.Game, s.Launches.LaunchesDirect(p.Game, "")); err != nil {
			return nil, err
		}
		return s.awaitStart(ctx, p.Game, "")
	case "stop":
		stop := func() error { return s.Launches.Stop(ctx, p.Game) }
		if p.Install != "" {
			stop = func() error { return s.Launches.StopInstall(ctx, p.Install) }
		}
		if err := stop(); err != nil {
			return nil, err
		}
		return s.Launches.Status(p.Game)
	case "updates.apply":
		return s.changed(p.Game, func() (any, error) { return s.applyEverywhere(ctx, p) })
	case "mods.by-author":
		return s.modsByAuthor(p)
	case "browse":
		return s.browseFromParams(ctx, p)
	case "pack.import":
		return s.packImport(ctx, p)
	case "pack.exportModpack":
		return s.packExportModpack(p)
	case "pack.farmExport", "pack.farmCheck", "pack.farmFix":
		return s.farmCall(ctx, method, p)
	case "profile.backup":
		return s.profileBackup(p)
	case "profile.restore":
		return s.profileRestore(ctx, p)
	}
	if res, ok, err := s.handleLibrary(ctx, method, p); ok {
		return res, err
	}
	if res, ok, err := s.handleMore(ctx, method, p); ok {
		return res, err
	}
	if method == "logs.search" {
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return s.Launches.SearchRuns(p.Game, prof.ID, p.Query)
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	id := prof.ID
	switch method {
	case "profile.rename":
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Rename(p.Game, id, p.Name) })
	case "profile.copy":
		return s.changed(p.Game, func() (any, error) { return s.copyProfile(p.Game, id, p.Name) })
	case "profile.compare":
		other, err := s.resolve(p.Game, p.Name)
		if err != nil {
			return nil, err
		}
		return profile.CompareProfilesCLI(prof, other), nil
	case "profile.match":
		if s.Shares == nil {
			return nil, errors.New("sharing is unavailable")
		}
		var preview sharesvc.Preview
		var err error
		if strings.HasSuffix(strings.ToLower(p.Path), ".mortar") {
			preview, err = s.Shares.PreviewFile(ctx, p.Game, p.Path, prof.ID)
		} else {
			preview, err = s.Shares.PreviewLink(ctx, p.Game, p.Path, prof.ID)
		}
		if err != nil {
			return nil, err
		}
		out := ProfileMatch{OnlyYours: preview.Replace.Remove}
		for _, im := range preview.Mods {
			switch {
			case im.Different:
				out.Different = append(out.Different, im.Name)
			case im.State == sharesvc.StateDownload || im.State == sharesvc.StateDependency:
				out.Missing = append(out.Missing, im.Name)
			case im.State == sharesvc.StateInstalled:
				out.Already++
			}
		}
		return out, nil
	case "profile.collection":
		if p.Unlink {
			return s.changed(p.Game, func() (any, error) {
				return s.Profiles.ClearCollection(p.Game, id)
			})
		}
		if s.Shares == nil {
			return nil, errors.New("sharing is unavailable")
		}
		if p.All {
			return s.changed(p.Game, func() (any, error) {
				preview, err := s.Shares.PreviewCollectionUpdate(ctx, p.Game, id)
				if err != nil {
					return nil, err
				}
				return s.Shares.Import(ctx, p.Game, preview.Session, id, nil)
			})
		}
		return s.Shares.CollectionStatus(ctx, p.Game, id)
	case "profile.loadOrder":
		return s.Profiles.LoadOrder(p.Game, id)
	case "history.diff":
		return s.historyDiff(p.Game, id, p.Name, p.Value)
	case "history.revert":
		return s.historyRevertItem(p.Game, id, p.Name, p.Value)
	case "profile.changes":
		return s.profileChanges(p.Game, id)
	case "profile.good":
		return s.profileGood(p.Game, id, p.All, p.Force)
	case "profile.history":
		events, err := s.Profiles.History(p.Game, id)
		if err != nil {
			return nil, err
		}
		// The CLI words each event from its fields; the settings snapshot and config list are not its to show.
		for i := range events {
			events[i].State, events[i].Configs = nil, nil
		}
		return events, nil
	case "profile.health":
		return s.Profiles.HealthHistory(p.Game, id)
	case "profile.revert":
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Revert(p.Game, id, p.Name) })
	case "profile.delete":
		return s.changed(p.Game, func() (any, error) {
			return Removed{Mods: []string{prof.Name}}, s.Profiles.Delete(p.Game, id)
		})
	case "profile.set":
		return s.changed(p.Game, func() (any, error) { return s.profileSet(p, prof, id) })
	case "store.repair":
		return s.storeRepair(ctx, p, id)
	case "profile.repair":
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Repair(p.Game, id) })
	case "profile.shortcut":
		return s.profileShortcut(p.Game, id, p.Remove)
	case "profile.steam":
		return s.profileSteam(p.Game, id)
	case "mods":
		return modRows(prof), nil
	case "mods.channel":
		return s.modsChannel(p, prof, id)
	case "mods.files":
		if len(p.IDs) == 0 {
			return nil, fmt.Errorf("mods files needs a mod")
		}
		return s.modExtraFiles(p.Game, prof, typedID(p.IDs[0]))
	case "mods.preset":
		return s.modsPreset(p, id, prof)
	case "mods.config":
		if len(p.IDs) == 0 {
			return nil, fmt.Errorf("mods config needs a mod")
		}
		refs, err := refsFor(prof, p.IDs[:1])
		if err != nil {
			return nil, err
		}
		ref := refs[0]
		if p.Key != "" {
			return s.changed(p.Game, func() (any, error) {
				if err := s.Profiles.SetConfigValue(p.Game, id, ref.Key, ref.ID, p.Key, p.Value); err != nil {
					return nil, err
				}
				return s.Profiles.ListConfigFields(p.Game, id, ref.Key, ref.ID)
			})
		}
		return s.Profiles.ListConfigFields(p.Game, id, ref.Key, ref.ID)
	case "mods.menu":
		return s.modsMenu(p)
	case "mods.report":
		return s.modReport(p.Game, id, prof, p)
	case "mod":
		return s.modInfo(ctx, p.Game, prof, p.IDs)
	case "mods.enable", "mods.disable":
		refs, err := refsFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetModsEnabled(p.Game, id, refs, method == "mods.enable")
		})
	case "mods.pin", "mods.unpin":
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.setEach(p.Game, id, prof, keys, func(k string) error {
			_, err := s.Profiles.SetPinned(p.Game, id, k, method == "mods.pin", p.Value)
			return err
		})
	case "mods.remove":
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) { return s.remove(p.Game, id, prof, keys) })
	case "mods.tag", "mods.untag":
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		if p.Value == "" {
			return nil, fmt.Errorf("mods %s needs a tag", strings.TrimPrefix(method, "mods."))
		}
		return s.changed(p.Game, func() (any, error) {
			if _, err := s.Profiles.SetEntryTagsMany(p.Game, id, keys, p.Value, method == "mods.tag"); err != nil {
				return nil, err
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	case "mods.category":
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			if _, err := s.Profiles.SetEntryCategoryMany(p.Game, id, keys, p.Value); err != nil {
				return nil, err
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	case "mods.note":
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			cur := s.reload(p.Game, id, prof)
			for _, k := range keys {
				tags := tagsOf(cur, k)
				if _, err := s.Profiles.SetEntryNoteTags(p.Game, id, k, p.Value, tags); err != nil {
					return nil, err
				}
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	case "mods.skip-version":
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.setEach(p.Game, id, prof, keys, func(k string) error {
			_, err := s.Profiles.SetSkipVersion(p.Game, id, k, p.Value)
			return err
		})
	case "mods.split":
		if len(p.IDs) != 2 {
			return nil, fmt.Errorf("mods split needs a mod and a file")
		}
		keys, err := keysFor(prof, p.IDs[:1])
		if err != nil {
			return nil, err
		}
		entry, ok := entryByKey(prof, keys[0])
		if !ok {
			return nil, fmt.Errorf("%q is not in this profile", keys[0])
		}
		extra, err := extraKeyOf(entry, p.IDs[1])
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			if _, err := s.Profiles.SplitExtra(p.Game, id, keys[0], extra); err != nil {
				return nil, err
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	case "mods.combine":
		if len(p.IDs) != 2 {
			return nil, fmt.Errorf("mods combine needs a mod and the entry to combine it into")
		}
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		if len(keys) != 2 {
			return nil, fmt.Errorf("those mods are already one entry")
		}
		return s.changed(p.Game, func() (any, error) {
			if _, err := s.Profiles.CombineEntries(p.Game, id, keys[1], keys[0]); err != nil {
				return nil, err
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	case "mods.win":
		return s.modsWin(p, prof, id)
	case "mods.group":
		return s.modsGroup(p, prof, id)
	case "install":
		return s.changed(p.Game, func() (any, error) { return s.install(ctx, p.Game, id, p.Path, p.Unscanned) })
	case "conflicts":
		res, err := s.Problems.ProblemsWithEvidence(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		out := []framework.AssetConflict{}
		for _, c := range res.AssetConflicts {
			if p.All || !c.Cosmetic {
				out = append(out, c)
			}
		}
		return out, nil
	case "who":
		return s.Problems.WhoChanges(ctx, p.Game, id, p.Query)
	case "conflicts.map":
		return s.Problems.AssetMap(ctx, p.Game, id, p.Query, false, 0)
	case "problems":
		return s.Problems.ProblemsWithEvidence(ctx, p.Game, id)
	case "compatibility":
		return s.Problems.CompatibilityFor(ctx, p.Game, id)
	case "problems.dismissed":
		res, err := s.Problems.Problems(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		return res.Dismissed, nil
	case "problems.dismiss":
		if p.Index < 1 {
			return nil, fmt.Errorf("missing problem index")
		}
		return s.changed(p.Game, func() (any, error) {
			return nil, s.dismissProblem(ctx, p.Game, id, p.Index)
		})
	case "problems.restore":
		return s.changed(p.Game, func() (any, error) {
			return nil, s.restoreDismissedProblem(ctx, p.Game, id, p.Name, p.Index)
		})
	case "modProblems":
		if p.Source == "" || p.ID == "" {
			return []ModProblem{}, nil
		}
		result, err := s.Problems.Problems(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		return modProblems(prof, result, p.Source, p.ID), nil
	case "installPackage":
		_, err := s.Queue.Add(ctx, []queue.Request{{Kind: queue.KindInstall, Game: p.Game, Profile: id, Package: p.Name}})
		return nil, err
	case "updates":
		return s.Problems.Updates(ctx, p.Game, id)
	case "updates.queue":
		r, err := s.Problems.Updates(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		var reqs []queue.Request
		for _, u := range r.Updates {
			named := len(p.IDs) > 0 && slices.ContainsFunc(typedIDs(p.IDs), func(want mod.ID) bool { return mod.Equal(want, u.ID) })
			if !queueableUpdate(u, named) || (!p.All && len(p.IDs) > 0 && !named) {
				continue
			}
			reqs = append(reqs, updateRequest(p.Game, id, u))
		}
		if len(reqs) == 0 {
			if p.All || len(p.IDs) == 0 {
				return nil, usererr.New(usererr.NotFound, "no updates available")
			}
			return nil, usererr.New(usererr.NotFound, "no update available for "+strings.Join(p.IDs, ", "))
		}
		var out QueuedUpdates
		if p.All {
			batch, err := s.Profiles.BeginUpdateBatch(p.Game, id)
			if err != nil {
				return nil, err
			}
			out.Before = batch.Before
			for i := range reqs {
				reqs[i].BatchID = batch.Batch
			}
		}
		if _, err := s.Queue.Add(ctx, reqs); err != nil {
			return nil, err
		}
		out.Queued, out.Queue = len(reqs), s.Queue.State()
		return out, nil
	case "share":
		res, err := share.Encode(p.Game, prof, s.Profiles.ShareFacts(p.Game, prof))
		if errors.Is(err, share.ErrTooLarge) {
			return ShareLink{TooLarge: true}, nil
		}
		if err != nil {
			return nil, err
		}
		return ShareLink{Web: res.Web, App: res.App}, nil
	case "export":
		return s.export(p.Game, prof, p.Path)
	case "runs":
		return s.Launches.Runs(p.Game, id)
	case "logs":
		return s.runLog(p.Game, id, p.Run)
	case "logs.share":
		return s.shareLog(ctx, p.Game, id, p.Run)
	case "logs.fixes":
		return s.Launches.RunProblems(p.Game, id, p.Run)
	case "logs.search":
		return s.Launches.SearchRuns(p.Game, id, p.Query)
	case "saves":
		return s.Saves.Saves(ctx, p.Game, id)
	case "play.check":
		return s.playCheck(ctx, p.Game, id, prof)
	case "play.test":
		return s.playTest(ctx, p.Game, id, p.Install)
	case "launch":
		return s.launch(ctx, p.Game, id, p.Install, p.Preset, p.Force)
	case "perf.reports":
		return s.Launches.PerformanceReports(p.Game, id)
	}
	return nil, fmt.Errorf("unknown method %q", method)
}

// changed runs a mutating call and tells the window to reload the game's profiles.
func (s *Services) changed(gameID string, fn func() (any, error)) (any, error) {
	res, err := fn()
	if err == nil && s.Emit != nil {
		s.Emit(ChangedEvent, gameID)
	}
	return res, err
}

func (s *Services) games() ([]GameRow, error) {
	list, err := s.Games.List()
	if err != nil {
		return nil, err
	}
	out := make([]GameRow, 0, len(list))
	for _, g := range list {
		ps, _ := s.Profiles.List(g.ID)
		n := 0
		for _, p := range ps {
			if p.Error == "" {
				n++
			}
		}
		out = append(out, GameRow{GameInfo: g, Configured: g.Installed && g.InstallDir != "", Profiles: n})
	}
	return out, nil
}

// optionalProfile is the id of the profile sel names, "" when sel is empty.
func (s *Services) optionalProfile(gameID, sel string) (string, error) {
	if sel == "" {
		return "", nil
	}
	p, err := s.resolve(gameID, sel)
	return p.ID, err
}

// resolve finds a profile by id, or by name ignoring case; an ambiguous name lists the matching ids.
func (s *Services) resolve(gameID, sel string) (profile.Profile, error) {
	if sel == "" {
		return profile.Profile{}, errors.New("name a profile")
	}
	all, err := s.Profiles.List(gameID)
	if err != nil {
		return profile.Profile{}, err
	}
	for _, p := range all {
		if p.ID == sel {
			return p, nil
		}
	}
	var hits []profile.Profile
	for _, p := range all {
		if p.Error == "" && strings.EqualFold(p.Name, sel) {
			hits = append(hits, p)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return profile.Profile{}, fmt.Errorf("no %s profile is named or has the id %q", gameID, sel)
	}
	ids := make([]string, len(hits))
	for i, p := range hits {
		ids[i] = p.ID
	}
	return profile.Profile{}, fmt.Errorf("%d profiles are named %q; use an id: %s", len(hits), sel, strings.Join(ids, ", "))
}

// resolveTrash finds a trashed profile by id, or by name ignoring case; an ambiguous name lists the matching ids.
func (s *Services) resolveTrash(gameID, sel string) (profile.TrashItem, error) {
	if sel == "" {
		return profile.TrashItem{}, errors.New("name a deleted profile")
	}
	all, err := s.Profiles.ListTrash(gameID)
	if err != nil {
		return profile.TrashItem{}, err
	}
	for _, item := range all {
		if item.ID == sel {
			return item, nil
		}
	}
	var hits []profile.TrashItem
	for _, item := range all {
		if strings.EqualFold(item.Name, sel) {
			hits = append(hits, item)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return profile.TrashItem{}, fmt.Errorf("no deleted %s profile is named or has the id %q", gameID, sel)
	}
	ids := make([]string, len(hits))
	for i, item := range hits {
		ids[i] = item.ID
	}
	return profile.TrashItem{}, fmt.Errorf("%d deleted profiles are named %q; use an id: %s", len(hits), sel, strings.Join(ids, ", "))
}

func (s *Services) reload(gameID, id string, fallback profile.Profile) profile.Profile {
	if p, err := s.resolve(gameID, id); err == nil {
		return p
	}
	return fallback
}

func (s *Services) copyProfile(gameID, id, name string) (profile.Profile, error) {
	p, err := s.Profiles.Duplicate(gameID, id)
	if err != nil || name == "" {
		return p, err
	}
	return s.Profiles.Rename(gameID, p.ID, name)
}

func source(src profile.Source) string {
	switch src.Kind {
	case profile.KindNexus:
		return fmt.Sprintf("nexus:%d/%d", src.ModID, src.FileID)
	case profile.KindGitHub:
		return "github:" + src.Repo + "@" + src.Tag
	}
	return src.Kind
}

func modRows(p profile.Profile) []ModRow {
	out := []ModRow{}
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			out = append(out, ModRow{
				ID: m.ID, Name: m.Name, Version: m.Version, Author: m.Author, Key: e.Key,
				Enabled: e.Enabled(m.ID),
				Pinned:  e.Pinned, PinReason: e.PinReason, Source: source(e.Source),
				Note: e.Note, Tags: e.Tags,
			})
		}
	}
	return out
}

func extraKeyOf(e profile.Entry, id string) (string, error) {
	if slices.Contains(e.ExtraStoreKeys, id) {
		return id, nil
	}
	prefixFor := func(extra string) string { return filepath.ToSlash(extra) + "/" }
	for _, extra := range e.ExtraStoreKeys {
		prefix := prefixFor(extra)
		for _, m := range e.Mods {
			folder := filepath.ToSlash(m.Folder)
			if folder != extra && !strings.HasPrefix(folder, prefix) {
				continue
			}
			if mod.Equal(m.ID, typedID(id)) || strings.EqualFold(m.Name, id) {
				return extra, nil
			}
		}
	}
	return "", fmt.Errorf("%q is not an extra file of this entry", id)
}

func refsFor(p profile.Profile, ids []string) ([]profile.EnableRef, error) {
	if len(ids) == 0 {
		return nil, errors.New("name at least one mod by id")
	}
	refs := make([]profile.EnableRef, 0, len(ids))
	for _, id := range ids {
		e, _, ok := p.FindMod("", typedID(id))
		if !ok {
			return nil, fmt.Errorf("profile %s has no mod %q", p.Name, id)
		}
		refs = append(refs, profile.EnableRef{Key: e.Key, ID: typedID(id)})
	}
	return refs, nil
}

func keysFor(p profile.Profile, ids []string) ([]string, error) {
	refs, err := refsFor(p, ids)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, r := range refs {
		if !slices.Contains(keys, r.Key) {
			keys = append(keys, r.Key)
		}
	}
	return keys, nil
}

// remove takes out whole entries, so a mod that shares its download with others takes those with it; the reply
// names every mod removed.
func (s *Services) remove(gameID, id string, p profile.Profile, keys []string) (Removed, error) {
	out := Removed{Mods: []string{}}
	for _, k := range keys {
		for _, e := range p.Entries {
			if e.Key == k {
				for _, m := range e.Mods {
					out.Mods = append(out.Mods, m.Name)
				}
			}
		}
		if _, err := s.Profiles.RemoveEntry(gameID, id, k); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (s *Services) install(ctx context.Context, gameID, id, path string, unscanned bool) (InstallOutcome, error) {
	if _, err := os.Stat(path); err != nil {
		return InstallOutcome{}, err
	}
	res, err := s.Profiles.InstallArchive(ctx, gameID, id, path)
	if det, ok := errors.AsType[*store.DetectedError](err); ok && unscanned {
		if err = s.Profiles.AllowUnscanned(gameID, id, det.Key, filepath.Base(path), det.Name); err == nil {
			res, err = s.Profiles.InstallArchive(ctx, gameID, id, path)
		}
	}
	if err != nil {
		if _, ok := errors.AsType[*profile.NeedChoicesError](err); ok {
			return InstallOutcome{Needs: "fomod"}, nil
		}
		return InstallOutcome{}, err
	}
	out := InstallOutcome{Added: res.Added, Updated: res.Updated, VersionChanged: res.VersionChanged}
	switch {
	case res.Fomod != nil:
		out.Needs = "fomod"
	case res.Remap != nil:
		out.Needs = "folder"
	}
	if out.Needs != "" && s.Emit != nil {
		s.Emit(InstallAskEvent, InstallAsk{Game: gameID, Profile: id, Fomod: res.Fomod, Remap: res.Remap})
	}
	if out.Added == nil {
		out.Added = []string{}
	}
	return out, nil
}

// dependentsOf is the mods of p that list uid as a dependency, split into those that require it and those that
// list it as optional.
func dependentsOf(p profile.Profile, uid mod.ID) (required, optional []mod.ID) {
	required, optional = []mod.ID{}, []mod.ID{}
	is := func(n mod.ID) bool { return mod.Equal(n, uid) }
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			switch {
			case !slices.ContainsFunc(m.Needs, is):
			case slices.ContainsFunc(m.Optional, is):
				optional = append(optional, m.ID)
			default:
				required = append(required, m.ID)
			}
		}
	}
	return required, optional
}

func (s *Services) modInfo(ctx context.Context, gameID string, p profile.Profile, ids []string) (ModInfo, error) {
	if len(ids) != 1 {
		return ModInfo{}, errors.New("name one mod by id")
	}
	uid := typedID(ids[0])
	e, _, ok := p.FindMod("", uid)
	if !ok {
		return ModInfo{}, fmt.Errorf("profile %s has no mod %q", p.Name, uid)
	}
	info := ModInfo{Needs: []mod.ID{}, Optional: []mod.ID{}, Missing: []problems.Missing{}, Conflicts: []framework.AssetConflict{}, Settings: []framework.SettingHint{}}
	for _, r := range modRows(p) {
		if mod.Equal(r.ID, uid) {
			info.ModRow = r
		}
	}
	for _, m := range e.Mods {
		if mod.Equal(m.ID, uid) {
			info.Optional = append(info.Optional, m.Optional...)
			for _, n := range m.Needs {
				if !slices.ContainsFunc(m.Optional, func(o mod.ID) bool { return mod.Equal(o, n) }) {
					info.Needs = append(info.Needs, n)
				}
			}
		}
	}
	info.Dependents, info.OptionalFor = dependentsOf(p, uid)
	res, err := s.Problems.Problems(ctx, gameID, p.ID)
	if err != nil {
		return info, err
	}
	for _, m := range res.Missing {
		if mod.Equal(m.DependentID, uid) {
			info.Missing = append(info.Missing, m)
		}
	}
	for _, c := range res.AssetConflicts {
		if slices.ContainsFunc(c.PackIDs, func(id mod.ID) bool { return mod.Equal(id, uid) }) {
			info.Conflicts = append(info.Conflicts, c)
		}
	}
	for _, h := range res.Settings {
		if mod.Equal(h.ID, uid) {
			info.Settings = append(info.Settings, h)
		}
	}
	return info, nil
}

func (s *Services) export(gameID string, p profile.Profile, path string) (Exported, error) {
	if path == "" {
		return Exported{}, errors.New("name the .mortar file to write")
	}
	modsDir, err := s.Store.ModsDir(gameID, p.ID)
	if err != nil {
		return Exported{}, err
	}
	dir, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return Exported{}, err
	}
	defer func() { _ = dir.Close() }()
	f, err := dir.Create(filepath.Base(path))
	if err != nil {
		return Exported{}, err
	}
	skipped, err := share.Write(f, gameID, p, modsDir)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = dir.Remove(filepath.Base(path))
		return Exported{}, err
	}
	if skipped == nil {
		skipped = []string{}
	}
	return Exported{Path: path, Skipped: skipped}, nil
}

func (s *Services) runLog(gameID, id, run string) (RunLog, error) {
	if run == "" {
		runs, err := s.Launches.Runs(gameID, id)
		if err != nil {
			return RunLog{}, err
		}
		if len(runs) == 0 {
			return RunLog{}, errors.New("this profile has no stored runs")
		}
		latest := runs[0]
		for _, r := range runs[1:] {
			if r.Started > latest.Started {
				latest = r
			}
		}
		run = latest.ID
	}
	text, err := s.Launches.RunLog(gameID, id, run)
	if err != nil {
		return RunLog{}, err
	}
	if text == "" {
		return RunLog{}, fmt.Errorf("run %s has no stored log", run)
	}
	return RunLog{Run: run, Text: text}, nil
}

func (s *Services) shareLog(ctx context.Context, gameID, id, run string) (map[string]string, error) {
	if err := support.CanShare(gameID, s.Store.LoaderID(gameID, id)); err != nil {
		return nil, err
	}
	l, err := s.runLog(gameID, id, run)
	if err != nil {
		return nil, err
	}
	link, err := logshare.New().Upload(ctx, l.Text)
	if err != nil {
		return nil, err
	}
	return map[string]string{"url": link, "run": l.Run}, nil
}

// launchWait bounds how long launch waits for the game to leave Launching.
const launchWait = 3 * time.Minute

func gameName(id string) string {
	if g := game.Find(id); g != nil {
		return g.Name()
	}
	return id
}

type launchWarningError struct {
	game   string
	update problems.UpdateWarning
	save   savessvc.Fit
	gap    bool
}

func (e launchWarningError) Error() string {
	var warnings []string
	if e.update.Changed {
		warning := fmt.Sprintf(
			"The game was updated: %s is now %s; this profile last launched on %s.",
			e.game,
			e.update.Installed,
			e.update.Recorded,
		)
		if len(e.update.Broken) == 0 {
			warning += " None of this profile's mods are marked broken for the new version."
		} else {
			names := make([]string, 0, len(e.update.Broken))
			for _, im := range e.update.Broken {
				names = append(names, im.Name)
			}
			warning += " Mods marked broken: " + strings.Join(names, ", ") + "."
		}
		warnings = append(warnings, warning)
	}
	if e.gap {
		farm := "The farm"
		if e.save.Farmer != "" {
			farm = e.save.Farmer + "'s farm"
		}
		warnings = append(warnings, fmt.Sprintf(
			"Your last save needs other mods: %s (%s) was last played with mods this profile does not have on.",
			farm,
			e.save.Folder,
		))
	}
	return strings.Join(append(warnings, "Play anyway with --force."), "\n")
}

const playIssueNameCap = 5

// PlayIssueGroup is one pre-Play warning bucket, matching the window's playIssues summary.
type PlayIssueGroup struct {
	Kind  string   `json:"kind"`
	Count int      `json:"count"`
	Names []string `json:"names"`
}

func (s *Services) playCheck(ctx context.Context, gameID, id string, prof profile.Profile) ([]PlayIssueGroup, error) {
	res, err := s.Problems.Problems(ctx, gameID, id)
	if err != nil {
		return nil, err
	}
	upd, err := s.Problems.Updates(ctx, gameID, id)
	if err != nil {
		return nil, err
	}
	smapiNever := false
	if s.Settings != nil {
		smapiNever = s.Settings.Get().SmapiBuilds == settings.SmapiBuildsNever
	}
	groups := playIssueGroups(prof, res, upd, smapiNever)
	if ch := s.changesPlayGroup(ctx, gameID, id); ch.Kind != "" {
		groups = append(groups, ch)
	}
	if s.Saves != nil {
		if save, gap := adviceOnly(s.Saves.LastSaveGap(ctx, gameID, id)); gap {
			groups = append(groups, saveGapGroup(save))
		}
	}
	return groups, nil
}

// saveGapGroup is the newest save's warning that Play gives: the mods it was last played with that the profile lacks,
// else the ones the save file itself names.
func saveGapGroup(save savessvc.Fit) PlayIssueGroup {
	lacks := save.LastMissing
	if len(lacks) == 0 {
		lacks = save.Missing
	}
	names := make([]string, 0, min(len(lacks), playIssueNameCap))
	for _, l := range lacks[:min(len(lacks), playIssueNameCap)] {
		names = append(names, l.Name)
	}
	return PlayIssueGroup{Kind: "save", Count: len(lacks), Names: names}
}

func playIssueGroups(prof profile.Profile, res problems.Result, upd problems.UpdatesResult, smapiNever bool) []PlayIssueGroup {
	var missing []string
	seen := map[string]bool{}
	for _, m := range res.Missing {
		// Missing has a row per dependent mod; the summary counts each absent mod once.
		if m.Optional || seen[m.ID.Fold()] {
			continue
		}
		seen[m.ID.Fold()] = true
		name := m.ID.Local()
		if m.Where != nil && m.Where.PageName != "" {
			name = m.Where.PageName
		}
		missing = append(missing, name)
	}
	var conflicts []string
	for _, c := range res.AssetConflicts {
		if c.Cosmetic {
			continue
		}
		names := make([]string, 0, len(c.Names))
		for _, n := range c.Names {
			if n != "" {
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			conflicts = append(conflicts, strings.Join(names, ", "))
		} else {
			conflicts = append(conflicts, c.Target)
		}
	}
	var updates []string
	for _, u := range upd.Updates {
		if smapiNever && u.Unofficial {
			continue
		}
		e, ok := entryByKey(prof, u.Key)
		if ok && (e.Pinned || (u.Source != "" && slices.Contains(e.SkipSources, u.Source)) || (e.SkipVersion != "" && e.SkipVersion == u.Version)) {
			continue
		}
		updates = append(updates, u.Name)
	}
	var broken []string
	for _, b := range res.Broken {
		if b.Status == "broken" || b.Status == "obsolete" || b.Status == "cycle" {
			broken = append(broken, b.Name)
		}
	}
	var out []PlayIssueGroup
	for _, g := range []struct {
		kind   string
		labels []string
	}{
		{"missing", missing},
		{"conflicts", conflicts},
		{"updates", updates},
		{"broken", broken},
	} {
		if len(g.labels) == 0 {
			continue
		}
		names := g.labels
		if len(names) > playIssueNameCap {
			names = names[:playIssueNameCap]
		}
		out = append(out, PlayIssueGroup{Kind: g.kind, Count: len(g.labels), Names: names})
	}
	return out
}

func entryByKey(p profile.Profile, key string) (profile.Entry, bool) {
	for _, e := range p.Entries {
		if e.Key == key {
			return e, true
		}
	}
	return profile.Entry{}, false
}

func tagsOf(p profile.Profile, key string) []string {
	e, ok := entryByKey(p, key)
	if !ok {
		return nil
	}
	return e.Tags
}

func resetSettings(svc *settings.Service, key, game string) error {
	found := false
	for _, spec := range settings.PrefSpecs() {
		if key != "" && spec.Key != key {
			continue
		}
		found = true
		if spec.Scope == settings.ScopeGame && game == "" {
			if key != "" {
				return fmt.Errorf("settings reset %s needs --game", key)
			}
			continue
		}
		if err := svc.SetByKey(spec.Key, spec.Default, game); err != nil {
			return err
		}
	}
	if key != "" && !found {
		return fmt.Errorf("unknown setting %q", key)
	}
	return nil
}

// adviceOnly is the save-gap check's answer when it is advice for the player: it needs the mod dataset from the
// network, so a failed lookup says nothing instead of refusing to launch.
func adviceOnly(save savessvc.Fit, gap bool, err error) (savessvc.Fit, bool) {
	if err != nil {
		return savessvc.Fit{}, false
	}
	return save, gap
}

func (s *Services) launch(ctx context.Context, gameID, id, installID, preset string, force bool) (launchsvc.Status, error) {
	if !force {
		update, err := s.Problems.UpdateWarning(ctx, gameID, id)
		if err != nil {
			return launchsvc.Status{}, err
		}
		if save, gap := adviceOnly(s.Saves.LastSaveGap(ctx, gameID, id)); update.Changed || gap {
			return launchsvc.Status{}, launchWarningError{game: gameName(gameID), update: update, save: save, gap: gap}
		}
	}
	if err := s.Launches.StartPreset(ctx, gameID, id, installID, preset, s.Launches.LaunchesDirect(gameID, id)); err != nil {
		return launchsvc.Status{}, err
	}
	return s.awaitStart(ctx, gameID, installID)
}

// awaitStart waits for a launch Mortar just began to reach the game running, or to fail.
func (s *Services) awaitStart(ctx context.Context, gameID, installID string) (launchsvc.Status, error) {
	deadline := time.Now().Add(launchWait)
	for {
		st, err := s.Launches.Status(gameID)
		if installID != "" {
			st, err = s.Launches.StatusInstall(gameID, installID)
		}
		if err != nil {
			return st, err
		}
		switch st.State {
		case launchsvc.Running:
			return st, nil
		case launchsvc.Idle:
			// Failed is announced but not kept, so a launch that failed reads as idle; its cause is kept apart.
			if why := s.Launches.LaunchFailure(gameID, st.Install); why != "" {
				return st, errors.New(why)
			}
			return st, nil
		case launchsvc.Launching:
		case launchsvc.Failed, launchsvc.NoSteam:
			if st.Error != "" {
				return st, errors.New(st.Error)
			}
			return st, fmt.Errorf("launch ended as %s", st.State)
		}
		if time.Now().After(deadline) {
			return st, errors.New("the game did not start within 3 minutes")
		}
		select {
		case <-ctx.Done():
			return st, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (s *Services) doctor() (Doctor, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return Doctor{}, err
	}
	gs, err := s.Games.List()
	if err != nil {
		return Doctor{}, err
	}
	env := map[string]problems.Environment{}
	for _, g := range gs {
		env[g.ID] = s.Problems.Environment(g.ID)
	}
	st := s.Settings.Get()
	return Doctor{Version: s.Version, DataDir: dir, Games: gs, Environment: env, NxmHandled: st.NxmHandled, NxmPrevious: st.NxmPreviousName}, nil
}

// typedID reads an id a user typed: bare ids are SMAPI's for now, since Stardew is the only game with mods.
func typedID(s string) mod.ID { return mod.Parse(s, mod.FormatSMAPI) }

func typedIDs(ids []string) []mod.ID {
	out := make([]mod.ID, len(ids))
	for i, id := range ids {
		out[i] = typedID(id)
	}
	return out
}

// updateRequest is the queue request that installs an update, from the source the update names.
// queueableUpdate reports whether updates.queue may queue u. An unofficial update or one that needs a file picked is left
// to the review, and so is one whose project forbids outside downloads (the review links its page). A switch to another
// site is offered, never applied on its own: only a mod the caller named is switched.
func queueableUpdate(u problems.Update, named bool) bool {
	return !u.Unofficial && !u.PickFile && !u.NotDistributable && (!u.Switch || named)
}

func updateRequest(game, profileID string, u problems.Update) queue.Request {
	req := queue.Request{Kind: queue.KindUpdate, Game: game, Profile: profileID, Name: u.Name, Version: u.Version, CurrentKey: u.Key}
	switch {
	case u.Package != "":
		// A Modrinth update names its exact version, since a version number can repeat across loaders.
		req.Package, req.Source, req.Version = u.Package, u.PackageSource, cmp.Or(u.PackageVersion, u.Version)
	case u.GitHubRepo != "":
		req.Repo = u.GitHubRepo
	default:
		req.ModID, req.FileID, req.FallbackRepo, req.FallbackID, req.Latest = u.NexusID, u.FileID, u.GitHubFallback, u.ID, true
	}
	return req
}

// QueuedUpdates is what updates.queue queued. Before is the restore point Update all took, which a revert to undoes the
// whole batch; it is empty when only named mods were queued.
type QueuedUpdates struct {
	Queued int         `json:"queued"`
	Before string      `json:"before,omitempty"`
	Queue  queue.State `json:"queue"`
}
