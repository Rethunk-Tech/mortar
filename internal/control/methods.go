package control

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/devsvc"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archivesvc"
	"github.com/Rethunk-Tech/mortar/internal/avscan"
	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/browse"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/loadersvc"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/playcheck"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savedgroups"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/storecheck"
	"github.com/Rethunk-Tech/mortar/internal/support"
	"github.com/Rethunk-Tech/mortar/internal/syncsvc"
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
	PlayCheck   *playcheck.Service
	Launches    *launchsvc.Service
	Saves       *savessvc.Service
	Queue       *queue.Service
	Tools       *tools.Service
	SavedGroups *savedgroups.Service
	Nexus       *nexussvc.Service
	Shares      *sharesvc.Service
	Data        *datasvc.Service
	Plays       *shortcut.Service
	Loaders     *loadersvc.Service
	Dev         *devsvc.Service
	Templates   *templates.Service
	Archives    *archivesvc.Service
	// Browse is the window's browse service, so a search honours the same settings and holdings as the window.
	Browse *browse.Service
	// Bisect and StoreCheck are the window's services; nil in tests that do not need them.
	Bisect     *bisect.Service
	StoreCheck *storecheck.Service
	Lan        *lan.Service
	Updates    *updatesvc.Service
	Nxm        *nxmsvc.Service
	Support    *support.Service
	Packs      *packsvc.Service
	// Sync is nil in tests that do not need it.
	Sync *syncsvc.Service
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
	// Item and File name the archive a folder game's per-file entry is cut from and its file in it; Tray marks the entry
	// that holds the archive's files in the game's shared Tray folder.
	Item string `json:"item,omitempty"`
	File string `json:"file,omitempty"`
	Tray bool   `json:"tray,omitempty"`
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
	// Requirements names the missing required mods the install queued for download, under the game's
	// missingRequirements setting.
	Requirements []string `json:"requirements,omitempty"`
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
	SteamRoots  []string                        `json:"steamRoots,omitempty"`
}

type SavedGroupRow struct {
	savedgroups.SavedGroup
	Profiles []string `json:"profiles"`
}

type SavedGroupApply struct {
	Added   int      `json:"added"`
	Missing []string `json:"missing"`
	Group   string   `json:"group"`
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
		out = append(out, ModProblem{Kind: "conflict", Text: fmt.Sprintf("conflicts with %s", strings.Join(otherPacks(conflict, involves), ", "))})
	}
	for _, run := range result.RunErrors {
		if involves(run.ID) {
			out = append(out, ModProblem{Kind: "error", Text: fmt.Sprintf("%s reported errors in the last run", run.Name)})
		}
	}
	return out
}

// otherPacks names the packs of a conflict that the mod in question is not, or "another mod" when none is named.
func otherPacks(conflict framework.AssetConflict, involves func(mod.ID) bool) []string {
	var names []string
	for i, id := range conflict.PackIDs {
		if !involves(id) && i < len(conflict.Names) {
			names = append(names, conflict.Names[i])
		}
	}
	if len(names) == 0 {
		return []string{"another mod"}
	}
	return names
}

func resolveSavedGroup(list []savedgroups.SavedGroup, name string) (savedgroups.SavedGroup, error) {
	for _, group := range list {
		if group.ID == name || group.Name == name {
			return group, nil
		}
	}
	return savedgroups.SavedGroup{}, fmt.Errorf("saved group %q not found", name)
}

// holdsAll reports whether the profile has every mod of the saved group. Optional files have no id and travel with
// their main mod, so they are not counted.
func holdsAll(prof profile.Profile, g savedgroups.SavedGroup) bool {
	have := map[string]bool{}
	for _, entry := range prof.Entries {
		for _, m := range entry.Mods {
			have[m.ID.Fold()] = true
		}
	}
	return !slices.ContainsFunc(g.Mods, func(m savedgroups.Mod) bool { return m.ID != "" && !have[m.ID.Fold()] })
}

// savedGroupRows lists the game's saved groups with the profiles that already hold every mod of each.
func (s *Services) savedGroupRows(game string) (any, error) {
	if s.SavedGroups == nil {
		return nil, errors.New("saved groups are unavailable")
	}
	list, err := s.SavedGroups.List(game)
	if err != nil {
		return nil, err
	}
	profiles, err := s.Profiles.List(game)
	if err != nil {
		return nil, err
	}
	out := make([]SavedGroupRow, 0, len(list))
	for _, g := range list {
		var names []string
		for _, prof := range profiles {
			if prof.Error == "" && holdsAll(prof, g) {
				names = append(names, prof.Name)
			}
		}
		out = append(out, SavedGroupRow{SavedGroup: g, Profiles: names})
	}
	return out, nil
}

// modsSplit moves one extra file of a mod out into an entry of its own.
func (s *Services) modsSplit(p Params, prof profile.Profile, id string) (any, error) {
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
}

type (
	globalMethod  func(s *Services, ctx context.Context, p Params) (any, error)
	profileMethod func(s *Services, ctx context.Context, p Params, prof profile.Profile, id string) (any, error)
)

func mergeGlobal(tables ...map[string]globalMethod) map[string]globalMethod {
	n := 0
	for _, t := range tables {
		n += len(t)
	}
	out := make(map[string]globalMethod, n)
	for _, t := range tables {
		maps.Copy(out, t)
	}
	return out
}

func mergeProfile(tables ...map[string]profileMethod) map[string]profileMethod {
	n := 0
	for _, t := range tables {
		n += len(t)
	}
	out := make(map[string]profileMethod, n)
	for _, t := range tables {
		maps.Copy(out, t)
	}
	return out
}

var (
	globalMethods  = mergeGlobal(coreGlobal, libraryGlobal, moreGlobal)
	profileMethods = mergeProfile(coreProfile)
)

// Handle runs one method against the live services.
func (s *Services) Handle(ctx context.Context, method string, p Params) (any, error) {
	if fn, ok := globalMethods[method]; ok {
		return fn(s, ctx, p)
	}
	if fn, ok := profileMethods[method]; ok {
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return fn(s, ctx, p, prof, prof.ID)
	}
	return nil, fmt.Errorf("unknown method %q", method)
}

var coreGlobal = map[string]globalMethod{
	"app.quit": func(s *Services, _ context.Context, p Params) (any, error) {
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
	},
	"dev.add":         func(s *Services, ctx context.Context, p Params) (any, error) { return s.devAdd(ctx, p) },
	"dev.clone":       func(s *Services, ctx context.Context, p Params) (any, error) { return s.devClone(ctx, p) },
	"dev.list":        func(s *Services, _ context.Context, p Params) (any, error) { return s.devList(p) },
	"dev.remove":      func(s *Services, _ context.Context, p Params) (any, error) { return s.devRemove(p) },
	"loader.versions": func(s *Services, ctx context.Context, p Params) (any, error) { return s.loaderVersions(ctx, p) },
	"loader.install":  func(s *Services, ctx context.Context, p Params) (any, error) { return s.loaderInstall(ctx, p) },
	"loader.pin":      func(s *Services, _ context.Context, p Params) (any, error) { return nil, s.loaderPin(p) },
	"games":           func(s *Services, _ context.Context, _ Params) (any, error) { return s.games() },
	"game.steamLaunchOption": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.gameSteamLaunchOption(p.Game, p.Set, p.Clear)
	},
	"game.folder": func(s *Services, _ context.Context, p Params) (any, error) {
		if err := s.SettingsSvc.SetGameFolder(p.Game, p.Path); err != nil {
			return nil, err
		}
		return GameFolderResult{Game: p.Game, Folder: s.Settings.Get().GameFolders[p.Game]}, nil
	},
	"game.launchPresetTemplates": func(s *Services, _ context.Context, p Params) (any, error) { return s.gameLaunchPresetTemplates(p) },
	"antivirus.status": func(s *Services, ctx context.Context, _ Params) (any, error) {
		v := s.Settings.Get()
		return avscan.StatusOf(ctx, v.AntivirusConfig()), nil
	},
	"settings.get": func(s *Services, _ context.Context, p Params) (any, error) {
		cur := s.Settings.Get()
		if p.Key == "" {
			return cur.AllPrefsGame(p.Game), nil
		}
		v, err := cur.LookupGame(p.Key, p.Game)
		if err != nil {
			return nil, err
		}
		return [][2]string{{p.Key, v}}, nil
	},
	"settings.set": func(s *Services, _ context.Context, p Params) (any, error) {
		if p.Key == "" {
			return nil, fmt.Errorf("settings set needs a key")
		}
		return nil, s.SettingsSvc.SetByKey(p.Key, p.Value, p.Game)
	},
	"settings.export": func(s *Services, _ context.Context, p Params) (any, error) {
		if p.Path == "" {
			return nil, fmt.Errorf("settings export needs a file")
		}
		body, err := settings.MarshalExport(s.Settings.Get())
		if err != nil {
			return nil, err
		}
		if err := safefs.WriteFileAtomic(p.Path, body, 0o600); err != nil {
			return nil, err
		}
		return map[string]string{"path": p.Path}, nil
	},
	"settings.import": func(s *Services, _ context.Context, p Params) (any, error) {
		if p.Path == "" {
			return nil, fmt.Errorf("settings import needs a file")
		}
		return nil, s.SettingsSvc.ApplyImport(p.Path, settings.ImportSections())
	},
	"settings.reset": func(s *Services, _ context.Context, p Params) (any, error) {
		return nil, resetSettings(s.SettingsSvc, p.Key, p.Game)
	},
	"profiles":   func(s *Services, _ context.Context, p Params) (any, error) { return s.Profiles.List(p.Game) },
	"trash.list": func(s *Services, _ context.Context, p Params) (any, error) { return s.Profiles.ListTrash(p.Game) },
	"trash.restore": func(s *Services, _ context.Context, p Params) (any, error) {
		item, err := s.resolveTrash(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Restore(p.Game, item.ID) })
	},
	"trash.delete": func(s *Services, _ context.Context, p Params) (any, error) {
		item, err := s.resolveTrash(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			return Removed{Mods: []string{item.Name}}, s.Profiles.Purge(p.Game, item.ID)
		})
	},
	"trash.empty": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.changed(p.Game, func() (any, error) { return nil, s.Profiles.PurgeTrash(p.Game) })
	},
	"tools": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Tools == nil {
			return nil, errors.New("tools are unavailable")
		}
		return s.Tools.List(p.Game)
	},
	"tools.run": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Tools == nil {
			return nil, errors.New("tools are unavailable")
		}
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return nil, s.Tools.Launch(p.Game, prof.ID, p.Name)
	},
	"profile.create": func(s *Services, _ context.Context, p Params) (any, error) {
		if strings.TrimSpace(p.Name) == "" {
			return nil, errors.New("a profile needs a name")
		}
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Create(p.Game, p.Name) })
	},
	"profile.fromSave": func(s *Services, ctx context.Context, p Params) (any, error) {
		if s.Saves == nil {
			return nil, errors.New("saves are unavailable")
		}
		return s.changed(p.Game, func() (any, error) { return s.Saves.FromSave(ctx, p.Game, p.Name) })
	},
	"saves.check": func(s *Services, ctx context.Context, p Params) (any, error) {
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
	},
	"doctor":    func(s *Services, _ context.Context, _ Params) (any, error) { return s.doctor() },
	"launchers": func(s *Services, _ context.Context, _ Params) (any, error) { return s.Games.Launchers() },
	"launchers.add": func(s *Services, _ context.Context, p Params) (any, error) {
		if err := s.SettingsSvc.AddLauncherRoot(p.Name, p.Path); err != nil {
			return nil, err
		}
		return s.Games.Launchers()
	},
	"launchers.remove": func(s *Services, _ context.Context, p Params) (any, error) {
		if err := s.SettingsSvc.RemoveLauncherRoot(p.Name, p.Path); err != nil {
			return nil, err
		}
		return s.Games.Launchers()
	},
	"queue": func(s *Services, _ context.Context, _ Params) (any, error) { return s.Queue.State(), nil },
	"queue.retry": func(s *Services, _ context.Context, p Params) (any, error) {
		var n int
		if p.Name == "" {
			n = s.Queue.RetryFailed()
		} else {
			n = s.Queue.Retry(p.Name)
		}
		st := s.Queue.State()
		st.Changed = n
		return st, nil
	},
	"queue.skip": func(s *Services, _ context.Context, p Params) (any, error) {
		var n int
		if p.Name == "" {
			n = s.Queue.SkipAll()
		} else {
			n = s.Queue.Skip(p.Name)
		}
		st := s.Queue.State()
		st.Changed = n
		return st, nil
	},
	"queue.add": func(s *Services, ctx context.Context, p Params) (any, error) { return s.queueAdd(ctx, p) },
	"queue.pause": func(s *Services, _ context.Context, _ Params) (any, error) {
		s.Queue.Pause()
		return s.Queue.State(), nil
	},
	"queue.resume": func(s *Services, _ context.Context, _ Params) (any, error) {
		s.Queue.Resume()
		return s.Queue.State(), nil
	},
	"queue.pause-item": func(s *Services, _ context.Context, p Params) (any, error) {
		s.Queue.PauseItem(p.Name)
		return s.Queue.State(), nil
	},
	"queue.resume-item": func(s *Services, _ context.Context, p Params) (any, error) {
		s.Queue.ResumeItem(p.Name)
		return s.Queue.State(), nil
	},
	"queue.top": func(s *Services, _ context.Context, p Params) (any, error) {
		s.Queue.Top(p.Name)
		return s.Queue.State(), nil
	},
	"queue.up": func(s *Services, _ context.Context, p Params) (any, error) {
		s.Queue.Move(p.Name, -1)
		return s.Queue.State(), nil
	},
	"queue.down": func(s *Services, _ context.Context, p Params) (any, error) {
		s.Queue.Move(p.Name, 1)
		return s.Queue.State(), nil
	},
	"queue.clear": func(s *Services, _ context.Context, _ Params) (any, error) {
		s.Queue.ClearFinished()
		return s.Queue.State(), nil
	},
	"backups": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		return s.Saves.ListBackups(p.Game, p.Profile)
	},
	"backups.restore": func(s *Services, ctx context.Context, p Params) (any, error) {
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		id, err := s.optionalProfile(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		return nil, s.Saves.RestoreBackup(ctx, p.Game, id, p.Name, p.IDs)
	},
	"backups.keep": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		return nil, s.Saves.SetBackupPinned(p.Game, p.Name, true)
	},
	"backups.unkeep": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		return nil, s.Saves.SetBackupPinned(p.Game, p.Name, false)
	},
	"backups.create": func(s *Services, ctx context.Context, p Params) (any, error) {
		if s.Saves == nil {
			return nil, errors.New("backups are unavailable")
		}
		id, err := s.optionalProfile(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		made, err := s.Saves.CreateBackup(ctx, p.Game, id, p.Name)
		if err == nil && !made {
			err = backup.ErrNoSaves
		}
		return nil, err
	},
	"groups": func(s *Services, _ context.Context, p Params) (any, error) { return s.savedGroupRows(p.Game) },
	"groups.apply": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.SavedGroups == nil {
			return nil, errors.New("saved groups are unavailable")
		}
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		list, err := s.SavedGroups.List(p.Game)
		if err != nil {
			return nil, err
		}
		group, err := resolveSavedGroup(list, p.Name)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			result, err := s.SavedGroups.Apply(p.Game, group.ID, prof.ID)
			if err != nil {
				return nil, err
			}
			return SavedGroupApply{Added: result.Added, Missing: result.Missing, Group: result.Group}, nil
		})
	},
	"source.untrack": func(s *Services, ctx context.Context, p Params) (any, error) {
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
	},
	"source.tracked": func(s *Services, ctx context.Context, p Params) (any, error) {
		if err := trackingSource(p.Source); err != nil {
			return nil, err
		}
		if s.Nexus == nil {
			return nil, errors.New("nexus is unavailable")
		}
		if p.Unused {
			return s.Nexus.TrackedUnusedCount(ctx, p.Game)
		}
		return s.Nexus.TrackedCount(ctx, p.Game)
	},
	"source.trackedMissing": func(s *Services, ctx context.Context, p Params) (any, error) { return s.trackedMissing(ctx, p) },
	"changelog": func(s *Services, ctx context.Context, p Params) (any, error) {
		if s.Nexus == nil {
			return nil, errors.New("nexus is unavailable")
		}
		return s.updateChangelog(ctx, p)
	},
	"cache.size": func(s *Services, _ context.Context, _ Params) (any, error) {
		if s.Data == nil {
			return nil, errors.New("data is unavailable")
		}
		return s.Data.CacheInfo()
	},
	"cache.clear": func(s *Services, _ context.Context, _ Params) (any, error) {
		if s.Data == nil {
			return nil, errors.New("data is unavailable")
		}
		return nil, s.Data.ClearCache()
	},
	"data.usageByMod": func(s *Services, _ context.Context, _ Params) (any, error) {
		if s.Data == nil {
			return nil, errors.New("data is unavailable")
		}
		return s.Data.ModUsage()
	},
	"store.check":   func(s *Services, ctx context.Context, p Params) (any, error) { return s.storeCheck(ctx, p.Game) },
	"bisect.start":  func(s *Services, ctx context.Context, p Params) (any, error) { return s.bisectStart(ctx, p) },
	"bisect.status": func(s *Services, _ context.Context, p Params) (any, error) { return s.bisectStatus(p.Name) },
	"bisect.stop":   func(s *Services, _ context.Context, p Params) (any, error) { return s.bisectStop(p.Name) },
	"store.report":  func(s *Services, _ context.Context, p Params) (any, error) { return s.storeReport(p) },
	"store.remove":  func(s *Services, _ context.Context, p Params) (any, error) { return s.storeRemove(p) },
	"history.all":   func(s *Services, _ context.Context, p Params) (any, error) { return s.Profiles.RecentHistory(p.Game) },
	shortcut.PlayRequestMethod: func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Handoff == nil {
			return nil, errors.New("play requests are not available")
		}
		r := shortcut.Request{Game: p.Game, Profile: p.Profile}
		s.Handoff([]string{shortcut.Arg(r.Game, r.Profile)})
		return r, nil
	},
	OpenRequestMethod: func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Handoff == nil {
			return nil, errors.New("open requests are not available")
		}
		if p.Path == "" {
			return nil, errors.New("open needs a share link or .mortar file")
		}
		s.Handoff([]string{p.Path})
		return map[string]string{"path": p.Path}, nil
	},
	"status": func(s *Services, _ context.Context, p Params) (any, error) {
		if p.Install != "" {
			return s.Launches.StatusInstall(p.Game, p.Install)
		}
		return s.Launches.Status(p.Game)
	},
	"sweep": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.Launches.Sweep(ctx, p.Game, p.Install)
	},
	"launch.vanilla": func(s *Services, ctx context.Context, p Params) (any, error) {
		if err := s.Launches.StartVanilla(ctx, p.Game, s.Launches.LaunchesDirect(p.Game, "")); err != nil {
			return nil, err
		}
		return s.awaitStart(ctx, p.Game, "")
	},
	"stop": func(s *Services, ctx context.Context, p Params) (any, error) {
		if p.Install != "" {
			if _, err := s.Launches.StatusInstall(p.Game, p.Install); err != nil {
				return nil, err
			}
			if err := s.Launches.StopInstall(ctx, p.Install); err != nil {
				return nil, err
			}
			return s.Launches.StatusInstall(p.Game, p.Install)
		}
		if err := s.Launches.Stop(ctx, p.Game); err != nil {
			return nil, err
		}
		return s.Launches.Status(p.Game)
	},
	"updates.apply": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.applyEverywhere(ctx, p) })
	},
	"mods.by-author":     func(s *Services, _ context.Context, p Params) (any, error) { return s.modsByAuthor(p) },
	"browse":             func(s *Services, ctx context.Context, p Params) (any, error) { return s.browseFromParams(ctx, p) },
	"pack.import":        func(s *Services, ctx context.Context, p Params) (any, error) { return s.packImport(ctx, p) },
	"pack.exportModpack": func(s *Services, _ context.Context, p Params) (any, error) { return s.packExportModpack(p) },
	"pack.exportR2z":     func(s *Services, _ context.Context, p Params) (any, error) { return s.packExportR2z(p) },
	"pack.farmExport": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.farmCall(ctx, "pack.farmExport", p)
	},
	"pack.farmCheck": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.farmCall(ctx, "pack.farmCheck", p)
	},
	"pack.farmFix": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.farmCall(ctx, "pack.farmFix", p)
	},
	"profile.backup":  func(s *Services, _ context.Context, p Params) (any, error) { return s.profileBackup(p) },
	"profile.restore": func(s *Services, ctx context.Context, p Params) (any, error) { return s.profileRestore(ctx, p) },
	"mods.favourites": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.Settings.Get().GamePrefs(p.Game).FavouriteMods, nil
	},
	"logs.search": func(s *Services, _ context.Context, p Params) (any, error) { return s.searchLogs(p) },
}

var coreProfile = map[string]profileMethod{
	"profile.rename": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Rename(p.Game, id, p.Name) })
	},
	"profile.copy": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.copyProfile(p.Game, id, p.Name) })
	},
	"profile.compare": func(s *Services, _ context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		other, err := s.resolve(p.Game, p.Name)
		if err != nil {
			return nil, err
		}
		return profile.CompareProfilesCLI(prof, other), nil
	},
	"profile.match": func(s *Services, ctx context.Context, p Params, prof profile.Profile, _ string) (any, error) {
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
	},
	"profile.collection": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
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
				return s.Shares.Import(ctx, p.Game, preview.Session, id, nil, false)
			})
		}
		return s.Shares.CollectionStatus(ctx, p.Game, id)
	},
	"profile.loadOrder": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Profiles.LoadOrder(p.Game, id)
	},
	"history.diff": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.historyDiff(p.Game, id, p.Name, p.Value)
	},
	"history.revert": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.historyRevertItem(p.Game, id, p.Name, p.Value)
	},
	"profile.changes": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.profileChanges(p.Game, id)
	},
	"profile.good": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.profileGood(p.Game, id, p.All, p.Force)
	},
	"profile.history": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		events, err := s.Profiles.History(p.Game, id)
		if err != nil {
			return nil, err
		}
		// The CLI words each event from its fields; the settings snapshot and config list are not its to show.
		for i := range events {
			events[i].State, events[i].Configs = nil, nil
		}
		return events, nil
	},
	"profile.health": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Profiles.HealthHistory(p.Game, id)
	},
	"profile.revert": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Revert(p.Game, id, p.Name) })
	},
	"profile.delete": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) {
			return Removed{Mods: []string{prof.Name}}, s.Profiles.Delete(p.Game, id)
		})
	},
	"profile.set": func(s *Services, ctx context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.profileSet(ctx, p, prof, id) })
	},
	"store.repair": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.storeRepair(ctx, p, id)
	},
	"profile.repair": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Repair(p.Game, id) })
	},
	"profile.freeze": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.Profiles.SetFrozen(p.Game, id, true) })
	},
	"profile.unfreeze": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.Profiles.SetFrozen(p.Game, id, false) })
	},
	"profile.favourite": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetFavourite(p.Game, id, true)
		})
	},
	"profile.unfavourite": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetFavourite(p.Game, id, false)
		})
	},
	"mods.favourite": func(s *Services, _ context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		return s.setModFavourites(p.Game, prof, p.IDs, true)
	},
	"mods.unfavourite": func(s *Services, _ context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		return s.setModFavourites(p.Game, prof, p.IDs, false)
	},
	"profile.shortcut": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.profileShortcut(p.Game, id, p.Remove)
	},
	"profile.steam": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.profileSteam(p.Game, id)
	},
	"mods": func(_ *Services, _ context.Context, _ Params, prof profile.Profile, _ string) (any, error) {
		return modRows(prof), nil
	},
	"mods.channel": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.modsChannel(p, prof, id)
	},
	"mods.files": func(s *Services, _ context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		if len(p.IDs) == 0 {
			return nil, fmt.Errorf("mods files needs a mod")
		}
		uid, err := typedID(modIDs(prof), p.IDs[0])
		if err != nil {
			return nil, err
		}
		return s.modExtraFiles(p.Game, prof, uid)
	},
	"mods.preset": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.modsPreset(p, id, prof)
	},
	"mods.config": func(s *Services, ctx context.Context, p Params, prof profile.Profile, id string) (any, error) {
		if len(p.IDs) == 0 {
			return nil, fmt.Errorf("mods config needs a mod")
		}
		refs, err := refsFor(prof, p.IDs[:1])
		if err != nil {
			return nil, err
		}
		ref := refs[0]
		cfg := s.configs()
		if p.Key != "" {
			return s.changed(p.Game, func() (any, error) {
				if ok, err := cfg.SetField(ctx, p.Game, id, string(ref.ID), p.Key, p.Value); ok || err != nil {
					if err != nil {
						return nil, err
					}
					return s.modConfigFields(cfg, p.Game, id, ref)
				}
				if err := s.Profiles.SetConfigValue(p.Game, id, ref.Key, ref.ID, p.Key, p.Value); err != nil {
					return nil, err
				}
				return s.modConfigFields(cfg, p.Game, id, ref)
			})
		}
		return s.modConfigFields(cfg, p.Game, id, ref)
	},
	"console.send": func(s *Services, ctx context.Context, p Params, _ profile.Profile, _ string) (any, error) {
		return nil, s.Launches.Send(ctx, p.Game, p.Value)
	},
	"screenshots": func(s *Services, _ context.Context, p Params, _ profile.Profile, _ string) (any, error) {
		return s.Games.Screenshots(p.Game)
	},
	"nexus.endorse": func(s *Services, ctx context.Context, p Params, _ profile.Profile, _ string) (any, error) {
		return s.nexusEndorse(ctx, p)
	},
	"sync.status": func(s *Services, ctx context.Context, p Params, _ profile.Profile, _ string) (any, error) {
		return s.syncCall(ctx, "sync.status", p)
	},
	"sync.apply": func(s *Services, ctx context.Context, p Params, _ profile.Profile, _ string) (any, error) {
		return s.syncCall(ctx, "sync.apply", p)
	},
	"mods.menu": func(s *Services, _ context.Context, p Params, _ profile.Profile, _ string) (any, error) {
		return s.modsMenu(p)
	},
	"mods.report": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.modReport(p.Game, id, prof, p)
	},
	"mod": func(s *Services, ctx context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		return s.modInfo(ctx, p.Game, prof, p.IDs)
	},
	"mods.enable": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		refs, err := refsFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetModsEnabled(p.Game, id, refs, true)
		})
	},
	"mods.disable": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		refs, err := refsFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetModsEnabled(p.Game, id, refs, false)
		})
	},
	"mods.pin": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.setEach(p.Game, id, prof, p.IDs, func(k string) error {
			_, err := s.Profiles.SetPinned(p.Game, id, k, true, p.Value)
			return err
		})
	},
	"mods.unpin": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.setEach(p.Game, id, prof, p.IDs, func(k string) error {
			_, err := s.Profiles.SetPinned(p.Game, id, k, false, p.Value)
			return err
		})
	},
	"mods.remove": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) { return s.remove(p.Game, id, prof, keys) })
	},
	"mods.tag": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		if p.Value == "" {
			return nil, fmt.Errorf("mods tag needs a tag")
		}
		return s.changed(p.Game, func() (any, error) {
			if _, err := s.Profiles.SetEntryTagsMany(p.Game, id, keys, p.Value, true); err != nil {
				return nil, err
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	},
	"mods.untag": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		keys, err := keysFor(prof, p.IDs)
		if err != nil {
			return nil, err
		}
		if p.Value == "" {
			return nil, fmt.Errorf("mods untag needs a tag")
		}
		return s.changed(p.Game, func() (any, error) {
			if _, err := s.Profiles.SetEntryTagsMany(p.Game, id, keys, p.Value, false); err != nil {
				return nil, err
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	},
	"mods.category": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
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
	},
	"mods.note": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
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
	},
	"mods.skip-version": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.setEach(p.Game, id, prof, p.IDs, func(k string) error {
			_, err := s.Profiles.SetSkipVersion(p.Game, id, k, p.Value)
			return err
		})
	},
	"mods.split": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.modsSplit(p, prof, id)
	},
	"mods.combine": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
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
	},
	"mods.win": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.modsWin(p, prof, id)
	},
	"mods.group": func(s *Services, _ context.Context, p Params, prof profile.Profile, id string) (any, error) {
		return s.modsGroup(p, prof, id)
	},
	"install": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) { return s.install(ctx, p.Game, id, p.Path, p.Unscanned) })
	},
	"conflicts": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
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
	},
	"who": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Problems.WhoChanges(ctx, p.Game, id, p.Query)
	},
	"conflicts.map": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Problems.AssetMap(ctx, p.Game, id, p.Query, false, 0)
	},
	"problems": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Problems.ProblemsWithEvidence(ctx, p.Game, id)
	},
	"compatibility": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Problems.CompatibilityFor(ctx, p.Game, id)
	},
	"problems.dismissed": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		res, err := s.Problems.Problems(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		return res.Dismissed, nil
	},
	"problems.dismiss": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		if p.Index < 1 {
			return nil, fmt.Errorf("missing problem index")
		}
		return s.changed(p.Game, func() (any, error) {
			return nil, s.dismissProblem(ctx, p.Game, id, p.Index)
		})
	},
	"problems.restore": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.changed(p.Game, func() (any, error) {
			return nil, s.restoreDismissedProblem(ctx, p.Game, id, p.Name, p.Index)
		})
	},
	"modProblems": func(s *Services, ctx context.Context, p Params, prof profile.Profile, id string) (any, error) {
		if p.Source == "" || p.ID == "" {
			return []ModProblem{}, nil
		}
		result, err := s.Problems.Problems(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		return modProblems(prof, result, p.Source, p.ID), nil
	},
	"installPackage": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		_, err := s.Queue.Add(ctx, []queue.Request{{Kind: queue.KindInstall, Game: p.Game, Profile: id, Package: p.Name, Source: p.Source}})
		return nil, err
	},
	"updates": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Problems.Updates(ctx, p.Game, id)
	},
	"updates.queue": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		out, _, err := s.queueUpdates(ctx, p.Game, id, p.IDs, p.All, false)
		if err != nil {
			return nil, err
		}
		return out, nil
	},
	"share": func(s *Services, _ context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		res, err := share.Encode(p.Game, prof, s.Profiles.ShareFacts(p.Game, prof), share.IncludeOf(s.Settings.Get()))
		if errors.Is(err, share.ErrTooLarge) {
			return ShareLink{TooLarge: true}, nil
		}
		if err != nil {
			return nil, err
		}
		return ShareLink{Web: res.Web, App: res.App}, nil
	},
	"export": func(s *Services, _ context.Context, p Params, prof profile.Profile, _ string) (any, error) {
		return s.export(p.Game, prof, p.Path)
	},
	"runs": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Launches.Runs(p.Game, id)
	},
	"logs": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.runLog(p.Game, id, p.Run)
	},
	"logs.share": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.shareLog(ctx, p.Game, id, p.Run)
	},
	"logs.fixes": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Launches.RunProblems(p.Game, id, p.Run)
	},
	"saves": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Saves.Saves(ctx, p.Game, id)
	},
	"play.check": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.PlayCheck.Check(ctx, p.Game, id)
	},
	"play.test": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.playTest(ctx, p.Game, id, p.Install)
	},
	"launch": func(s *Services, ctx context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.launch(ctx, p.Game, id, p.Install, p.Preset, p.Force)
	},
	"perf.reports": func(s *Services, _ context.Context, p Params, _ profile.Profile, id string) (any, error) {
		return s.Launches.PerformanceReports(p.Game, id)
	},
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
// searchLogs searches one profile's stored run logs, or every readable profile's of the game when none is named.
func (s *Services) searchLogs(p Params) (launchsvc.RunSearch, error) {
	if p.Profile != "" {
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return launchsvc.RunSearch{}, err
		}
		return s.Launches.SearchRuns(p.Game, prof.ID, p.Query)
	}
	all, err := s.Profiles.List(p.Game)
	if err != nil {
		return launchsvc.RunSearch{}, err
	}
	out := launchsvc.RunSearch{Hits: []launchsvc.RunHit{}}
	for _, prof := range all {
		if prof.Error != "" {
			continue
		}
		got, err := s.Launches.SearchRuns(p.Game, prof.ID, p.Query)
		if err != nil {
			return launchsvc.RunSearch{}, err
		}
		out.Hits = append(out.Hits, got.Hits...)
		out.Truncated = out.Truncated || got.Truncated
	}
	return out, nil
}

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
				Item: e.Item, File: e.File, Tray: e.InTray(),
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
			if idMatches(id, m.ID) || strings.EqualFold(m.Name, id) {
				return extra, nil
			}
		}
	}
	return "", fmt.Errorf("%q is not an extra file of this entry", id)
}

// refsFor names the mods the ids stand for. An id is a mod id (typed, or the part after the colon), or the store item of
// a folder game's archive, which stands for every mod of every entry cut from it, its Tray entry included.
func refsFor(p profile.Profile, ids []string) ([]profile.EnableRef, error) {
	if len(ids) == 0 {
		return nil, errors.New("name at least one mod by id")
	}
	refs := make([]profile.EnableRef, 0, len(ids))
	known := modIDs(p)
	for _, id := range ids {
		uid, err := typedID(known, id)
		if err != nil {
			return nil, err
		}
		if e, _, ok := p.FindMod("", uid); ok {
			refs = append(refs, profile.EnableRef{Key: e.Key, ID: uid})
			continue
		}
		before := len(refs)
		add := func(e profile.Entry, m profile.Component) {
			ref := profile.EnableRef{Key: e.Key, ID: m.ID}
			if !slices.Contains(refs, ref) {
				refs = append(refs, ref)
			}
		}
		for _, e := range p.Entries {
			for _, m := range e.Mods {
				if strings.EqualFold(m.ID.Local(), strings.TrimSpace(id)) || e.StoreKey() == id {
					add(e, m)
				}
			}
		}
		if len(refs) == before {
			return nil, fmt.Errorf("profile %s has no mod %q", p.Name, id)
		}
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
	if out.Needs == "" && len(out.Added) > 0 && s.Queue != nil {
		out.Requirements = s.queueMissing(ctx, gameID, res.Profile, out.Added)
	}
	return out, nil
}

// missingRequests is the downloads for the required mods that the mods named added still lack, one per page. It
// leaves out what Mortar cannot fetch: a requirement outside the mod sites, and one no source is known for.
func missingRequests(gameID, id string, added []string, missing []problems.Missing) []queue.Request {
	var out []queue.Request
	for _, m := range missing {
		if m.Reason != "absent" || m.External || (m.Listed && m.Optional) || m.Where == nil || m.Where.URL == "" || !slices.Contains(added, m.DependentName) {
			continue
		}
		req := queue.Request{Kind: queue.KindDependency, Game: gameID, Profile: id}
		switch {
		case m.Where.Site == "GitHub" && m.Where.GitHub != "":
			req.Repo, req.Name = m.Where.GitHub, m.Where.GitHub
		case m.Where.Site == "Nexus" && m.Where.PageID > 0:
			req.ModID, req.FileID, req.Latest = m.Where.PageID, int(m.Where.FileID), true
			req.Name, req.FileName, req.Version = m.Where.PageName, m.Where.FileName, m.Where.Version
		default:
			continue
		}
		if !slices.ContainsFunc(out, func(o queue.Request) bool { return o.Repo == req.Repo && o.ModID == req.ModID }) {
			out = append(out, req)
		}
	}
	return out
}

// queueMissing downloads what the just-installed mods still need when the game's missingRequirements setting says
// to, and names what it queued. With nobody to ask, "ask" queues nothing; a failed lookup is logged and the install
// stands.
func (s *Services) queueMissing(ctx context.Context, gameID string, p profile.Profile, added []string) []string {
	mode := settings.ResolveAt(s.Settings.Get(), "missingRequirements", settings.Scope{Game: gameID, Profile: p.ID}, p.PrefOverrides())
	if mode != settings.MissingReqAutodownload {
		return nil
	}
	result, err := s.Problems.Problems(ctx, gameID, p.ID)
	if err != nil {
		slog.Warn("install: missing requirements not checked", "id", p.ID, "err", err)
		return nil
	}
	reqs := missingRequests(gameID, p.ID, added, result.Missing)
	if len(reqs) == 0 {
		return nil
	}
	if _, err := s.Queue.Add(ctx, reqs); err != nil {
		slog.Warn("install: missing requirements not queued", "id", p.ID, "err", err)
		return nil
	}
	names := make([]string, len(reqs))
	for i, r := range reqs {
		names[i] = r.Name
	}
	return names
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
	uid, err := typedID(modIDs(p), ids[0])
	if err != nil {
		return ModInfo{}, err
	}
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
	include := share.IncludeOf(s.Settings.Get())
	if s.Shares != nil {
		include = s.Shares.WithDismissed(gameID, p.ID, include)
	}
	skipped, err := share.Write(f, gameID, p, modsDir, include)
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
	link, err := s.Support.Upload(ctx, gameID, s.Store.LoaderID(gameID, id), l.Text)
	if err != nil {
		return nil, err
	}
	return map[string]string{"url": link, "run": l.Run}, nil
}

// launchWait bounds how long launch waits for the game to leave Launching.
var launchWait = 3 * time.Minute

// gameName is the catalog display name. Launch warnings keep the id when the
// game is unknown so the player still sees which game failed; shortcut and
// Steam need a real name, so those callers treat ok=false as NotFound.
func gameName(id string) (string, bool) {
	return game.NameOf(id), game.Find(id) != nil
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
		if err := svc.SetByKey(spec.Key, spec.DefaultFor(game), game); err != nil {
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
	if id != "" && s.Launches.UpdatesBeforePlay(gameID, id) {
		if err := s.updateBeforePlay(ctx, gameID, id); err != nil {
			return launchsvc.Status{}, fmt.Errorf("updating mods before Play: %w", err)
		}
	}
	if !force && !s.Launches.SkipsPlayCheck(gameID, id) {
		update, err := s.Problems.UpdateWarning(ctx, gameID, id)
		if err != nil {
			return launchsvc.Status{}, err
		}
		if save, gap := adviceOnly(s.Saves.LastSaveGap(ctx, gameID, id)); update.Changed || gap {
			name, _ := gameName(gameID)
			return launchsvc.Status{}, launchWarningError{game: name, update: update, save: save, gap: gap}
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
		var (
			st  launchsvc.Status
			err error
		)
		if installID != "" {
			st, err = s.Launches.StatusInstall(gameID, installID)
		} else {
			st, err = s.Launches.Status(gameID)
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
	return Doctor{Version: s.Version, DataDir: dir, Games: gs, Environment: env, NxmHandled: st.NxmHandled, NxmPrevious: st.NxmPreviousName, SteamRoots: st.LauncherRoots[game.LauncherSteam]}, nil
}

// queueUpdates queues the profile's updates: the named ones, or every one that needs no choice. beforePlay is the
// unattended update before a launch, which leaves out an update whose author asks the player to read a caution
// first and is content with nothing to do.
func (s *Services) queueUpdates(ctx context.Context, gameID, id string, ids []string, all, beforePlay bool) (QueuedUpdates, []queue.Item, error) {
	r, err := s.Problems.Updates(ctx, gameID, id)
	if err != nil {
		return QueuedUpdates{}, nil, err
	}
	var reqs []queue.Request
	for _, u := range pickUpdates(r.Updates, ids, all, beforePlay) {
		reqs = append(reqs, u.Request(gameID, id))
	}
	if len(reqs) == 0 {
		switch {
		case beforePlay:
			return QueuedUpdates{}, nil, nil
		case all || len(ids) == 0:
			return QueuedUpdates{}, nil, usererr.New(usererr.NotFound, "no updates available")
		}
		return QueuedUpdates{}, nil, usererr.New(usererr.NotFound, "no update available for "+strings.Join(ids, ", "))
	}
	var out QueuedUpdates
	if all {
		batch, err := s.Profiles.BeginUpdateBatch(gameID, id)
		if err != nil {
			return QueuedUpdates{}, nil, err
		}
		out.Before = batch.Before
		for i := range reqs {
			reqs[i].BatchID = batch.Batch
		}
	}
	items, err := s.Queue.Add(ctx, reqs)
	if err != nil {
		return QueuedUpdates{}, nil, err
	}
	out.Queued, out.Queue = len(reqs), s.Queue.State()
	return out, items, nil
}

// pickUpdates is the updates a queue request takes: the named ones (or every one with all or no names) that may be
// queued, leaving out before Play an update whose author asks the player to read a note first.
func pickUpdates(updates []problems.Update, ids []string, all, beforePlay bool) []problems.Update {
	var out []problems.Update
	for _, u := range updates {
		named := slices.ContainsFunc(ids, func(typed string) bool { return idMatches(typed, u.ID) })
		if !queueableUpdate(u, named) || (!all && len(ids) > 0 && !named) || (beforePlay && strings.TrimSpace(u.CautionMessage) != "") {
			continue
		}
		out = append(out, u)
	}
	return out
}

// updatesWait is how long a launch waits for the updates it queued.
var updatesWait = 5 * time.Minute

// updatesSettled reports whether every queued update has ended, and the failure of the first that did not finish. A
// download a free Nexus account must click through is handed to the player and does not hold the game back.
func updatesSettled(items []queue.Item, ids []string) (bool, error) {
	for _, id := range ids {
		i := slices.IndexFunc(items, func(it queue.Item) bool { return it.ID == id })
		if i < 0 {
			continue
		}
		switch it := items[i]; it.State {
		case queue.StateDone, queue.StateWaitingClick:
		case queue.StateQueued, queue.StateDownloading, queue.StateInstalling:
			return false, nil
		default:
			return true, errors.New(cmp.Or(it.Error, "the update for "+cmp.Or(it.Name, "a mod")+" did not finish ("+it.State+")"))
		}
	}
	return true, nil
}

// updateBeforePlay updates the profile's mods as the window does before Play, and waits for them.
func (s *Services) updateBeforePlay(ctx context.Context, gameID, id string) error {
	_, items, err := s.queueUpdates(ctx, gameID, id, nil, true, true)
	if err != nil || len(items) == 0 {
		return err
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	deadline := time.Now().Add(updatesWait)
	for {
		if done, err := updatesSettled(s.Queue.State().Items, ids); done {
			return err
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for mod updates")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func queueableUpdate(u problems.Update, named bool) bool {
	return !u.Unofficial && !u.PickFile && !u.NotDistributable && (!u.Switch || named)
}

// QueuedUpdates is what updates.queue queued. Before is the restore point Update all took, which a revert to undoes the
// whole batch; it is empty when only named mods were queued.
type QueuedUpdates struct {
	Queued int         `json:"queued"`
	Before string      `json:"before,omitempty"`
	Queue  queue.State `json:"queue"`
}
