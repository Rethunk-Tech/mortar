package control

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/share"
	"github.com/Rethunk-AI/mortar/internal/tools"
)

// ChangedEvent tells the window a profile changed outside it, so it reloads that game's profiles.
const ChangedEvent = "profiles:changed"

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
	// Emit is nil in tests that do not watch events.
	Emit func(name string, data any)
}

// GameRow is one supported game for `mortar games`.
type GameRow struct {
	game.GameInfo
	Configured bool `json:"configured"`
	Profiles   int  `json:"profiles"`
}

// ModRow is one mod of a profile.
type ModRow struct {
	UniqueID string `json:"uniqueId"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Author   string `json:"author"`
	Enabled  bool   `json:"enabled"`
	Pinned   bool   `json:"pinned"`
	Key      string `json:"key"`
	Source   string `json:"source"`
}

// ModInfo is one mod with what relates to it.
type ModInfo struct {
	ModRow
	Needs      []string                 `json:"needs"`
	Optional   []string                 `json:"optional"`
	Dependents []string                 `json:"dependents"`
	Missing    []problems.Missing       `json:"missing"`
	Conflicts  []problems.AssetConflict `json:"conflicts"`
	Settings   []problems.SettingHint   `json:"settings"`
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

// Handle runs one method against the live services.
func (s *Services) Handle(ctx context.Context, method string, p Params) (any, error) {
	switch method {
	case "games":
		return s.games()
	case "profiles":
		return s.Profiles.List(p.Game)
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
	case "status":
		return s.Launches.Status(p.Game)
	case "stop":
		if _, err := s.Launches.Status(p.Game); err != nil {
			return nil, err
		}
		if err := s.Launches.Stop(p.Game); err != nil {
			return nil, err
		}
		return s.Launches.Status(p.Game)
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
	case "profile.history":
		events, err := s.Profiles.History(p.Game, id)
		if err != nil {
			return nil, err
		}
		out := make([]HistoryRow, 0, len(events))
		for _, event := range events {
			out = append(out, HistoryRow{ID: event.ID, At: event.At, Kind: event.Kind, Summary: event.Label})
		}
		return out, nil
	case "profile.revert":
		return s.changed(p.Game, func() (any, error) { return s.Profiles.Revert(p.Game, id, p.Name) })
	case "profile.delete":
		return s.changed(p.Game, func() (any, error) { return Removed{Mods: []string{prof.Name}}, s.Profiles.Delete(p.Game, id) })
	case "mods":
		return modRows(prof), nil
	case "mod":
		return s.modInfo(ctx, p.Game, prof, p.UniqueIDs)
	case "mods.enable", "mods.disable":
		refs, err := refsFor(prof, p.UniqueIDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			return s.Profiles.SetModsEnabled(p.Game, id, refs, method == "mods.enable")
		})
	case "mods.pin", "mods.unpin":
		keys, err := keysFor(prof, p.UniqueIDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) {
			for _, k := range keys {
				if _, err := s.Profiles.SetPinned(p.Game, id, k, method == "mods.pin"); err != nil {
					return nil, err
				}
			}
			return modRows(s.reload(p.Game, id, prof)), nil
		})
	case "mods.remove":
		keys, err := keysFor(prof, p.UniqueIDs)
		if err != nil {
			return nil, err
		}
		return s.changed(p.Game, func() (any, error) { return s.remove(p.Game, id, prof, keys) })
	case "install":
		return s.changed(p.Game, func() (any, error) { return s.install(p.Game, id, p.Path) })
	case "conflicts":
		res, err := s.Problems.Problems(ctx, p.Game, id)
		if err != nil {
			return nil, err
		}
		out := []problems.AssetConflict{}
		for _, c := range res.AssetConflicts {
			if p.All || !c.Cosmetic {
				out = append(out, c)
			}
		}
		return out, nil
	case "problems":
		return s.Problems.Problems(ctx, p.Game, id)
	case "updates":
		return s.Problems.Updates(ctx, p.Game, id)
	case "share":
		res, err := share.Encode(prof)
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
	case "saves":
		return s.Saves.Saves(ctx, p.Game, id)
	case "launch":
		return s.launch(ctx, p.Game, id)
	}
	return nil, fmt.Errorf("unknown method %q", method)
}

// HistoryRow is the compact history item shown by the CLI.
type HistoryRow struct {
	ID      string    `json:"id"`
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Summary string    `json:"summary"`
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
		out = append(out, GameRow{GameInfo: g, Configured: g.Installed && g.InstallDir != "", Profiles: len(ps)})
	}
	return out, nil
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
		if strings.EqualFold(p.Name, sel) {
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
				UniqueID: m.UniqueID, Name: m.Name, Version: m.Version, Author: m.Author, Key: e.Key,
				Enabled: !slices.ContainsFunc(e.Disabled, func(d string) bool { return strings.EqualFold(d, m.UniqueID) }),
				Pinned:  e.Pinned, Source: source(e.Source),
			})
		}
	}
	return out
}

// entryOf finds the entry holding a UniqueID, ignoring case.
func entryOf(p profile.Profile, uniqueID string) (profile.Entry, bool) {
	for _, e := range p.Entries {
		if slices.ContainsFunc(e.Mods, func(m profile.EntryMod) bool { return strings.EqualFold(m.UniqueID, uniqueID) }) {
			return e, true
		}
	}
	return profile.Entry{}, false
}

func refsFor(p profile.Profile, ids []string) ([]profile.EnableRef, error) {
	if len(ids) == 0 {
		return nil, errors.New("name at least one mod by UniqueID")
	}
	refs := make([]profile.EnableRef, 0, len(ids))
	for _, id := range ids {
		e, ok := entryOf(p, id)
		if !ok {
			return nil, fmt.Errorf("profile %s has no mod %q", p.Name, id)
		}
		refs = append(refs, profile.EnableRef{Key: e.Key, UniqueID: id})
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

func (s *Services) install(gameID, id, path string) (InstallOutcome, error) {
	if _, err := os.Stat(path); err != nil {
		return InstallOutcome{}, err
	}
	res, err := s.Profiles.InstallArchive(gameID, id, path)
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
	if out.Added == nil {
		out.Added = []string{}
	}
	return out, nil
}

func (s *Services) modInfo(ctx context.Context, gameID string, p profile.Profile, ids []string) (ModInfo, error) {
	if len(ids) != 1 {
		return ModInfo{}, errors.New("name one mod by UniqueID")
	}
	uid := ids[0]
	e, ok := entryOf(p, uid)
	if !ok {
		return ModInfo{}, fmt.Errorf("profile %s has no mod %q", p.Name, uid)
	}
	info := ModInfo{Needs: []string{}, Optional: []string{}, Dependents: []string{}, Missing: []problems.Missing{}, Conflicts: []problems.AssetConflict{}, Settings: []problems.SettingHint{}}
	for _, r := range modRows(p) {
		if strings.EqualFold(r.UniqueID, uid) {
			info.ModRow = r
		}
	}
	for _, m := range e.Mods {
		if strings.EqualFold(m.UniqueID, uid) {
			info.Optional = append(info.Optional, m.Optional...)
			for _, n := range m.Needs {
				if !slices.ContainsFunc(m.Optional, func(o string) bool { return strings.EqualFold(o, n) }) {
					info.Needs = append(info.Needs, n)
				}
			}
		}
	}
	for _, other := range p.Entries {
		for _, m := range other.Mods {
			if slices.ContainsFunc(m.Needs, func(n string) bool { return strings.EqualFold(n, uid) }) {
				info.Dependents = append(info.Dependents, m.UniqueID)
			}
		}
	}
	res, err := s.Problems.Problems(ctx, gameID, p.ID)
	if err != nil {
		return info, err
	}
	for _, m := range res.Missing {
		if strings.EqualFold(m.DependentID, uid) {
			info.Missing = append(info.Missing, m)
		}
	}
	for _, c := range res.AssetConflicts {
		if slices.ContainsFunc(c.PackIDs, func(id string) bool { return strings.EqualFold(id, uid) }) {
			info.Conflicts = append(info.Conflicts, c)
		}
	}
	for _, h := range res.Settings {
		if strings.EqualFold(h.UniqueID, uid) {
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
	skipped, err := share.Write(f, p, modsDir)
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

// launchWait bounds how long launch waits for the game to leave Launching.
const launchWait = 3 * time.Minute

func (s *Services) launch(ctx context.Context, gameID, id string) (launchsvc.Status, error) {
	if err := s.Launches.Start(ctx, gameID, id, false); err != nil {
		return launchsvc.Status{}, err
	}
	deadline := time.Now().Add(launchWait)
	for {
		st, err := s.Launches.Status(gameID)
		if err != nil {
			return st, err
		}
		switch st.State {
		case launchsvc.Running, launchsvc.Idle:
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
