package updatesvc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Rethunk-AI/mortar/internal/desktopnotify"
)

const (
	modUpdateInitialDelay = 5 * time.Minute
	modUpdateInterval     = 4 * time.Hour
)

type ModUpdateSetting interface {
	ModUpdatesEnabled() bool
}

type ModUpdateSource interface {
	ProfileModUpdates(context.Context) ([]ProfileModUpdates, error)
}

type ModDigestSettings interface {
	UpdateDigestMode() string
	LastModUpdateDigest() []string
	LastModUpdateDigestAt() string
}

type ModDigestStore interface {
	ModDigestSettings
	SaveModUpdateDigest(keys []string, at string) error
}

type ModUpdateDigestNotifier interface {
	NotifyDigest(ModUpdateDigestNotice)
}

func StartModBackground(ctx context.Context, enabled ModUpdateSetting, source ModUpdateSource, store ModDigestStore, notify ModUpdateDigestNotifier) {
	go func() {
		timer := time.NewTimer(modUpdateInitialDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		run := func() {
			if !enabled.ModUpdatesEnabled() {
				return
			}
			profiles, err := source.ProfileModUpdates(ctx)
			if err != nil {
				log.Printf("mod update check: %v", err)
				return
			}
			keys, summary := DigestFromProfiles(profiles)
			now := time.Now()
			should, persist := DecideUpdateDigest(
				store.UpdateDigestMode(),
				store.LastModUpdateDigest(),
				keys,
				store.LastModUpdateDigestAt(),
				now,
			)
			at := ""
			if should {
				at = now.Format(time.RFC3339)
			}
			if err := store.SaveModUpdateDigest(persist, at); err != nil {
				log.Printf("mod update digest save: %v", err)
				return
			}
			if should && summary.TotalUpdates > 0 {
				notify.NotifyDigest(summary)
				desktopnotify.SendIf(
					desktopnotify.Pref("desktopModUpdates"),
					fmt.Sprintf("%d mod updates available", summary.TotalUpdates),
					"",
					map[string]any{"game": summary.Game, "profile": summary.ProfileID},
				)
			}
		}
		run()
		ticker := time.NewTicker(modUpdateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
