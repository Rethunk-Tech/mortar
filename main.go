package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/archivesvc"
	"github.com/Rethunk-Tech/mortar/internal/backdrop"
	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/browse"
	"github.com/Rethunk-Tech/mortar/internal/bundles"
	"github.com/Rethunk-Tech/mortar/internal/cli"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/desktopnotify"
	"github.com/Rethunk-Tech/mortar/internal/folderwatch"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/loadersvc"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/modpic"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/picker"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
	modstore "github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/storecheck"
	"github.com/Rethunk-Tech/mortar/internal/support"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/tidy"
	"github.com/Rethunk-Tech/mortar/internal/tools"
	"github.com/Rethunk-Tech/mortar/internal/updatesvc"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed build/config.yml
var buildConfig []byte

// version is build/config.yml's info.version, the one place the app version is set.
var version = mustVersion()

func mustVersion() string {
	v, err := appversion.FromConfig(buildConfig)
	if err != nil {
		panic(err)
	}
	return v
}

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
	application.RegisterEvent[string](control.ChangedEvent)
	application.RegisterEvent[control.InstallAsk](control.InstallAskEvent)
	application.RegisterEvent[shortcut.Request](shortcut.RequestedEvent)
	application.RegisterEvent[nexussvc.Account](nexussvc.ChangedEvent)
	application.RegisterEvent[nexussvc.SSOState](nexussvc.SSOEvent)
	application.RegisterEvent[queue.State](queue.ChangedEvent)
	application.RegisterEvent[queue.Progress](queue.ProgressEvent)
	application.RegisterEvent[nxmsvc.Arrival](nxmsvc.ArrivedEvent)
	application.RegisterEvent[nxmsvc.Rejection](nxmsvc.RejectedEvent)
	application.RegisterEvent[string](folderwatch.ModsFolderEvent)
	application.RegisterEvent[string](folderwatch.ExtraFolderEvent)
	application.RegisterEvent[string](folderwatch.DownloadsEvent)
	application.RegisterEvent[sharesvc.Arrival](sharesvc.ArrivedEvent)
	application.RegisterEvent[lan.Arrival](lan.ArrivedEvent)
	application.RegisterEvent[lan.TransferProgress](lan.TransferProgressEvent)
	application.RegisterEvent[launchsvc.NoticeClick](launchsvc.NoticeClickEvent)
	application.RegisterEvent[launchsvc.BackupWarning](launchsvc.BackupWarningEvent)
	application.RegisterEvent[launchsvc.SettingsRestoreWarning](launchsvc.SettingsRestoreWarningEvent)
	application.RegisterEvent[updatesvc.Release](updatesvc.StagedEvent)
	application.RegisterEvent[updatesvc.ModUpdateDigestNotice](updatesvc.ModUpdateDigestEvent)
	application.RegisterEvent[string](quitRequestedEvent)
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
	if cli.Is(os.Args[1:]) {
		os.Exit(cli.Run(version, os.Args[1:], os.Stdout, os.Stderr))
	}
	if nativehost.Invoked(os.Args[1:]) {
		return serveNativeHost()
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
		lanSvc   *lan.Service
		window   *application.WebviewWindow
		// showWindow brings Mortar's window up, building a new one when closing to the tray removed it.
		showWindow func()
		// windowClosed reports whether closing to the tray removed the window.
		windowClosed func() bool
		profiles     *profile.Store
		pictures     *modpic.Cache
	)
	ready := make(chan struct{})
	plays := &shortcut.Service{}
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
			// An nxm link leaves an open window where it is, minimised or behind the browser, so a burst of clicks
			// keeps the browser in front; the window sends a desktop notification when a link needs a profile chosen.
			// Only a window closed to the tray is brought back, since nothing else could show the link.
			nxmLink := nxmSvc.Receive(d.Args)
			if shareSvc.Receive(sharesvc.InDir(d.Args, d.WorkingDir)) || plays.Receive(d.Args) || !nxmLink || windowClosed() {
				showWindow()
			}
		}
	}()
	dataDir, err := dataDirOrRecover()
	if err != nil {
		return err
	}
	updates := &updatesvc.Service{}
	app := application.New(application.Options{
		Name:         "Mortar",
		Icon:         appIcon,
		ErrorHandler: logAppError,
		PanicHandler: panicHandler(dataDir),
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
			if lanSvc != nil {
				lanSvc.Shutdown()
			}
			slog.Info("shutdown", "clean", true)
			_ = updates.ApplyOnQuit(context.Background())
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: singleInstanceID(dataDir),
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				log.Printf("second instance: %d args queued", len(d.Args)-1)
				handoffs <- d
			},
		},
	})
	// A windowsgui build has no stderr, so logs and fatal panics would end without a trace; both also go to
	// files in the data folder, the previous run's log kept beside the current one.
	if crash, err := fsx.OpenFile(filepath.Join(dataDir, "crash.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600); err == nil {
		_ = debug.SetCrashOutput(crash, debug.CrashOptions{})
		_ = crash.Close()
	}
	logPath := filepath.Join(dataDir, "mortar.log")
	_ = fsx.Rename(logPath, filepath.Join(dataDir, "mortar.prev.log"))
	if logFile, err := fsx.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600); err == nil {
		// The file comes first: MultiWriter stops at the first failing writer, and a GUI launch can have a dead
		// stderr, which would otherwise leave mortar.log without the clean-shutdown line crash detection reads.
		out := io.MultiWriter(&cappedWriter{w: logFile, left: maxLogBytes}, os.Stderr)
		log.SetOutput(out)
		slog.SetDefault(slog.New(slog.NewTextHandler(out, nil)))
	}
	support.DetectLastRunCrashed(dataDir)
	tidied := &tidy.Collector{}
	if capCrashLog(dataDir) {
		tidied.Add("Emptied the crash log after it was reported", "data folder", 1, "crash.log")
	}

	store, err = settings.Open()
	if err != nil {
		return err
	}
	svc := settings.NewService(store)
	svc.ValidateGameFolder = game.ValidateFolder
	svc.ValidateLauncherRoot = game.ValidateLauncherRoot
	svc.ValidateImage = backdrop.Check

	gamesSvc := game.NewService(home, store)

	items, err := modstore.Open()
	if err != nil {
		return err
	}
	if removed, err := items.Cleanup(); err != nil {
		log.Printf("store cleanup: %v", err)
	} else {
		tidied.Add("Removed leftover temporary folders", "store", 0, removed...)
	}

	profiles, err = profile.Open(items)
	if err != nil {
		return err
	}
	profiles.Tidied = func(what, profileName, folder string) { tidied.Add(what, "profile "+profileName, 1, folder) }
	profiles.ShortcutRenamed = shortcut.Renamed
	profiles.ShortcutRemoved = shortcut.Removed
	plays.Covers = func(gameID, profileID string) ([]string, error) {
		covers, err := profiles.Covers(gameID, profileID)
		if err != nil {
			return nil, err
		}
		if all, err := profiles.List(gameID); err == nil {
			for _, p := range all {
				if p.Error != "" || p.ID != profileID {
					continue
				}
				switch p.Cover {
				case "cover.png", "cover.jpg", "cover.webp":
					covers = append([]string{filepath.Join(dataDir, "profiles", gameID, profileID, p.Cover)}, covers...)
				}
				break
			}
		}
		return covers, nil
	}
	launches := launchsvc.NewService(home, store, profiles)
	gamesSvc.Running = launches.Busy
	profiles.Running = launches.Running
	profiles.GameRunning = launches.Busy
	bisectSvc := bisect.NewService(profiles, launches)
	modMeta := &meta.Client{CacheDir: filepath.Join(dataDir, "cache")}
	componentClient := components.NewClient(&http.Client{Timeout: 30 * time.Second})
	game.ConfigureComponents(componentClient)
	if g, ok := componentClient.Game("stardew"); ok {
		nexus.Configure(g.Nexus.Domain, g.Nexus.ID)
	}
	loaders := loadersvc.NewService(home, store, items, profiles, componentClient)
	loadersvc.Attach(loaders, "stardew")
	launches.EnsureLoader = func(ctx context.Context, id string, fromStart bool) error {
		_, err := loaders.Ensure(ctx, id, fromStart)
		return err
	}
	loaders.OnReady = func(id string) { launches.MaybeSweep(context.Background(), id) }
	now := time.Now()
	if err := profiles.PurgeTrash(now); err != nil {
		log.Printf("purge trash: %v", err)
	}

	savesSvc, err := savessvc.NewService(home, profiles, store, modMeta)
	if err != nil {
		return err
	}
	savesSvc.Launches = launches
	launches.OnSavePlayed = savesSvc.NotePlayed

	nexusClient := nexus.New(version)
	nexusSvc := nexussvc.NewService(store, nexusClient, modMeta)
	nexusSvc.Profiles = profiles
	nexusSvc.GitHub = &github.Client{}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	go updatesvc.RemoveOldExecutables(exe)
	nxmHandler, err := nxm.New(exe)
	if err != nil {
		return err
	}
	if err := nxmHandler.Refresh(); err != nil {
		log.Printf("desktop entry: %v", err)
	}
	if err := shortcut.Repoint(selfexe.Launchable(exe)); err != nil {
		log.Printf("profile shortcuts: %v", err)
	}
	if h, ok := any(nxmHandler).(interface{ WriteNativeHosts() error }); ok && store.Get().NxmHandled {
		if err := h.WriteNativeHosts(); err != nil {
			log.Printf("browser extension host: %v", err)
		}
	}
	nxmSvc = nxmsvc.NewService(store, nxmHandler)

	pictures = modpic.New(dataDir, &http.Client{Timeout: 30 * time.Second})
	if err := nexusSvc.UseDataDir(dataDir); err != nil {
		return err
	}
	emit := func(name string, data any) { app.Event.Emit(name, data) }
	plays.Emit = emit
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
		Stored: func(game, key string) (profile.Source, bool) {
			if _, err := items.Path(game, key); err != nil {
				return profile.Source{}, false
			}
			return profiles.SourceOf(game, key), true
		},
		StoredOverlay: profiles.StoredOverlay,
		Stage:         profiles.StageGitHub,
		InstallStaged: profiles.InstallStaged,
		InstallRemap:  profiles.InstallRemap,
		Newest: func(game, profileID string, modID, current int) int {
			all, err := profiles.List(game)
			if err != nil {
				return 0
			}
			for _, p := range all {
				if p.Error == "" && p.ID == profileID {
					return profile.NewestFromPage(p, modID, current)
				}
			}
			return 0
		},
		SamePage: func(game, profileID string, modID, fileID int, category string) (profile.MergeAsk, bool) {
			all, err := profiles.List(game)
			if err != nil {
				return profile.MergeAsk{}, false
			}
			for _, p := range all {
				if p.Error == "" && p.ID == profileID {
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
		HistoryBatch: func(game, profileID, batchID string) error {
			return profiles.RecordHistoryBatch(game, profileID, batchID)
		},
		Parallel:     func() int { return store.Get().ParallelDownloads },
		KeepArchives: func() bool { return store.Get().KeepDownloadArchives },
		DownloadDir:  func() string { return store.Get().ArchiveDir() },
		RetryFetches: func() int {
			switch store.Get().AutoRetryDownloads {
			case settings.AutoRetry1:
				return 1
			case settings.AutoRetry3:
				return 3
			default:
				return 0
			}
		},
		PauseWhilePlaying: func() bool { return store.Get().PauseDownloadsWhilePlaying },
		GameBusy:          func() bool { return launches.Busy(settings.GameStardew) },
		VerifyNexusMD5:    func() bool { return store.Get().VerifyNexusMD5 },
		Track: func(ctx context.Context, modID int) {
			if !store.Get().AutoTrackNexus || modID <= 0 {
				return
			}
			_ = nexusSvc.Track(ctx, modID)
		},
	})
	if err != nil {
		return err
	}
	launches.Unlocked = func() { queue.NotifyUnlocked(queueSvc) }
	nxmSvc.Route = queueSvc.Route
	notifier := notifications.New()
	desktopnotify.Setup(notifier, nxmSvc.NotificationIcon, func(key string) bool {
		v, err := store.Get().Lookup(key)
		return err == nil && v == "true"
	})

	pick := &picker.Service{}
	profileSvc := profile.NewService(profiles, home, store)
	profileSvc.QueueProfileDeleted = queueSvc.SkipProfile
	profileSvc.QueueProfileRestored = queueSvc.RestoreProfile
	savesSvc.Enqueue = queueSvc.Add
	profileSvc.Version = version
	bundlesSvc := bundles.NewService(profiles, dataDir)
	problemsSvc := problems.NewService(home, store, profiles, modMeta)
	problemsSvc.Runs = launches
	problemsSvc.NexusFiles = func(ctx context.Context, ids []int) (map[int][]nexus.BatchFile, error) {
		c, err := nexussvc.Authed(store, nexusClient)
		if err != nil {
			return nil, err
		}
		return c.FilesOf(ctx, ids)
	}
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
		CollectionArchive: func(ctx context.Context, link string) ([]byte, error) {
			c, err := nexussvc.Authed(store, nexusClient)
			if err != nil {
				return nil, err
			}
			return c.CollectionArchive(ctx, link)
		},
		Env:   problemsSvc.Environment,
		Queue: queueSvc,
		Stored: func(gameID, key string) bool {
			_, err := items.Path(gameID, key)
			return err == nil
		},
		Dir:  dataDir,
		Emit: emit,
	})
	lanSvc = lan.NewService(lan.Deps{
		Shares: shareSvc, Settings: store, Store: items, Version: version,
		NexusKey: func() (string, error) { return secret.Get("nexus") },
		Emit:     emit,
	})
	if err := lanSvc.SetEnabled(store.Get().LanSharing); err != nil {
		log.Printf("LAN sharing: %v", err)
	}
	// Queue changes reach shareSvc, so links are routed only once both exist.
	nxmSvc.Receive(os.Args[1:])
	plays.Receive(os.Args[1:])
	shareSvc.Receive(sharesvc.InDir(os.Args[1:], sharesvc.LaunchDir()))
	shareSvc.QueueChanged(queueSvc.State())

	toolsSvc, err := tools.NewService(home, store, profiles)
	if err != nil {
		return err
	}

	quitSvc := &QuitService{app: app, queue: queueSvc, lan: lanSvc, launch: launches}

	archivesSvc := archivesvc.NewService(archivesvc.Deps{
		Dirs: func() []string { return downloadDirs(store, dataDir) },
		Install: func(game, profileID, path string, src profile.Source) (profile.InstallResult, error) {
			if src.ModID > 0 {
				return profiles.InstallNexus(game, profileID, path, src)
			}
			return profileSvc.InstallArchive(game, profileID, path)
		},
		HashCache: filepath.Join(dataDir, "cache", "archive-hashes.json"),
		Offer:     func(g string) bool { return settings.ToggleOn(store.Get().GamePrefs(g).OfferNewDownloads) },
		Seen:      func(g string) int64 { return store.Get().GamePrefs(g).LastDownloadsSeen },
		SetSeen:   store.RecordDownloadsSeen,
		Keys:      items.Keys,
		Profiles:  profiles.List,
		NexusMods: func() map[int]bool {
			ids := map[int]bool{}
			for _, h := range queueSvc.History() {
				if h.ModID > 0 {
					ids[h.ModID] = true
				}
			}
			return ids
		},
	})
	templatesSvc := templates.NewService(templates.Deps{
		Profiles: profiles, Bundles: bundlesSvc,
		GameSettings: launches.GameSettings, SetGameSettings: launches.SetGameSettings,
	}, dataDir)

	keepSources := []datasvc.KeySource{
		bundlesSvc.ReferencedStoreKeys, templatesSvc.ReferencedStoreKeys,
		func() (map[string][]string, error) { return queueSvc.StagedKeys(), nil },
	}
	dataSvc := datasvc.NewService(items, profiles, keepSources,
		queueSvc, lanSvc,
		datasvc.BusyFunc(func() bool { return launches.Busy(settings.GameStardew) }),
	)
	dataSvc.Restart = datasvc.RestartSelf
	dataSvc.OnClearCache = problemsSvc.ForgetCached
	checkSvc := storecheck.New(storecheck.Deps{
		Items: items, Source: profiles.SourceOf, Add: queueSvc.Add,
		NexusMD5: func(ctx context.Context, modID, fileID int) (string, error) {
			c, err := nexussvc.Authed(store, nexusClient)
			if err != nil {
				return "", err
			}
			files, err := c.Files(ctx, modID)
			for _, f := range files {
				if f.FileID == fileID {
					return f.MD5, err
				}
			}
			return "", err
		},
		ArchiveDir: func() string { return archiveDir(store, dataDir) },
		Busy:       func() bool { return launches.Busy(settings.GameStardew) || queueSvc.Active() },
		Emit:       emit,
	})
	problemsSvc.Damage = items.Damaged

	for _, s := range []application.Service{
		application.NewService(svc), application.NewService(gamesSvc),
		application.NewService(profileSvc), application.NewService(loaders), application.NewService(launches), application.NewService(pick),
		application.NewService(bundlesSvc), application.NewService(templatesSvc), application.NewService(archivesSvc),
		application.NewService(savesSvc), application.NewService(plays), application.NewService(nexusSvc), application.NewService(nxmSvc), application.NewService(notifier),
		application.NewService(problemsSvc), application.NewService(queueSvc), application.NewService(shareSvc), application.NewService(lanSvc),
		application.NewService(supportSvc), application.NewService(updates), application.NewService(bisectSvc),
		application.NewService(dataSvc), application.NewService(toolsSvc),
		application.NewService(checkSvc), application.NewService(&tidy.Service{Report: tidied}),
		application.NewService(quitSvc),
		application.NewService(browse.NewService(version, profileSvc)),
	} {
		app.RegisterService(s)
	}

	if packaged == "" {
		if exe, err := os.Executable(); err == nil {
			packaged = updatesvc.PackagedBy(exe)
		}
	}
	if err := updatesvc.Configure(updates, app.Updater, version, updateKey, packaged, func() bool {
		return store.Get().IncludeBetaReleases
	}, dataDir); err != nil {
		return err
	}
	updates.AutoInstall = func() bool { return store.Get().AutoInstallMortar() }
	updateCtx, stopUpdates := context.WithCancel(context.Background())
	defer stopUpdates()
	updates.StartBackground(updateCtx, emit)
	savesSvc.Emit = emit
	go savesSvc.RunScheduledBackups(updateCtx)
	queueCtx, stopQueue := context.WithCancel(context.Background())
	launchsvc.SetLife(launches, queueCtx)
	go storecheck.Run(queueCtx, checkSvc)
	watchLibraryFolders(queueCtx, home, dataDir, store, emit)
	waitQueue := queue.Run(queueCtx, queueSvc, nxmSvc.Assigned)
	defer func() {
		stopQueue()
		waitQueue()
	}()
	ctl := &control.Services{
		Version: version, Settings: store, SettingsSvc: svc, Games: gamesSvc, Store: profiles, Profiles: profileSvc,
		Problems: problemsSvc, Launches: launches, Saves: savesSvc, Queue: queueSvc, Tools: toolsSvc, Bundles: bundlesSvc,
		Nexus: nexusSvc, Shares: shareSvc, Data: dataSvc, Plays: plays, Loaders: loaders, Templates: templatesSvc, Archives: archivesSvc, Emit: emit,
		Quit: func() {
			// Busy downloads or a running game get the window's own confirmation, as the tray Quit does.
			if quitSvc.BusySummary() == "" {
				quitSvc.ConfirmQuit()
			} else {
				quitSvc.RequestQuit()
			}
		},
	}
	go func() {
		if err := control.Serve(queueCtx, dataDir, version, ctl.Handle); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("control: %v", err)
		}
	}()
	svc.App = app
	loaders.App = app
	profileSvc.App = app
	launches.App = app
	if err := launches.RecoverGameSettings(); err != nil {
		log.Printf("game settings restore: %v", err)
		app.Event.Emit(launchsvc.SettingsRestoreWarningEvent, launchsvc.SettingsRestoreWarning{Game: "stardew", Error: err.Error()})
	}
	loadersvc.EnsureExisting(loaders, "stardew")
	launchsvc.SweepOnStart(launches, "stardew")
	pick.App = app
	nexusSvc.App = app
	nxmSvc.App = app
	shareSvc.App = app
	supportSvc.App = app
	notifier.OnNotificationResponse(func(result notifications.NotificationResult) {
		showWindow()
		gameID, profileID, tab := launchsvc.NoticeProfileFromResponse(result.Response.ID, result.Response.UserInfo)
		if profileID != "" && gameID != "" {
			app.Event.Emit(launchsvc.NoticeClickEvent, launchsvc.NoticeClick{Game: gameID, Profile: profileID, Tab: tab})
		}
	})

	var windowMu sync.Mutex
	windowGone := false
	newWindow := func() *application.WebviewWindow {
		opts := application.WebviewWindowOptions{
			Title:            "Mortar",
			Width:            defaultWindowWidth,
			Height:           defaultWindowHeight,
			MinWidth:         minWindowWidth,
			MinHeight:        minWindowHeight,
			Frameless:        true,
			BackgroundType:   application.BackgroundTypeSolid,
			BackgroundColour: application.NewRGBA(25, 25, 30, 255),
			EnableFileDrop:   true,
			URL:              "/",
			Hidden:           store.Get().StartMinimised,
		}
		if store.Get().RememberWindow {
			if g, ok := loadWindowGeom(dataDir, screensOf(app)); ok {
				opts.X, opts.Y, opts.Width, opts.Height = g.X, g.Y, g.W, g.H
			}
		}
		w := app.Window.NewWithOptions(opts)
		w.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
			app.Event.Emit(picker.DroppedEvent, e.Context().DroppedFiles())
		})
		// Wayland gives an app no say over where a re-shown window goes, so closing to the tray
		// destroys the window and showing builds a fresh one that the compositor places as new.
		w.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
			if store.Get().RememberWindow {
				x, y := w.Position()
				saveWindowGeom(dataDir, windowGeom{X: x, Y: y, W: w.Width(), H: w.Height()})
			}
			if !store.Get().KeepInTray {
				if quitSvc.AllowWindowClose() {
					return
				}
				quitSvc.RequestQuit()
				return
			}
			windowMu.Lock()
			windowGone = true
			windowMu.Unlock()
		})
		return w
	}
	window = newWindow()
	go func() {
		defer func() {
			if tidied.Pending() > 0 {
				emit(tidy.ReportEvent, nil)
			}
			tidied.Close()
		}()
		// Mods extracted before zip names were decoded may sit in folders the game cannot open. Extraction decodes
		// names now, so one clean pass over both trees is enough.
		repairedMarker := filepath.Join(dataDir, "names-repaired")
		if _, err := os.Stat(repairedMarker); err != nil {
			clean := true
			for _, root := range []string{filepath.Join(dataDir, "store"), filepath.Join(dataDir, "profiles")} {
				if n, err := archive.RepairNames(root); err != nil {
					clean = false
					log.Printf("repair names under %s: %v", root, err)
				} else if n > 0 {
					log.Printf("repaired %d file names under %s", n, root)
					tidied.Add("Repaired mod file names", filepath.Base(root), n)
				}
			}
			if clean {
				if err := datadir.WriteFile(repairedMarker, nil, 0o600); err != nil {
					log.Printf("repair names marker: %v", err)
				}
			}
		}
		if n, c, err := profiles.MigrateHistory(); err != nil {
			log.Printf("history migration: %v", err)
		} else if n > 0 || c > 0 {
			log.Printf("history migration: gzipped %d snapshots, counted %d histories", n, c)
			tidied.Add("Compressed profile history snapshots", "profiles", n)
		}
		// An unreadable profile.json stops collection: its keys are unknown, and their items must not be deleted.
		retention := store.Get().StoreUnusedFor()
		if keys, err := datasvc.KeepSet(profiles, retention != 0, keepSources); err != nil {
			log.Printf("store collect skipped: %v", err)
		} else {
			items.UnusedFor = retention
			if retention == 0 {
				items.UnusedFor = -1
			}
			if err := items.Collect(keys, now); err != nil {
				log.Printf("store collect: %v", err)
			}
			for g := range keys {
				if err := profiles.RefreshDependencies(g); err != nil {
					log.Printf("refresh dependencies: %s: %v", g, err)
				}
			}
		}
	}()
	go func() {
		failurePath := filepath.Join(dataDir, "cache", "components-failure")
		if b, err := fsx.ReadFile(failurePath); err == nil {
			if at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b))); err == nil && time.Since(at) < time.Hour {
				return
			}
		}
		if _, err := componentClient.Load(context.Background(), modMeta, updateKey); err != nil {
			log.Printf("components manifest unavailable; using bundled copy: %v", err)
			_ = os.MkdirAll(filepath.Dir(failurePath), 0o700)
			_ = fsx.WriteFile(failurePath, []byte(time.Now().Format(time.RFC3339Nano)), 0o600)
			return
		}
		_ = os.Remove(failurePath)
		if g, ok := componentClient.Game("stardew"); ok {
			nexus.Configure(g.Nexus.Domain, g.Nexus.ID)
		}
		// A fetched manifest can name a newer bridge than the one synced at startup from the bundled copy.
		loadersvc.SyncBundled(loaders, "stardew")
	}()
	windowClosed = func() bool {
		windowMu.Lock()
		defer windowMu.Unlock()
		return windowGone
	}
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
	startModUpdateBackground(updateCtx, svc, gamesSvc, profiles, problemsSvc, app)

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
		running := launches.Busy(settings.GameStardew)
		gameName := "Stardew Valley"
		if g := game.Find("stardew"); g != nil {
			gameName = g.Name()
		}
		if running {
			trayMenu.Add("Stop " + gameName).OnClick(func(*application.Context) {
				_ = launches.Stop("stardew")
			})
		}
		left := 0
		for _, item := range queueSvc.State().Items {
			if item.State != queue.StateDone && item.State != queue.StateSkipped && item.State != queue.StateCancelled {
				left++
			}
		}
		if left > 0 {
			trayMenu.Add(fmt.Sprintf("%d downloads…", left)).OnClick(func(*application.Context) {
				showWindow()
				app.Event.Emit("queue:open", nil)
			})
		}
		recent, _ := launches.RecentLaunches("stardew", 3)
		for _, row := range recent {
			item := trayMenu.Add("Play " + row.Name)
			if running {
				item.SetEnabled(false)
			} else {
				profileID := row.ProfileID
				item.OnClick(func(*application.Context) {
					app.Event.Emit(shortcut.RequestedEvent, shortcut.Request{Game: "stardew", Profile: profileID})
				})
			}
		}
		trayMenu.AddSeparator()
		trayMenu.Add("Quit").OnClick(func(*application.Context) {
			quitSvc.RequestQuit()
		})
		trayMenu.Update()
	}
	syncTray := func() {
		if !store.Get().KeepInTray && !store.Get().StartMinimised {
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
				if strings.HasSuffix(n.Title, " crashed") && !desktopnotify.Pref("desktopRunCrashed") {
					return
				}
				desktopnotify.Send(n.Title, n.Body, map[string]any{"game": n.Game, "profile": n.Profile})
			}
			// Menus are GTK objects: rebuilding one off the main thread, or twice at once from a burst of
			// events, leaves items without their native handle and panics.
			app.Event.On(launchsvc.StateEvent, func(*application.CustomEvent) {
				application.InvokeSync(refreshTrayMenu)
			})
		}
		refreshTrayMenu()
	}
	syncTray()
	app.Event.On(settings.ChangedEvent, func(*application.CustomEvent) {
		application.InvokeSync(syncTray)
		if err := lanSvc.SetEnabled(store.Get().LanSharing); err != nil {
			log.Printf("LAN sharing: %v", err)
		}
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

// serveNativeHost runs Mortar as the browser extension's native messaging host: each nxm link or Nexus collection
// page link is handed to a new Mortar process, which forwards it to the running one like any launch, or starts Mortar
// when none runs. The browser waits for the reply, so the child is not waited for.
func serveNativeHost() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if img := os.Getenv("APPIMAGE"); img != "" {
		exe = img
	}
	return nativehost.ServeFrom(os.Args[1:], os.Stdin, os.Stdout, func(link string) error {
		if !nxm.IsLink(link) && !sharesvc.IsCollectionURL(link) {
			return fmt.Errorf("not an nxm or collection link: %q", link)
		}
		return nativehost.Start(exe, link)
	})
}
