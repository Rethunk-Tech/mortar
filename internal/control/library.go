package control

import (
	"context"
	"errors"
)

var errUnavailable = errors.New("that service is not running")

// handleLibrary answers the methods that manage templates, history and backup size, library cleanups, the data
// folder and archives; ok is false for any other method.
func (s *Services) handleLibrary(ctx context.Context, method string, p Params) (res any, ok bool, err error) {
	switch method {
	case "templates", "templates.save", "templates.delete", "templates.new":
		res, err = s.templatesMethod(method, p)
	case "history.usage":
		res, err = s.Profiles.HistoryUsage(p.Game)
	case "history.trim":
		res, err = s.profileCall(p, func(id string) (any, error) { return s.Profiles.TrimHistory(p.Game, id, p.Keep) })
	case "backups.usage", "backups.trim":
		res, err = s.backupSize(method, p)
	case "library.extra":
		res, err = s.Profiles.ExtraFolderMods(p.Game)
	case "library.hidden":
		res, err = s.profileCall(p, func(id string) (any, error) { return s.Profiles.DotHiddenMods(p.Game, id, p.Key) })
	case "library.old-files":
		res, err = s.profileCall(p, func(id string) (any, error) { return s.Profiles.PendingOldFiles(p.Game, id) })
	case "library.old-files.resolve":
		res, err = s.profileCall(p, func(id string) (any, error) {
			return nil, s.Profiles.ResolveOldFiles(p.Game, id, p.Key, p.Set)
		})
	case "library.strays":
		res, err = s.Profiles.NewGameModsFolders(p.Game)
	case "library.strays.move":
		res, err = s.profileCall(p, func(id string) (any, error) {
			return s.changed(p.Game, func() (any, error) { return s.Profiles.MoveGameModsFolders(p.Game, id, p.IDs) })
		})
	case "library.strays.dismiss":
		res, err = nil, s.Profiles.DismissGameModsFolder(p.Game, p.Name)
	case "queue.retry-failed":
		res, err = s.Queue.RetryAllFailed(ctx)
	case "data.location":
		if s.Data == nil {
			return nil, true, errUnavailable
		}
		res, err = s.Data.DataLocation()
	case "archive.preview":
		if s.Archives == nil {
			return nil, true, errUnavailable
		}
		res, err = s.Archives.ArchivePreview(p.Path)
	case "archive.downloads":
		if s.Archives == nil {
			return nil, true, errUnavailable
		}
		res, err = s.Archives.DownloadsArchives(p.Game)
	default:
		return nil, false, nil
	}
	return res, true, err
}

func (s *Services) profileCall(p Params, fn func(id string) (any, error)) (any, error) {
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	return fn(prof.ID)
}

func (s *Services) templatesMethod(method string, p Params) (any, error) {
	if s.Templates == nil {
		return nil, errUnavailable
	}
	switch method {
	case "templates":
		return s.Templates.Templates(p.Game)
	case "templates.save":
		return s.profileCall(p, func(id string) (any, error) { return s.Templates.SaveTemplateFromProfile(p.Game, id, p.Name) })
	case "templates.delete":
		return nil, s.Templates.DeleteTemplate(p.Game, p.Name)
	default:
		return s.changed(p.Game, func() (any, error) { return s.Templates.NewProfileFromTemplate(p.Game, p.Name, p.Value) })
	}
}

func (s *Services) backupSize(method string, p Params) (any, error) {
	if s.Saves == nil {
		return nil, errors.New("backups are unavailable")
	}
	if method == "backups.usage" {
		return s.Saves.BackupsUsage(p.Game)
	}
	return s.Saves.TrimBackups(p.Game, p.Keep)
}
