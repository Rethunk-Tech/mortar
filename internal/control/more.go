package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/tools"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

var errNoLinks = errors.New("link handling is unavailable")

// handleMore answers the bundle, template, tool, LAN, data-folder, update and link methods; ok is false for any
// other method.
func (s *Services) handleMore(ctx context.Context, method string, p Params) (res any, ok bool, err error) {
	switch method {
	case "bundles.create", "bundles.delete", "bundles.rename", "bundles.add", "bundles.remove":
		res, err = s.bundlesWrite(method, p)
	case "templates.apply", "templates.preview", "templates.undo", "templates.rename", "templates.restore":
		res, err = s.templatesWrite(method, p)
	case "tools.add", "tools.update", "tools.remove":
		res, err = s.toolsWrite(method, p)
	case "external.sources":
		res, err = s.externalSources(p)
	case "external.import":
		res, err = s.externalImport(ctx, p)
	case "lan.peers", "lan.send", "lan.inbox", "lan.accept", "lan.decline", "lan.paircode", "lan.pair", "lan.paired", "lan.unpair":
		res, err = s.lanMethod(ctx, method, p)
	case "data.move":
		res, err = s.dataMove(p)
	case "data.cleanup":
		res, err = s.dataCleanup(p)
	case "app.update.check":
		res, err = s.appUpdate(ctx, false)
	case "app.update.install":
		res, err = s.appUpdate(ctx, true)
	case "links.register":
		res, err = nil, s.withNxm(func() error { return s.Nxm.RegisterLinks() })
	case "links.enable":
		res, err = nil, s.withNxm(func() error { return s.Nxm.EnableSource(p.Source) })
	case "links.disable":
		res, err = nil, s.withNxm(func() error { return s.Nxm.DisableSource(p.Source) })
	case "problems.checkUpdates":
		res, err = s.profileCall(p, func(id string) (any, error) { return s.Problems.CheckUpdatesNow(ctx, p.Game, id) })
	case "support.diagnostics":
		res, err = s.supportDiagnostics(p)
	case "game.resetInstall":
		res, err = nil, s.Games.ResetInstall(p.Game)
	case "game.steamStatus":
		res = s.Games.SteamStatus()
	default:
		return nil, false, nil
	}
	return res, true, err
}

func (s *Services) bundlesWrite(method string, p Params) (any, error) {
	if s.Bundles == nil {
		return nil, errors.New("bundles are unavailable")
	}
	if method == "bundles.create" {
		return s.profileCall(p, func(id string) (any, error) { return s.Bundles.Create(p.Game, p.Name, id, typedIDs(p.IDs)) })
	}
	list, err := s.Bundles.List(p.Game)
	if err != nil {
		return nil, err
	}
	b, err := resolveBundle(list, p.Name)
	if err != nil {
		return nil, err
	}
	switch method {
	case "bundles.delete":
		return nil, s.Bundles.Delete(p.Game, b.ID)
	case "bundles.rename":
		return s.Bundles.Rename(p.Game, b.ID, p.Value)
	case "bundles.add":
		return s.profileCall(p, func(id string) (any, error) { return s.Bundles.AddMods(p.Game, b.ID, id, typedIDs(p.IDs)) })
	default:
		return s.Bundles.RemoveMods(p.Game, b.ID, typedIDs(p.IDs))
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
		return s.profileCall(p, func(id string) (any, error) { return nil, s.Lan.Send(ctx, p.Name, p.Game, id, share.OwnInclude()) })
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
		log.Printf("lan: importing %q from %s failed: %v", arrival.ProfileName, arrival.Sender, err)
		if errors.Is(err, sharesvc.ErrSignedOut) {
			err = usererr.Wrap(usererr.Invalid, err)
		}
		return sharesvc.Result{}, err
	}
	log.Printf("lan: imported %q from %s: placed %d mods, queued %d downloads", arrival.ProfileName, arrival.Sender, len(res.Profile.Entries), res.Queued)
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
