package control

import (
	"context"
	"errors"
)

var errUnavailable = errors.New("that service is not running")

var libraryGlobal = map[string]globalMethod{
	"templates": func(s *Services, _ context.Context, p Params) (any, error) { return s.templatesMethod("templates", p) },
	"templates.save": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesMethod("templates.save", p)
	},
	"templates.delete": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesMethod("templates.delete", p)
	},
	"templates.new": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.templatesMethod("templates.new", p)
	},
	"history.usage": func(s *Services, _ context.Context, p Params) (any, error) { return s.Profiles.HistoryUsage(p.Game) },
	"history.trim": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.profileCall(p, func(id string) (any, error) { return s.Profiles.TrimHistory(p.Game, id, p.Keep) })
	},
	"backups.usage": func(s *Services, _ context.Context, p Params) (any, error) { return s.backupSize("backups.usage", p) },
	"backups.trim":  func(s *Services, _ context.Context, p Params) (any, error) { return s.backupSize("backups.trim", p) },
	"library.extra": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.Profiles.ExtraFolderMods(ctx, p.Game)
	},
	"library.hidden": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.profileCall(p, func(id string) (any, error) { return s.Profiles.DotHiddenMods(p.Game, id, p.Key) })
	},
	"library.old-files": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.profileCall(p, func(id string) (any, error) { return s.Profiles.PendingOldFiles(p.Game, id) })
	},
	"library.old-files.resolve": func(s *Services, _ context.Context, p Params) (any, error) {
		return s.profileCall(p, func(id string) (any, error) {
			return nil, s.Profiles.ResolveOldFiles(p.Game, id, p.Key, p.Set)
		})
	},
	"library.strays": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.Profiles.NewGameModsFolders(ctx, p.Game)
	},
	"library.strays.move": func(s *Services, ctx context.Context, p Params) (any, error) {
		return s.profileCall(p, func(id string) (any, error) {
			return s.changed(p.Game, func() (any, error) { return s.Profiles.MoveGameModsFolders(ctx, p.Game, id, p.IDs) })
		})
	},
	"library.strays.dismiss": func(s *Services, _ context.Context, p Params) (any, error) {
		return nil, s.Profiles.DismissGameModsFolder(p.Game, p.Name)
	},
	"queue.retry-failed": func(s *Services, ctx context.Context, _ Params) (any, error) { return s.Queue.RetryAllFailed(ctx) },
	"data.location": func(s *Services, _ context.Context, _ Params) (any, error) {
		if s.Data == nil {
			return nil, errUnavailable
		}
		return s.Data.DataLocation()
	},
	"archive.preview": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Archives == nil {
			return nil, errUnavailable
		}
		return s.Archives.ArchivePreview(p.Path)
	},
	"archive.downloads": func(s *Services, _ context.Context, p Params) (any, error) {
		if s.Archives == nil {
			return nil, errUnavailable
		}
		return s.Archives.DownloadsArchives(p.Game)
	},
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
