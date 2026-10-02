package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-AI/mortar/internal/backdrop"
	"github.com/Rethunk-AI/mortar/internal/bisect"
	"github.com/Rethunk-AI/mortar/internal/components"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/datasvc"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/github"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/loadersvc"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/modpic"
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
	"github.com/Rethunk-AI/mortar/internal/tools"
	"github.com/Rethunk-AI/mortar/internal/updatesvc"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// version is the app version sent to Nexus; keep it equal to build/config.yml.
const version = "0.0.1"

// packaged is set by nfpm, Flatpak and AUR builds (`-X main.packaged=deb`) so the self-updater stays off.
var packaged string

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
	application.RegisterEvent[launchsvc.Crash](launchsvc.CrashEvent)
	application.RegisterEvent[loadersvc.Progress](loadersvc.ProgressEvent)
	application.RegisterEvent[[]string](picker.DroppedEvent)
	application.RegisterEvent[settings.Settings](settings.ChangedEvent)
	application.RegisterEvent[nexussvc.Account](nexussvc.ChangedEvent)
	application.RegisterEvent[queue.State](queue.ChangedEvent)
	application.RegisterEvent[nxmsvc.Arrival](nxmsvc.ArrivedEvent)
	application.RegisterEvent[nxmsvc.Rejection](nxmsvc.RejectedEvent)
	application.RegisterEvent[sharesvc.Arrival](sharesvc.ArrivedEvent)
	application.RegisterEvent[launchsvc.NoticeClick](launchsvc.NoticeClickEvent)
	application.RegisterEvent[updatesvc.Release](updatesvc.StagedEvent)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "--release-links" {
		return releaseLinks()
	}
	registerEvents()
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// application.New exits a second instance after forwarding its arguments, so it runs before anything that
	// writes to disk: a second instance must not clean, collect or rewrite the running instance's data.
	var (
		store    *settings.Store
		nxmSvc   *nxmsvc.Service
		shareSvc *sharesvc.Service
		window   *application.WebviewWindow
		// showWindow brings Mortar's window up, building a new one when closing to the tray removed it.
		showWindow func()
		profiles   *profile.Store
		pictures   *modpic.Cache
	)
	ready := make(chan struct{})
	closeReady := sync.OnceFunc(func() { close(ready) })
	defer closeReady()
	// A burst of nxm clicks starts one second instance per link; each launch is queued at once and handled here in
	// order, so a slow handoff (waiting for startup, asking the window whether it is minimised) never holds up the next.
	handoffs := make(chan application.SecondInstanceData, 256)
	go func() {
		<-ready
		for d := range handoffs {
			if window == nil || nxmSvc == nil || shareSvc == nil {
				log.Printf("second instance: dropped, Mortar did not finish starting")
				continue
			}
			// A minimised window stays down: the window sends a desktop notification whose click brings it up.
			nxmLink := nxmSvc.Receive(d.Args)
			if shareSvc.Receive(sharesvc.InDir(d.Args, d.WorkingDir)) || !nxmLink || !window.IsMinimised() {
				showWindow()
			}
		}
	}()
	dataDir, err := datadir.Dir()
	if err != nil {
		return err
	}
	updates := &updatesvc.Service{}
	app := application.New(application.Options{
		Name: "Mortar",
		Icon: appIcon,
		// ApplicationID is the GtkApplication / Wayland app_id and the Linux desktop file id. It must not equal
		// UniqueID: both become D-Bus names, and GApplication also owns ApplicationID on the session bus.
		// Mortar decides itself whether closing the last window quits or leaves it in the tray.
		Linux: application.LinuxOptions{ApplicationID: "tech.rethunk.Mortar", DisableQuitOnLastWindowClosed: true},
		Windows: application.WindowsOptions{
			WebviewUserDataPath:           filepath.Join(dataDir, "webview"),
			DisableQuitOnLastWindowClosed: true,
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			Middleware: application.ChainMiddleware(
				game.ArtMiddleware(home),
				profile.CoverMiddleware(func() *profile.Store { return profiles }),
				modpic.Middleware(func() *modpic.Cache { return pictures }),
				backdrop.Middleware(func() settings.Settings {
					if store == nil {
						return settings.Defaults()
					}
					return store.Get()
				}, backdrop.SystemDefault, backdrop.DesktopWallpaper),
			),
		},
		OnShutdown: func() {
			_ = updates.ApplyOnQuit(context.Background())
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "tech.rethunk.mortar",
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				log.Printf("second instance: %d args queued", len(d.Args)-1)
				handoffs <- d
			},
		},
	})

	store, err = settings.Open()
	if err != nil {
		return err
	}
	svc := settings.NewService(store)
	svc.ValidateGameFolder = game.ValidateFolder
	svc.ValidateImage = backdrop.Check

	gamesSvc := game.NewService(home, store)

	items, err := modstore.Open()
	if err != nil {
		return err
	}
	if err := items.Cleanup(); err != nil {
		log.Printf("store cleanup: %v", err)
	}

	profiles, err = profile.Open(items)
	if err != nil {
		return err
	}
	launches := launchsvc.NewService(home, store, profiles)
	profiles.Running = launches.Running
	bisectSvc := bisect.NewService(profiles, launches)
	profiles.BackupsKept = func() int { return store.Get().BackupsKept }
	modMeta := &meta.Client{CacheDir: filepath.Join(dataDir, "cache")}
	componentClient := components.NewClient(&http.Client{Timeout: 30 * time.Second})
	if _, err := componentClient.Load(context.Background(), modMeta, updateKey); err != nil {
		log.Printf("components manifest unavailable; using bundled copy: %v", err)
	}
	game.ConfigureComponents(componentClient)
	loaders := loadersvc.NewService(home, store, items, profiles, componentClient)
	loadersvc.Attach(loaders, "stardew")
	launches.EnsureLoader = func(ctx context.Context, id string, fromStart bool) error {
		_, err := loaders.Ensure(ctx, id, fromStart)
		return err
	}
	now := time.Now()
	if err := profiles.PurgeTrash(now); err != nil {
		log.Printf("purge trash: %v", err)
	}

	savesSvc, err := savessvc.NewService(profiles, store, modMeta)
	if err != nil {
		return err
	}
	savesSvc.Launches = launches

	nexusClient := nexus.New(version)
	nexusSvc := nexussvc.NewService(store, nexusClient, modMeta)

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	nxmHandler, err := nxm.New(exe)
	if err != nil {
		return err
	}
	if err := nxmHandler.Refresh(); err != nil {
		log.Printf("desktop entry: %v", err)
	}
	nxmSvc = nxmsvc.NewService(store, nxmHandler)

	pictures = modpic.New(dataDir, &http.Client{Timeout: 30 * time.Second})
	if err := nexusSvc.UseDataDir(dataDir); err != nil {
		return err
	}
	emit := func(name string, data any) { app.Event.Emit(name, data) }
	queueSvc, err := queue.New(queue.Deps{
		Client:  func() (*nexus.Client, error) { return nexussvc.Authed(store, nexusClient) },
		Premium: func() bool { return store.Get().NexusPremium },
		Install: func(game, profileID, path string, src profile.Source) (profile.InstallResult, error) {
			res, err := profiles.InstallNexus(game, profileID, path, src)
			if err == nil {
				// Warm the detail dialog's cache while the account is known to be signed in and online.
				go func() { _, _ = nexusSvc.Details(context.Background(), src.ModID) }()
				go pictures.Ensure(context.Background(), src.Picture)
			}
			return res, err
		},
		Stage:         profiles.StageGitHub,
		InstallStaged: profiles.InstallStaged,
		InstallRemap:  profiles.InstallRemap,
		SamePage: func(game, profileID string, modID, fileID int, category string) (profile.MergeAsk, bool) {
			all, err := profiles.List(game)
			if err != nil {
				return profile.MergeAsk{}, false
			}
			for _, p := range all {
				if p.ID == profileID {
					return profile.SamePageAsk(p, modID, fileID, category)
				}
			}
			return profile.MergeAsk{}, false
		},
		InstallExtra: func(game, profileID, entryKey, path string, src profile.Source) (profile.InstallResult, error) {
			res, err := profiles.InstallNexusExtra(game, profileID, entryKey, path, src)
			if err == nil {
				go func() { _, _ = nexusSvc.Details(context.Background(), src.ModID) }()
				go pictures.Ensure(context.Background(), src.Picture)
			}
			return res, err
		},
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
		return err
	}
	launches.Unlocked = func() { queue.NotifyUnlocked(queueSvc) }
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
	profileSvc := profile.NewService(profiles, home, store)
	profileSvc.Version = version
	problemsSvc := problems.NewService(home, store, profiles, modMeta)
	problemsSvc.Runs = launches
	supportSvc := support.NewService(version, problemsSvc.Environment, home, profiles.ModsDir)
	supportSvc.RecentLog = func(gameID, profileID string) string {
		entries, err := launches.Lines(gameID, profileID)
		if err != nil || len(entries) == 0 {
			return ""
		}
		if len(entries) > 2000 {
			entries = entries[len(entries)-2000:]
		}
		var b strings.Builder
		for i, e := range entries {
			if i > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "[%s %s %s] %s", e.Time, e.Level, e.Mod, e.Message)
		}
		return b.String()
	}
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
	shareSvc.Receive(sharesvc.InDir(os.Args[1:], sharesvc.LaunchDir()))
	shareSvc.QueueChanged(queueSvc.State())

	toolsSvc, err := tools.NewService(home, store, profiles)
	if err != nil {
		return err
	}

	dataSvc := datasvc.NewService(items, profiles, queueSvc.StagedKeys)
	dataSvc.Busy = func() bool {
		st, err := launches.Status("stardew")
		return err == nil && (st.State == launchsvc.Launching || st.State == launchsvc.Running)
	}
	dataSvc.Restart = datasvc.RestartSelf

	for _, s := range []application.Service{
		application.NewService(svc), application.NewService(gamesSvc),
		application.NewService(profileSvc), application.NewService(loaders), application.NewService(launches), application.NewService(pick),
		application.NewService(savesSvc), application.NewService(nexusSvc), application.NewService(nxmSvc), application.NewService(notifier),
		application.NewService(problemsSvc), application.NewService(queueSvc), application.NewService(shareSvc),
		application.NewService(supportSvc), application.NewService(updates), application.NewService(bisectSvc),
		application.NewService(dataSvc), application.NewService(toolsSvc),
	} {
		app.RegisterService(s)
	}

	if err := updatesvc.Configure(updates, app.Updater, version, updateKey, packaged, func() bool {
		return store.Get().IncludeBetaReleases
	}, dataDir); err != nil {
		return err
	}
	updateCtx, stopUpdates := context.WithCancel(context.Background())
	defer stopUpdates()
	updates.StartBackground(updateCtx, emit)
	queueCtx, stopQueue := context.WithCancel(context.Background())
	launchsvc.SetLife(launches, queueCtx)
	waitQueue := queue.Run(queueCtx, queueSvc, nxmSvc.Assigned)
	defer func() {
		stopQueue()
		waitQueue()
	}()
	svc.App = app
	loaders.App = app
	profileSvc.App = app
	launches.App = app
	loadersvc.EnsureExisting(loaders, "stardew")
	pick.App = app
	nexusSvc.App = app
	nxmSvc.App = app
	shareSvc.App = app
	supportSvc.App = app
	notifier.OnNotificationResponse(func(result notifications.NotificationResult) {
		showWindow()
		gameID, profileID := launchsvc.NoticeProfileFromResponse(result.Response.ID, result.Response.UserInfo)
		if profileID != "" && gameID != "" {
			app.Event.Emit(launchsvc.NoticeClickEvent, launchsvc.NoticeClick{Game: gameID, Profile: profileID})
		}
	})

	var windowMu sync.Mutex
	windowGone := false
	newWindow := func() *application.WebviewWindow {
		w := app.Window.NewWithOptions(application.WebviewWindowOptions{
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
		w.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
			app.Event.Emit(picker.DroppedEvent, e.Context().DroppedFiles())
		})
		// Wayland gives an app no say over where a re-shown window goes, so closing to the tray
		// destroys the window and showing builds a fresh one that the compositor places as new.
		w.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
			if !store.Get().KeepInTray {
				app.Quit()
				return
			}
			windowMu.Lock()
			windowGone = true
			windowMu.Unlock()
		})
		return w
	}
	window = newWindow()
	showWindow = func() {
		windowMu.Lock()
		defer windowMu.Unlock()
		if windowGone {
			window = newWindow()
			windowGone = false
			return
		}
		window.Restore()
		window.Show().Focus()
	}

	var tray *application.SystemTray
	var trayMenu *application.Menu
	refreshTrayMenu := func() {
		if trayMenu == nil {
			return
		}
		trayMenu.Clear()
		trayMenu.Add("Show Mortar").OnClick(func(*application.Context) {
			showWindow()
		})
		st, _ := launches.Status("stardew")
		running := st.State == launchsvc.Launching || st.State == launchsvc.Running
		gameName := "Stardew Valley"
		if g := game.Find("stardew"); g != nil {
			gameName = g.Name()
		}
		if running {
			trayMenu.Add(gameName + " is running").SetEnabled(false)
		}
		recent, _ := launches.RecentLaunches("stardew", 3)
		for _, row := range recent {
			item := trayMenu.Add("Play " + row.Name)
			if running {
				item.SetEnabled(false)
			} else {
				profileID := row.ProfileID
				item.OnClick(func(*application.Context) {
					_ = launches.Start("stardew", profileID, false)
				})
			}
		}
		trayMenu.AddSeparator()
		trayMenu.Add("Quit").OnClick(func(*application.Context) {
			app.Quit()
		})
		trayMenu.Update()
	}
	syncTray := func() {
		if !store.Get().KeepInTray {
			if tray != nil {
				tray.Destroy()
				tray = nil
				trayMenu = nil
			}
			return
		}
		if tray == nil {
			tray = app.SystemTray.New().SetIcon(appIcon)
			trayMenu = app.NewMenu()
			tray.SetMenu(trayMenu)
			tray.OnClick(func() {
				showWindow()
			})
			launches.NotifyRunEnd = func(n launchsvc.RunEndNotice) {
				id := fmt.Sprintf("run-end-%s-%d", n.Profile, time.Now().UnixNano())
				opts := notifications.NotificationOptions{
					ID:    id,
					Title: n.Title,
					Body:  n.Body,
					Data:  map[string]any{"game": n.Game, "profile": n.Profile},
				}
				if icon := nxmSvc.NotificationIcon(); icon != "" {
					opts.Attachments = []notifications.NotificationAttachment{
						{ID: "icon", Path: icon, Type: "appLogoOverride"},
					}
				}
				if err := notifier.SendNotification(opts); err != nil {
					log.Printf("run-end notification: %v", err)
				}
			}
			app.Event.On(launchsvc.StateEvent, func(*application.CustomEvent) {
				refreshTrayMenu()
			})
		}
		refreshTrayMenu()
	}
	syncTray()
	app.Event.On(settings.ChangedEvent, func(*application.CustomEvent) {
		syncTray()
	})

	closeReady()
	err = app.Run()
	stopQueue()
	return err
}

func releaseLinks() error {
	store, err := settings.Open()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	h, err := nxm.New(exe)
	if err != nil {
		return err
	}
	return nxmsvc.ReleaseLinks(store, h)
}
