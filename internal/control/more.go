package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/tools"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

var errNoLinks = errors.New("link handling is unavailable")

var moreGlobal = map[string]globalMethod{
	"groups.create": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.savedGroupsWrite("groups.create", p)
	},
	"groups.delete": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.savedGroupsWrite("groups.delete", p)
	},
	"groups.rename": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.savedGroupsWrite("groups.rename", p)
	},
	"groups.add": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.savedGroupsWrite("groups.add", p)
	},
	"groups.remove": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.savedGroupsWrite("groups.remove", p)
	},
	"templates.apply": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesWrite("templates.apply", p)
	},
	"templates.preview": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesWrite("templates.preview", p)
	},
	"templates.undo": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesWrite("templates.undo", p)
	},
	"templates.rename": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesWrite("templates.rename", p)
	},
	"templates.restore": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesWrite("templates.restore", p)
	},
	"tools.add":        func(s *Services, _ context.Context, p Params) (any, error) { return s.toolsWrite("tools.add", p) },
	"tools.update":     func(s *Services, _ context.Context, p Params) (any, error) { return s.toolsWrite("tools.update", p) },
	"tools.remove":     func(s *Services, _ context.Context, p Params) (any, error) { return s.toolsWrite("tools.remove", p) },
	"external.sources": func(s *Services, _ context.Context, p Params) (any, error) { return s.externalSources(p) },
	"external.import":  func(s *Services, ctx context.Context, p Params) (any, error) { return s.externalImport(ctx, p) },
	"lan.peers":        func(s *Services, ctx context.Context, p Params) (any, error) { return s.lanMethod(ctx, "lan.peers", p) },
	"lan.send":         func(s *Services, ctx context.Context, p Params) (any, error) { return s.lanMethod(ctx, "lan.send", p) },
	"lan.inbox":        func(s *Services, ctx context.Context, p Params) (any, error) { return s.lanMethod(ctx, "lan.inbox", p) },
	"lan.accept": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.lanMethod(ctx, "lan.accept", p)
	},
	"lan.decline": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.lanMethod(ctx, "lan.decline", p)
	},
	"lan.paircode": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.lanMethod(ctx, "lan.paircode", p)
	},
	"lan.pair": func(s *Services, ctx context.Context, p Params) (any, error) { return s.lanMethod(ctx, "lan.pair", p) },
	"lan.paired": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.lanMethod(ctx, "lan.paired", p)
	},
	"lan.unpair": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.lanMethod(ctx, "lan.unpair", p)
	},
	"data.move":          func(s *Services, _ context.Context, p Params) (any, error) { return s.dataMove(p) },
	"data.cleanup":       func(s *Services, _ context.Context, p Params) (any, error) { return s.dataCleanup(p) },
	"app.update.check":   func(s *Services, ctx context.Context, _ Params) (any, error) { return s.appUpdate(ctx, false) },
	"app.update.install": func(s *Services, ctx context.Context, _ Params) (any, error) { return s.appUpdate(ctx, true) },
	"links.register": func(s *Services, _ context.Context, _ Params) (any, error) {
		return nil, s.withNxm(func() error { return s.Nxm.RegisterLinks() })
	},
	"links.enable": func(s *Services, _ context.Context, p Params) (any, error) {
		return nil, s.withNxm(func() error { return s.Nxm.EnableSource(p.Source) })
	},
	"links.disable": func(s *Services, _ context.Context, p Params) (any, error) {
		return nil, s.withNxm(func() error { return s.Nxm.DisableSource(p.Source) })
	},
	"links": func(s *Services, _ context.Context, _ Params) (any, error) {
		if s.Nxm == nil {
			return nil, errNoLinks
		}
		return s.Nxm.ListOwners(), nil
	},
	"problems.checkUpdates": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.profileCall(p, func(id string) (any, error) { return s.Problems.CheckUpdatesNow(ctx, p.Game, id) })
	},
	"support.diagnostics": func(s *Services, _ context.Context, p Params) (any, error) { return s.supportDiagnostics(p) },
	"game.resetInstall":   func(s *Services, _ context.Context, p Params) (any, error) { return nil, s.Games.ResetInstall(p.Game) },
	"game.steamStatus":    func(s *Services, _ context.Context, _ Params) (any, error) { return s.Games.SteamStatus(), nil },
}

// resolveIDs is the profile p names and the mods p.IDs pick from it.
func (s *Services) resolveIDs(p Params) (profile.Profile, []mod.ID, error) {
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return profile.Profile{}, nil, err
	}
	ids, err := typedIDs(modIDs(prof), p.IDs)
	return prof, ids, err
}

func (s *Services) savedGroupsWrite(method string, p Params) (any, error) {
	if s.SavedGroups == nil {
		return nil, errors.New("saved groups are unavailable")
	}
	if method == "groups.create" {
		prof, ids, err := s.resolveIDs(p)
		if err != nil {
			return nil, err
		}
		return s.SavedGroups.Create(p.Game, p.Name, prof.ID, ids)
	}
	list, err := s.SavedGroups.List(p.Game)
	if err != nil {
		return nil, err
	}
	g, err := resolveSavedGroup(list, p.Name)
	if err != nil {
		return nil, err
	}
	switch method {
	case "groups.delete":
		return nil, s.SavedGroups.Delete(p.Game, g.ID)
	case "groups.rename":
		return s.SavedGroups.Rename(p.Game, g.ID, p.Value)
	case "groups.add":
		prof, ids, err := s.resolveIDs(p)
		if err != nil {
			return nil, err
		}
		return s.SavedGroups.AddMods(p.Game, g.ID, prof.ID, ids)
	default:
		members := make([]mod.ID, len(g.Mods))
		for i, m := range g.Mods {
			members[i] = m.ID
		}
		ids, err := typedIDs(members, p.IDs)
		if err != nil {
			return nil, err
		}
		return s.SavedGroups.RemoveMods(p.Game, g.ID, ids)
	}
}

func (s *Services) templatesWrite(method string, p Params) (any, error) {
	if s.Templates == nil {
		return nil, errUnavailable
	}
	switch method {
	case "templates.apply":
		return s.profileCall(p, func(id string) (any, error) {
			return s.changed(p.Game, func() (any, error) { return s.Templates.ApplyTemplate(p.Game, p.Name, id) })
		})
	case "templates.preview":
		return s.profileCall(p, func(id string) (any, error) { return s.Templates.PreviewApplyTemplate(p.Game, p.Name, id) })
	case "templates.undo":
		var u templates.Undo
		if err := json.Unmarshal(p.Body, &u); err != nil {
			return nil, fmt.Errorf("undo data: %w", err)
		}
		return s.profileCall(p, func(id string) (any, error) {
			return s.changed(p.Game, func() (any, error) { return s.Templates.UndoApplyTemplate(p.Game, id, u) })
		})
	case "templates.rename":
		return nil, s.Templates.RenameTemplate(p.Game, p.Name, p.Value)
	default:
		var t templates.Template
		if err := json.Unmarshal(p.Body, &t); err != nil {
			return nil, fmt.Errorf("template data: %w", err)
		}
		return nil, s.Templates.RestoreTemplate(p.Game, t)
	}
}

// toolsWrite: Key is the tool id, Name its label, Path its executable, IDs its arguments and Value its folder.
func (s *Services) toolsWrite(method string, p Params) (any, error) {
	if s.Tools == nil {
		return nil, errUnavailable
	}
	t := tools.Tool{ID: p.Key, Name: p.Name, Executable: p.Path, Arguments: p.IDs, WorkingDir: p.Value}
	switch method {
	case "tools.add":
		return s.Tools.Add(p.Game, t)
	case "tools.update":
		return nil, s.Tools.Update(p.Game, t)
	default:
		return nil, s.Tools.Remove(p.Game, p.Key)
	}
}

func (s *Services) lanMethod(ctx context.Context, method string, p Params) (any, error) {
	if s.Lan == nil {
		return nil, errors.New("LAN sharing is unavailable")
	}
	switch method {
	case "lan.peers":
		return s.Lan.Peers(ctx)
	case "lan.send":
		return s.profileCall(p, func(id string) (any, error) {
			return nil, s.Lan.Send(ctx, p.Name, p.Game, id, share.IncludeOf(s.Settings.Get()))
		})
	case "lan.inbox":
		return s.Lan.Pending(), nil
	case "lan.paircode":
		return s.Lan.PairCode(ctx)
	case "lan.pair":
		return nil, s.Lan.Pair(ctx, p.Name, p.Value)
	case "lan.paired":
		return s.Lan.Paired()
	case "lan.unpair":
		return nil, s.Lan.Unpair(p.Name)
	}
	id, err := strconv.Atoi(p.Name)
	if err != nil {
		return nil, fmt.Errorf("transfer id %q is not a number", p.Name)
	}
	if method == "lan.decline" {
		s.Lan.Dismiss(id)
		return struct{}{}, nil
	}
	return s.lanAccept(ctx, id)
}

// lanAccept takes a waiting share as the window's accept does: a paired sender's files come over first, then the
// profile is imported with the import dialog's defaults.
func (s *Services) lanAccept(ctx context.Context, id int) (sharesvc.Result, error) {
	pending := s.Lan.Pending()
	i := slices.IndexFunc(pending, func(a lan.Arrival) bool { return a.ID == id })
	if i < 0 {
		return sharesvc.Result{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("no waiting share %d", id))
	}
	arrival := pending[i]
	if s.Shares == nil {
		return sharesvc.Result{}, errUnavailable
	}
	if arrival.Paired {
		if err := s.Lan.Transfer(ctx, id); err != nil {
			return sharesvc.Result{}, err
		}
	}
	res, err := s.Shares.ImportData(ctx, arrival.Game, arrival.Payload)
	if err != nil {
		slog.Warn("lan: importing failed", "profile", arrival.ProfileName, "sender", arrival.Sender, "err", err)
		if errors.Is(err, sharesvc.ErrSignedOut) {
			err = usererr.Wrap(usererr.Invalid, err)
		}
		return sharesvc.Result{}, err
	}
	slog.Info("lan: imported a profile", "profile", arrival.ProfileName, "sender", arrival.Sender, "mods", len(res.Profile.Entries), "downloads", res.Queued)
	s.Lan.Dismiss(id)
	return res, nil
}

func (s *Services) dataMove(p Params) (any, error) {
	if s.Data == nil {
		return nil, errUnavailable
	}
	if p.Preview {
		return s.Data.MoveDataFolderPreview(p.Path)
	}
	return nil, s.Data.MoveDataFolder(p.Path)
}

func (s *Services) dataCleanup(p Params) (any, error) {
	if s.Data == nil {
		return nil, errUnavailable
	}
	preview, err := s.Data.CleanupPreview()
	if err != nil || p.Preview {
		return preview, err
	}
	return preview, s.Data.Cleanup(preview)
}

func (s *Services) appUpdate(ctx context.Context, install bool) (any, error) {
	if s.Updates == nil {
		return nil, errors.New("updates are unavailable")
	}
	if install {
		return nil, s.Updates.Install(ctx)
	}
	return s.Updates.Check(ctx)
}

func (s *Services) withNxm(fn func() error) error {
	if s.Nxm == nil {
		return errNoLinks
	}
	return fn()
}

// supportDiagnostics writes the diagnostics zip to Path; Profile, when named, adds that profile's latest log.
func (s *Services) supportDiagnostics(p Params) (any, error) {
	if s.Support == nil {
		return nil, errors.New("diagnostics are unavailable")
	}
	profileID := ""
	if p.Profile != "" {
		prof, err := s.resolve(p.Game, p.Profile)
		if err != nil {
			return nil, err
		}
		profileID = prof.ID
	}
	path, err := s.Support.WriteDiagnostics(p.Game, profileID, p.Path)
	return map[string]string{"path": path}, err
}
