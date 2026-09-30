package main

import (
	"context"
	"embed"
	"log"
	"os"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backdrop"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/loadersvc"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/nexussvc"
	"github.com/Rethunk-AI/mortar/internal/nxm"
	"github.com/Rethunk-AI/mortar/internal/nxmsvc"
	"github.com/Rethunk-AI/mortar/internal/picker"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
	"github.com/Rethunk-AI/mortar/internal/settings"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
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
	application.RegisterEvent[queue.State](queue.ChangedEvent)
	application.RegisterEvent[nxmsvc.Arrival](nxmsvc.ArrivedEvent)
	application.RegisterEvent[nxmsvc.Rejection](nxmsvc.RejectedEvent)
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
	loadersvc.Attach(loaders, "stardew")
	launches := launchsvc.NewService(home, store, profiles)
	profiles.Running = launches.Running
	launches.EnsureLoader = func(ctx context.Context, id string) error {
		_, err := loaders.Ensure(ctx, id)
		return err
	}
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

	nexusClient := nexus.New(version)
	nexusSvc := nexussvc.NewService(store, nexusClient)

	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	nxmHandler, err := nxm.New(exe)
	if err != nil {
		log.Fatal(err)
	}
	nxmSvc := nxmsvc.NewService(store, nxmHandler)

	dataDir, err := datadir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	var app *application.App
	queueSvc, err := queue.New(queue.Deps{
		Client:        func() (*nexus.Client, error) { return nexussvc.Authed(store, nexusClient) },
		Premium:       func() bool { return store.Get().NexusPremium },
		Install:       profiles.InstallNexus,
		Stage:         profiles.StageGitHub,
		InstallStaged: profiles.InstallStaged,
		Verify: func(ctx context.Context, uniqueID, owner, repo string) (bool, error) {
			return github.Verify(ctx, modMeta, uniqueID, owner, repo)
		},
		GitHub:  &github.Client{},
		OpenURL: func(url string) error { return app.Browser.OpenURL(url) },
		Emit: func(name string, data any) {
			if app != nil {
				app.Event.Emit(name, data)
			}
		},
		Dir: dataDir,
	})
	if err != nil {
		log.Fatal(err)
	}
	nxmSvc.Route = queueSvc.Route
	nxmSvc.Receive(os.Args[1:])
	notifier := notifications.New()

	var window *application.WebviewWindow
	pick := &picker.Service{}
	profileSvc := profile.NewService(profiles)
	problemsSvc := problems.NewService(home, store, profiles, modMeta)

	app = application.New(application.Options{
		Name:        "Mortar",
		Description: "Multi-game desktop mod manager",
		Services: []application.Service{
			application.NewService(svc), application.NewService(gamesSvc),
			application.NewService(profileSvc), application.NewService(loaders), application.NewService(launches), application.NewService(pick),
			application.NewService(savesSvc), application.NewService(nexusSvc), application.NewService(nxmSvc), application.NewService(notifier),
			application.NewService(problemsSvc), application.NewService(queueSvc),
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
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				// A minimised window stays down: the window sends a desktop notification whose click brings it up.
				if !nxmSvc.Receive(d.Args) || !window.IsMinimised() {
					window.Restore()
					window.Focus()
				}
			},
		},
	})

	queue.Run(context.Background(), queueSvc, nxmSvc.Assigned)
	svc.App = app
	loaders.App = app
	launches.App = app
	loadersvc.EnsureExisting(loaders, "stardew")
	pick.App = app
	nexusSvc.App = app
	nxmSvc.App = app
	notifier.OnNotificationResponse(func(notifications.NotificationResult) {
		window.Restore()
		window.Focus()
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
