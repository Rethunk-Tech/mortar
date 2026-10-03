package main

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/updatesvc"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func startModUpdateBackground(
	ctx context.Context,
	svc *settings.Service,
	gamesSvc *game.Service,
	profiles *profile.Store,
	problemsSvc *problems.Service,
	app *application.App,
) {
	updatesvc.StartModBackground(
		ctx,
		modUpdateSettingFunc(func() bool {
			check := svc.Get().CheckModUpdatesOnStart
			return check != nil && *check
		}),
		modUpdateScanSource{gamesSvc: gamesSvc, profiles: profiles, problemsSvc: problemsSvc},
		updatesvc.SettingsDigestStore{Svc: svc},
		modUpdateDigestNotifierFunc(func(notice updatesvc.ModUpdateDigestNotice) {
			app.Event.Emit(updatesvc.ModUpdateDigestEvent, notice)
		}),
	)
}

type modUpdateScanSource struct {
	gamesSvc    *game.Service
	profiles    *profile.Store
	problemsSvc *problems.Service
}

func (s modUpdateScanSource) ProfileModUpdates(ctx context.Context) ([]updatesvc.ProfileModUpdates, error) {
	games, err := s.gamesSvc.List()
	if err != nil {
		return nil, err
	}
	var out []updatesvc.ProfileModUpdates
	for _, g := range games {
		if !g.Available || !g.Installed {
			continue
		}
		all, err := s.profiles.List(g.ID)
		if err != nil {
			continue
		}
		for _, p := range all {
			if p.Error != "" || p.Hidden {
				continue
			}
			result, err := s.problemsSvc.Updates(ctx, g.ID, p.ID)
			if err != nil {
				continue
			}
			out = append(out, updatesvc.ProfileModUpdates{
				Game: g.ID, ProfileID: p.ID, ProfileName: p.Name, Updates: result.Updates,
			})
		}
	}
	return out, nil
}

type modUpdateDigestNotifierFunc func(updatesvc.ModUpdateDigestNotice)

func (f modUpdateDigestNotifierFunc) NotifyDigest(notice updatesvc.ModUpdateDigestNotice) {
	f(notice)
}
