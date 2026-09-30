package main

import (
	"embed"
	"log"
	"os"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backdrop"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/loadersvc"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/modmenu"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/nexussvc"
	"github.com/Rethunk-AI/mortar/internal/picker"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
	"github.com/Rethunk-AI/mortar/internal/settings"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// version is the app version sent to Nexus; keep it equal to build/config.yml.
const version = "0.0.1"

//go:embed all:frontend/dist
var assets embed.FS

// registerEvents declares the custom events for the binding generator and the runtime's payload checks.
func registerEvents() {
	application.RegisterEvent[launchsvc.Status](launchsvc.StateEvent)
	application.RegisterEvent[launchsvc.Lines](launchsvc.LineEvent)
	application.RegisterEvent[loadersvc.Progress](loadersvc.ProgressEvent)
	application.RegisterEvent[[]string](picker.DroppedEvent)
	application.RegisterEvent[settings.Settings](settings.ChangedEvent)
	application.RegisterEvent[nexussvc.Account](nexussvc.ChangedEvent)
	application.RegisterEvent[modmenu.Target](modmenu.DetailsEvent)
	application.RegisterEvent[modmenu.Target](modmenu.RemoveEvent)
	application.RegisterEvent[profile.Profile](modmenu.ChangedEvent)
	application.RegisterEvent[string](modmenu.FailedEvent)
}

func main() {
	registerEvents()
	store, err := settings.Open()
	if err != nil {
		log.Fatal(err)
	}
	svc := settings.NewService(store)
	svc.ValidateGameFolder = game.ValidateFolder
	svc.ValidateImage = backdrop.Check

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
	loaders := loadersvc.NewService(home, store, items, profiles)
	profiles.Bundled = loadersvc.BundledKey(loaders)
	loadersvc.SyncBundled(loaders, "stardew")
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

	modMeta := &meta.Client{}
	savesSvc, err := savessvc.NewService(profiles, store, modMeta)
	if err != nil {
		log.Fatal(err)
	}

	nexusSvc := nexussvc.NewService(store, nexus.New(version))

	var window *application.WebviewWindow
	pick := &picker.Service{}
	profileSvc := profile.NewService(profiles)
	problemsSvc := problems.NewService(home, store, profiles, modMeta)

	menuSvc := &modmenu.Service{}

	app := application.New(application.Options{
		Name:        "Mortar",
		Description: "Multi-game desktop mod manager",
		Services: []application.Service{
			application.NewService(svc), application.NewService(gamesSvc),
			application.NewService(profileSvc), application.NewService(loaders), application.NewService(launches), application.NewService(pick),
			application.NewService(savesSvc), application.NewService(nexusSvc),
			application.NewService(problemsSvc), application.NewService(menuSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			Middleware: application.ChainMiddleware(
				game.ArtMiddleware(home),
				backdrop.Middleware(store.Get, backdrop.SystemDefault, backdrop.DesktopWallpaper),
			),
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
	nexusSvc.App = app
	menuSvc.Register(app, modmenu.Backend{
		SetEnabled: func(t modmenu.Target, enabled bool) (profile.Profile, error) {
			return profileSvc.SetModEnabled(t.Game, t.Profile, t.Key, t.UniqueID, enabled)
		},
		ShowFiles: func(t modmenu.Target) error {
			return profileSvc.ShowFiles(t.Game, t.Profile, t.Key, t.UniqueID)
		},
		PageURL: func(t modmenu.Target) (string, error) {
			r, err := problemsSvc.Relations(t.Game, t.Profile, t.Key, t.UniqueID)
			return r.PageURL, err
		},
		OpenURL: app.Browser.OpenURL,
		Emit:    app.Event.Emit,
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Mortar",
		Width:            1280,
		Height:           720,
		MinWidth:         768,
		MinHeight:        432,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGBA(25, 25, 30, 255),
		EnableFileDrop:   true,
		URL:              "/",
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		app.Event.Emit(picker.DroppedEvent, e.Context().DroppedFiles())
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
