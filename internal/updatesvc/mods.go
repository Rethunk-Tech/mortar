package updatesvc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Rethunk-AI/mortar/internal/desktopnotify"
	"github.com/Rethunk-AI/mortar/internal/settings"
)

const (
	modUpdateInitialDelay = 5 * time.Minute
	modUpdateInterval     = 4 * time.Hour
)

func StartModBackground(ctx context.Context, svc *settings.Service, source func(context.Context) ([]ProfileModUpdates, error), notify func(ModUpdateDigestNotice)) {
	go func() {
		timer := time.NewTimer(modUpdateInitialDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		run := func() {
			cur := svc.Get()
			if cur.CheckModUpdatesOnStart == nil || !*cur.CheckModUpdatesOnStart {
				return
			}
			profiles, err := source(ctx)
			if err != nil {
				log.Printf("mod update check: %v", err)
				return
			}
			keys, summary := DigestFromProfiles(profiles)
			now := time.Now()
			should, persist := DecideUpdateDigest(
				cur.UpdateDigest,
				cur.LastModUpdateDigest,
				keys,
				cur.LastModUpdateDigestAt,
				now,
			)
			at := ""
			if should {
				at = now.Format(time.RFC3339)
			}
			if err := svc.SetLastModUpdateDigest(persist, at); err != nil {
				log.Printf("mod update digest save: %v", err)
				return
			}
			if should && summary.TotalUpdates > 0 {
				notify(summary)
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
