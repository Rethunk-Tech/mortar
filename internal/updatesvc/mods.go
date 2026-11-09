package updatesvc

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/desktopnotify"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const modUpdateInitialDelay = 5 * time.Minute

// modUpdateEvery is the wait between background checks: the player's update check interval, read before each wait
// so a change takes effect at the next check.
func modUpdateEvery(set settings.Settings) time.Duration {
	minutes := set.UpdateCheckIntervalMinutes
	if minutes < settings.MinUpdateCheckIntervalMinutes || minutes > settings.MaxUpdateCheckIntervalMinutes {
		minutes = settings.DefaultUpdateCheckIntervalMinutes
	}
	return time.Duration(minutes) * time.Minute
}

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
				slog.Warn("mod update check", "err", err)
				return
			}
			keys, summary := DigestFromProfiles(profiles)
			slog.Info("mod update check: profiles with updates", "profiles", len(profiles), "updates", summary.TotalUpdates, "withUpdates", summary.ProfilesWith)
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
				slog.Warn("mod update digest save", "err", err)
				return
			}
			if should && summary.TotalUpdates > 0 {
				notify(summary)
				desktopnotify.SendIf(
					desktopnotify.Pref("desktopModUpdates"),
					desktopnotify.KindModUpdates,
					i18n.T("Mod updates available: {count}", "count", strconv.Itoa(summary.TotalUpdates)),
					"",
					map[string]any{"game": summary.Game, "profile": summary.ProfileID},
				)
			}
		}
		for {
			run()
			timer.Reset(modUpdateEvery(svc.Get()))
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()
}
