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
	"github.com/Rethunk-AI/mortar/internal/sharesvc"
	modstore "github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/support"
	"github.com/Rethunk-AI/mortar/internal/updatesvc"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// version is the app version sent to Nexus; keep it equal to build/config.yml.
const version = "0.0.1"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// updateKey verifies release signatures; its private half never enters the repository (docs/architecture.md § Release).
//
//go:embed build/updater/public.key
var updateKey []byte

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
	application.RegisterEvent[sharesvc.Arrival](sharesvc.ArrivedEvent)
}

func main() {
	registerEvents()
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// application.New exits a second instance after forwarding its arguments, so it runs before anything that
	// writes to disk: a second instance must not clean, collect or rewrite the running instance's data.
	var (
		store    *settings.Store
		nxmSvc   *nxmsvc.Service
		shareSvc *sharesvc.Service
		window   *application.WebviewWindow
		profiles *profile.Store
	)
	ready := make(chan struct{})
	app := application.New(application.Options{
		Icon: appIcon,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			Middleware: application.ChainMiddleware(
				game.ArtMiddleware(home),
				profile.CoverMiddleware(func() *profile.Store { return profiles }),
				backdrop.Middleware(func() settings.Settings {
					if store == nil {
						return settings.Defaults()
					}
					return store.Get()
				}, backdrop.SystemDefault, backdrop.DesktopWallpaper),
			),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "tech.rethunk.mortar",
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				<-ready
				// A minimised window stays down: the window sends a desktop notification whose click brings it up.
				nxmLink := nxmSvc.Receive(d.Args)
				if shareSvc.Receive(sharesvc.InDir(d.Args, d.WorkingDir)) || !nxmLink || !window.IsMinimised() {
					window.Restore()
					window.Focus()
				}
			},
		},
	})

	store, err = settings.Open()
	if err != nil {
		log.Fatal(err)
	}
	svc := settings.NewService(store)
	svc.ValidateGameFolder = game.ValidateFolder
	svc.ValidateImage = backdrop.Check

	gamesSvc := game.NewService(home, store)

	items, err := modstore.Open()
	if err != nil {
		log.Fatal(err)
	}
	if err := items.Cleanup(); err != nil {
		log.Printf("store cleanup: %v", err)
	}

	profiles, err = profile.Open(items)
	if err != nil {
		log.Fatal(err)
	}
	launches := launchsvc.NewService(home, store, profiles)
	profiles.Running = launches.Running
	profiles.BackupsKept = func() int { return store.Get().BackupsKept }
	loaders := loadersvc.NewService(home, store, items, profiles)
	loadersvc.Attach(loaders, "stardew")
	launches.EnsureLoader = func(ctx context.Context, id string, fromStart bool) error {
		_, err := loaders.Ensure(ctx, id, fromStart)
		return err
	}
	now := time.Now()
	if err := profiles.PurgeTrash(now); err != nil {
		log.Printf("purge trash: %v", err)
	}

	modMeta := &meta.Client{}
	savesSvc, err := savessvc.NewService(profiles, store, modMeta)
	if err != nil {
		log.Fatal(err)
	}

	nexusClient := nexus.New(version)
	nexusSvc := nexussvc.NewService(store, nexusClient, modMeta)

	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	nxmHandler, err := nxm.New(exe)
	if err != nil {
		log.Fatal(err)
	}
	if err := nxmHandler.Refresh(); err != nil {
		log.Printf("desktop entry: %v", err)
	}
	nxmSvc = nxmsvc.NewService(store, nxmHandler)

	dataDir, err := datadir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	updates := &updatesvc.Service{}
	emit := func(name string, data any) { app.Event.Emit(name, data) }
	queueSvc, err := queue.New(queue.Deps{
		Client:  func() (*nexus.Client, error) { return nexussvc.Authed(store, nexusClient) },
		Premium: func() bool { return store.Get().NexusPremium },
		Install: func(game, profileID, path string, src profile.Source) (profile.InstallResult, error) {
			res, err := profiles.InstallNexus(game, profileID, path, src)
			if err == nil {
				// Warm the detail dialog's cache while the account is known to be signed in and online.
				go func() { _, _ = nexusSvc.Details(context.Background(), src.ModID) }()
			}
			return res, err
		},
		Stage:         profiles.StageGitHub,
		InstallStaged: profiles.InstallStaged,
		Verify: func(ctx context.Context, uniqueID, owner, repo string) (bool, error) {
			return github.Verify(ctx, modMeta, uniqueID, owner, repo)
		},
		GitHub:  &github.Client{},
		OpenURL: func(url string) error { return app.Browser.OpenURL(url) },
		Running: launches.Running,
		Emit:    emit,
		Dir:     dataDir,
		Changed: func(st queue.State) { shareSvc.QueueChanged(st) },
	})
	if err != nil {
		log.Fatal(err)
	}
	nxmSvc.Route = queueSvc.Route
	// An unreadable profile.json stops collection: its keys are unknown, and their items must not be deleted.
	if keys, err := profiles.StoreKeys(); err != nil {
		log.Printf("store collect skipped: %v", err)
	} else {
		for g, staged := range queueSvc.StagedKeys() {
			keys[g] = append(keys[g], staged...)
		}
		if err := items.Collect(keys, now); err != nil {
			log.Printf("store collect: %v", err)
		}
	}
	notifier := notifications.New()

	pick := &picker.Service{}
	profileSvc := profile.NewService(profiles)
	problemsSvc := problems.NewService(home, store, profiles, modMeta)
	shareSvc = sharesvc.NewService(sharesvc.Deps{
		Profiles: profiles,
		Meta:     modMeta,
		Files: func(ctx context.Context, modID int) ([]nexus.File, error) {
			c, err := nexussvc.Authed(store, nexusClient)
			if err != nil {
				return nil, err
			}
			return c.Files(ctx, modID)
		},
		SignedIn: func() bool { return store.Get().NexusUserID != 0 },
		Premium:  func() bool { return store.Get().NexusPremium },
		Env:      problemsSvc.Environment,
		Queue:    queueSvc,
		Dir:      dataDir,
		Emit:     emit,
	})
	// Queue changes reach shareSvc, so links are routed only once both exist.
	nxmSvc.Receive(os.Args[1:])
	shareSvc.Receive(os.Args[1:])
	shareSvc.QueueChanged(queueSvc.State())

	for _, s := range []application.Service{
		application.NewService(svc), application.NewService(gamesSvc),
		application.NewService(profileSvc), application.NewService(loaders), application.NewService(launches), application.NewService(pick),
		application.NewService(savesSvc), application.NewService(nexusSvc), application.NewService(nxmSvc), application.NewService(notifier),
		application.NewService(problemsSvc), application.NewService(queueSvc), application.NewService(shareSvc),
		application.NewService(support.NewService(version, problemsSvc.Environment, home, profiles.ModsDir)), application.NewService(updates),
	} {
		app.RegisterService(s)
	}

	if err := updatesvc.Configure(updates, app.Updater, version, updateKey); err != nil {
		log.Fatal(err)
	}
	queue.Run(context.Background(), queueSvc, nxmSvc.Assigned)
	svc.App = app
	loaders.App = app
	launches.App = app
	loadersvc.EnsureExisting(loaders, "stardew")
	pick.App = app
	nexusSvc.App = app
	nxmSvc.App = app
	shareSvc.App = app
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

	close(ready)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
