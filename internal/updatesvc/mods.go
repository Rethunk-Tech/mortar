package updatesvc

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	modUpdateInitialDelay = 5 * time.Minute
	modUpdateInterval     = 4 * time.Hour
)

type ModUpdate struct {
	Game        string
	ProfileID   string
	ProfileName string
	Count       int
}

type ModUpdateSource interface {
	ModUpdates(context.Context) ([]ModUpdate, error)
}

type ModUpdateNotifier interface {
	Notify(ModUpdate)
}

type ModUpdateSetting interface {
	ModUpdatesEnabled() bool
}

type modUpdateState struct {
	notified map[string]int
}

func decideModUpdateNotification(state *modUpdateState, updates []ModUpdate) (ModUpdate, bool) {
	if state.notified == nil {
		state.notified = make(map[string]int)
	}
	var notify ModUpdate
	found := false
	for _, update := range updates {
		key := update.Game + "\x00" + update.ProfileID
		if update.Count > state.notified[key] {
			if !found || update.Count > notify.Count {
				notify = update
				found = true
			}
		}
		state.notified[key] = update.Count
	}
	return notify, found
}

func StartModBackground(ctx context.Context, enabled ModUpdateSetting, source ModUpdateSource, notify ModUpdateNotifier) {
	go func() {
		timer := time.NewTimer(modUpdateInitialDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		state := &modUpdateState{}
		run := func() {
			if !enabled.ModUpdatesEnabled() {
				return
			}
			updates, err := source.ModUpdates(ctx)
			if err != nil {
				log.Printf("mod update check: %v", err)
				return
			}
			if update, ok := decideModUpdateNotification(state, updates); ok {
				notify.Notify(update)
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

func ModUpdateNotification(update ModUpdate) (title, body string) {
	label := "mod updates"
	if update.Count == 1 {
		label = "mod update"
	}
	return "Mod updates available", fmt.Sprintf("%d %s available for %s", update.Count, label, update.ProfileName)
}
