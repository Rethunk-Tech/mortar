package main

import (
	"embed"
	"log"
	"os"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/loadersvc"
	"github.com/Rethunk-AI/mortar/internal/picker"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// registerEvents declares the custom events for the binding generator and the runtime's payload checks.
func registerEvents() {
	application.RegisterEvent[launchsvc.Status](launchsvc.StateEvent)
	application.RegisterEvent[launchsvc.Line](launchsvc.LineEvent)
	application.RegisterEvent[loadersvc.Progress](loadersvc.ProgressEvent)
	application.RegisterEvent[[]string](picker.DroppedEvent)
	application.RegisterEvent[settings.Settings](settings.ChangedEvent)
}

func main() {
	registerEvents()
	store, err := settings.Open()
	if err != nil {
		log.Fatal(err)
	}
	svc := settings.NewService(store)
	svc.ValidateGameFolder = game.ValidateFolder

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	gamesSvc := game.NewService(home, store)

	items, err := modstore.Open()
	if err != nil {
		log.Fatal(err)
	}
	if err := items.Cleanup(); err != nil {
		log.Printf("store cleanup: %v", err)
	}

	profiles, err := profile.Open(items)
	if err != nil {
		log.Fatal(err)
	}
	profiles.Bundled = loadersvc.BundledKey(store)
	loaders := loadersvc.NewService(home, store, items, profiles)
	launches := launchsvc.NewService(home, store, profiles)
	profiles.Running = launches.Running
	now := time.Now()
	if err := profiles.PurgeTrash(now); err != nil {
		log.Printf("purge trash: %v", err)
	}
	// An unreadable profile.json stops collection: its keys are unknown, and their items must not be deleted.
	if keys, err := profiles.StoreKeys(); err != nil {
		log.Printf("store collect skipped: %v", err)
	} else if err := items.Collect(keys, now); err != nil {
		log.Printf("store collect: %v", err)
	}

	var window *application.WebviewWindow
	pick := &picker.Service{}

	app := application.New(application.Options{
		Name:        "Mortar",
		Description: "Multi-game desktop mod manager",
		Services: []application.Service{
			application.NewService(svc), application.NewService(gamesSvc),
			application.NewService(profile.NewService(profiles)), application.NewService(loaders), application.NewService(launches), application.NewService(pick),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: game.ArtMiddleware(home),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "tech.rethunk.mortar",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				window.Restore()
				window.Focus()
			},
		},
	})

	svc.App = app
	loaders.App = app
	launches.App = app
	pick.App = app

	// Wails fixes BackgroundType at window creation, so the stored value applies on restart.
	background, colour := application.BackgroundTypeSolid, application.NewRGBA(25, 25, 30, 255)
	if store.Get().Translucent {
		background, colour = application.BackgroundTypeTranslucent, application.NewRGBA(25, 25, 30, 204)
	}

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Mortar",
		Width:            1280,
		Height:           720,
		MinWidth:         768,
		MinHeight:        432,
		Frameless:        true,
		BackgroundType:   background,
		BackgroundColour: colour,
		Windows: application.WindowsWindow{
			BackdropType: application.Acrylic,
		},
		EnableFileDrop: true,
		URL:            "/",
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		app.Event.Emit(picker.DroppedEvent, e.Context().DroppedFiles())
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
