package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	goavscan "github.com/Rethunk-Tech/go-avscan"
	"github.com/Rethunk-Tech/go-safefs"
	tstore "github.com/Rethunk-Tech/go-thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/archivesvc"
	"github.com/Rethunk-Tech/mortar/internal/avscan"
	"github.com/Rethunk-Tech/mortar/internal/backdrop"
	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/browse"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/configsvc"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/desktopnotify"
	"github.com/Rethunk-Tech/mortar/internal/devsvc"
	"github.com/Rethunk-Tech/mortar/internal/folderwatch"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/loadersvc"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/metadata"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modpic"
	"github.com/Rethunk-Tech/mortar/internal/netstate"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
	"github.com/Rethunk-Tech/mortar/internal/picker"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savedgroups"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/scratch"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
	githubsrc "github.com/Rethunk-Tech/mortar/internal/source/github"
	itchsource "github.com/Rethunk-Tech/mortar/internal/source/itch"
	modiosource "github.com/Rethunk-Tech/mortar/internal/source/modio"
	nexussource "github.com/Rethunk-Tech/mortar/internal/source/nexus"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	modstore "github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/storecheck"
	"github.com/Rethunk-Tech/mortar/internal/support"
	"github.com/Rethunk-Tech/mortar/internal/syncsvc"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/tidy"
	"github.com/Rethunk-Tech/mortar/internal/tools"
	"github.com/Rethunk-Tech/mortar/internal/updatesvc"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// boot holds what Mortar's startup builds, shared by the phases run calls in order.
type boot struct {
	app             *application.App
	archivesSvc     *archivesvc.Service
	bisectSvc       *bisect.Service
	browseSvc       *browse.Service
	checkSvc        *storecheck.Service
	closeReady      func()
	componentClient *components.Client
	configSvc       *configsvc.Service
	dataDir         string
	dataSvc         *datasvc.Service
	devSvc          *devsvc.Service
	emit            func(name string, data any)
	gamesSvc        *game.Service
	gate            *startGate
	handoffs        chan application.SecondInstanceData
	home            string
	implemented     []string
	inTray          bool
	intents         *UIIntentService
	items           *modstore.Store
	keepSources     []datasvc.KeySource
	lanSvc          *lan.Service
	launches        *launchsvc.Service
	loaders         *loadersvc.Service
	modMeta         *meta.Client
	newWindow       func(hidden bool) *application.WebviewWindow
	nexusClient     *nexusmods.Client
	nexusPages      func(ctx context.Context, gameID string, ids []int) (map[int]nexusmods.Page, error)
	nexusServers    nexus.DownloadServers
	nexusSvc        *nexussvc.Service
	notifier        *notifications.NotificationService
	now             time.Time
	nxmSvc          *nxmsvc.Service
	openSvc         *opener.Service
	packs           *packsvc.Service
	pageAnswers     func() bool
	pick            *picker.Service
	pictures        *modpic.Cache
	plays           *shortcut.Service
	problemsSvc     *problems.Service
	profileSvc      *profile.Service
	profiles        *profile.Store
	queueSvc        *queue.Service
	quitSvc         *QuitService
	ready           chan struct{}
	savedGroupsSvc  *savedgroups.Service
	savesSvc        *savessvc.Service
	settled         chan struct{}
	shareSvc        *sharesvc.Service
	showWindow      func()
	shutdown        func()
	solo            bool
	stopQueue       context.CancelFunc
	store           *settings.Store
	supportSvc      *support.Service
	svc             *settings.Service
	syncSvc         *syncsvc.Service
	templatesSvc    *templates.Service
	tidied          *tidy.Collector
	toPage          func(name string, data any)
	toolsSvc        *tools.Service
	tsClient        *tstore.Client
	updates         *updatesvc.Service
	waitQueue       func()
	warmDetails     func(ctx context.Context, game string, src profile.Source, err error)
	watchHolds      *folderwatch.Holds
	window          *application.WebviewWindow
	windowClosed    func() bool
	windowGone      bool
	windowMu        sync.Mutex
}

func (c *boot) initInstanceState() {
	c.ready = make(chan struct{})
	c.plays = &shortcut.Service{}
	c.plays.SteamRoots = func() []string {
		if c.store == nil {
			return nil
		}
		return c.store.Get().LauncherRoots[game.LauncherSteam]
	}
	c.closeReady = sync.OnceFunc(func() { close(c.ready) })
	c.inTray = startedInTray(os.Args[1:])
	c.handoffs = make(chan application.SecondInstanceData, 256)
}

// serveHandoffs handles second-instance launches in order. A burst of nxm clicks starts one second instance per link; each launch
// is queued at once, so a slow handoff (waiting for startup, asking the window whether it is minimised) never holds up the next.
func (c *boot) serveHandoffs() {
	<-c.ready
	for d := range c.handoffs {
		if c.window == nil || c.nxmSvc == nil || c.shareSvc == nil {
			log.Printf("second instance: dropped, Mortar did not finish starting")
			continue
		}
		// An nxm link leaves an open window where it is, minimised or behind the browser, so a burst of clicks
		// keeps the browser in front; the window sends a desktop notification when a link needs a profile chosen.
		// Only a window closed or hidden to the tray is brought back, since nothing else could show the link.
		// A restart that asks for the tray finds Mortar already running and leaves its window where it is.
		if len(d.Args) == 2 && d.Args[1] == minimisedArg {
			continue
		}
		c.showForHandoff(d)
	}
}

func (c *boot) showForHandoff(d application.SecondInstanceData) {
	nxmLink := c.nxmSvc.Receive(d.Args)
	closed, visible := c.windowClosed(), !c.windowClosed() && c.window.IsVisible()
	if !c.shareSvc.Receive(sharesvc.InDir(d.Args, d.WorkingDir)) && !c.plays.Receive(d.Args) && nxmLink && !closed && visible {
		return
	}
	c.showWindow()
	log.Printf("second instance: window shown (was closed %v, visible %v)", closed, visible)
	// A rebuilt window is still loading its page, so only one that was already there is asked.
	if !closed && !c.pageAnswers() {
		log.Printf("second instance: the page did not answer within %s; reloading it", pageAnswerWait)
		c.window.ForceReload()
	}
}

func (c *boot) preflight() (bool, error) {
	var err error
	c.dataDir, err = dataDirOrRecover()
	if err != nil {
		return false, err
	}
	if err := requireWebView2(); err != nil {
		return false, err
	}
	// The single-instance handoff exits this process at once, so a Steam session for a running Mortar forwards its
	// request over the control channel instead and stays until the game closes, for Steam to count the playtime.
	if r, ok := shortcut.Parse(os.Args[1:]); ok && shortcut.SteamSession(os.Args[1:]) {
		if _, running := controlwire.Live(context.Background(), c.dataDir); running {
			if err := shortcut.Follow(c.dataDir, r, time.Second, steamSessionPatience); err != nil {
				log.Printf("steam session: %v", err)
			}
			return true, nil
		}
	}
	if forwardLaunch(c.dataDir, os.Args[1:]) {
		return true, nil
	}
	endStuckInstance(c.dataDir)
	return false, nil
}

func (c *boot) newApp() error {
	c.updates = &updatesvc.Service{}
	// Run by the shutdown hook or, when the graceful quit stalls, by the quit watchdog; whichever comes first.
	c.shutdown = sync.OnceFunc(func() {
		if c.lanSvc != nil {
			c.lanSvc.Shutdown()
		}
		desktopnotify.StopClicks()
		slog.Info("shutdown", "clean", true)
		_ = c.updates.ApplyOnQuit(context.Background())
	})
	currentSettings := func() settings.Settings {
		if c.store == nil {
			return settings.Defaults()
		}
		return c.store.Get()
	}
	c.app = application.New(application.Options{
		Name:         "Mortar",
		Icon:         appIcon,
		ErrorHandler: logAppError,
		// Server mode only: a request still running at quit (a bound call waiting on a locked keyring's prompt) is
		// abandoned after this long, not waited for 30 s by Wails' default.
		Server: application.ServerOptions{ShutdownTimeout: serverShutdownTimeout},
		// A bound call's error reaches the window with its usererr kind as the Wails cause.
		MarshalError: usererr.Marshal,
		PanicHandler: panicHandler(c.dataDir),
		// ApplicationID is the GtkApplication / Wayland app_id and the Linux desktop file id. It must not equal
		// UniqueID: both become D-Bus names, and GApplication also owns ApplicationID on the session bus.
		// Mortar decides itself whether closing the last window quits or leaves it in the tray.
		Linux: application.LinuxOptions{ApplicationID: appversion.AppID, DisableQuitOnLastWindowClosed: true},
		Windows: application.WindowsOptions{
			WebviewUserDataPath:           filepath.Join(c.dataDir, "webview"),
			DisableQuitOnLastWindowClosed: true,
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			Middleware: application.ChainMiddleware(
				game.ArtMiddleware(c.home, currentSettings),
				game.InstallArtMiddleware(c.home, currentSettings),
				game.ScreenshotMiddleware(c.home, currentSettings),
				profile.CoverMiddleware(func() *profile.Store { return c.profiles }),
				modpic.Middleware(func() *modpic.Cache { return c.pictures }),
				backdrop.Middleware(currentSettings, backdrop.SystemDefault, backdrop.DesktopWallpaper, filepath.Join(c.dataDir, "cache", "backdrop")),
			),
		},
		OnShutdown: c.shutdown,
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: singleInstanceID(c.dataDir),
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				log.Printf("second instance: %d args queued", len(d.Args)-1)
				c.handoffs <- d
			},
		},
	})
	c.openSvc = &opener.Service{Open: func(u string) error {
		if serverBuild {
			return opener.ErrOpenInPage
		}
		return c.app.Browser.OpenURL(u)
	}}
	if pid, ok := otherInstance(c.dataDir); ok {
		return fmt.Errorf("mortar (pid %d) is already running with the data folder %s", pid, c.dataDir)
	}
	return nil
}

func (c *boot) openLogs() error {
	// A windowsgui build has no stderr, so logs and fatal panics would end without a trace; both also go to
	// files in the data folder, the previous run's log kept beside the current one.
	if crash, err := safefs.OpenFile(filepath.Join(c.dataDir, "crash.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600); err == nil {
		_ = debug.SetCrashOutput(crash, debug.CrashOptions{})
		_ = crash.Close()
	}
	logPath := filepath.Join(c.dataDir, "mortar.log")
	_ = safefs.Rename(logPath, filepath.Join(c.dataDir, "mortar.prev.log"))
	if logFile, err := safefs.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600); err == nil {
		// The file comes first: MultiWriter stops at the first failing writer, and a GUI launch can have a dead
		// stderr, which would otherwise leave mortar.log without the clean-shutdown line crash detection reads.
		out := io.MultiWriter(&cappedWriter{w: logFile, left: maxLogBytes}, os.Stderr)
		log.SetOutput(out)
		slog.SetDefault(slog.New(slog.NewTextHandler(out, nil)))
		// Wails discards its own info lines without a logger, and those are the only trace of a WebView2 process
		// failure it chose not to recover from.
		c.app.Logger = slog.Default()
	}
	// A user's log is often the only report there is, so it starts by naming the build.
	log.Printf("mortar %s (build %s) on %s/%s", version, cmp.Or(appversion.Commit(), "unknown"), runtime.GOOS, runtime.GOARCH)
	support.StartStartupProfile(c.dataDir)
	startupStep("log open")
	c.tidied = &tidy.Collector{}
	if checkCrashLog(c.dataDir) {
		c.tidied.Add("crashLog", "Emptied the crash log after it was reported", "dataFolder", "", 1, "crash.log")
	}

	return nil
}

func (c *boot) openStores() error {
	var err error
	c.store, err = settings.Open()
	if err != nil {
		return err
	}
	i18n.SetLanguageSource(func() string { return c.store.Get().Language })
	applyGraphics(c.store.Get().Graphics)
	applyLinkChoices(c.store.Get())
	c.svc = settings.NewService(c.store)
	c.svc.ValidateGameFolder = game.ValidateFolder
	c.svc.ValidateLauncherRoot = game.ValidateLauncherRoot
	c.svc.ValidateImage = backdrop.Check

	c.gamesSvc = game.NewService(c.home, c.store)

	c.items, err = modstore.Open()
	if err != nil {
		return err
	}
	if removed, err := c.items.Cleanup(); err != nil {
		log.Printf("store cleanup: %v", err)
	} else {
		c.tidied.Add("tempFolders", "Removed leftover temporary folders", "store", "", 0, removed...)
	}

	c.items.SetScanner(
		func() goavscan.Scanner {
			v := c.store.Get()
			return avscan.New(avscan.Config{Mode: v.Antivirus, Socket: v.AntivirusSocket, Command: v.AntivirusCommand})
		},
		func(game, key string, err error) {
			v := c.store.Get()
			why, tell := avscan.Config{Mode: v.Antivirus, Socket: v.AntivirusSocket, Command: v.AntivirusCommand}.Notice(err)
			if !tell {
				return
			}
			log.Printf("antivirus scan of %s/%s did not finish: %s", game, key, why)
			if c.app != nil {
				c.app.Event.Emit("avscan:failed", map[string]string{"game": game, "key": key, "error": why})
			}
		},
	)
	c.profiles, err = profile.Open(c.items)
	if err != nil {
		return err
	}
	startupStep("settings, mod store and profiles open")
	if err := c.profiles.PurgeScratch(); err != nil {
		log.Printf("profiles: remove leftover crash-check copies: %v", err)
	}
	if n, err := scratch.Sweep(24 * time.Hour); err != nil {
		log.Printf("scratch: remove old unpack folders: %v", err)
	} else {
		log.Printf("scratch: removed %d old unpack folders", n)
	}
	c.profiles.Tidied = func(code, what, profileName, folder string) { c.tidied.Add(code, what, "", profileName, 1, folder) }
	// Every Thunderstore Driver on this cache folder shares the one client made here, under Mortar's own user agent.
	c.tsClient, err = (thunderstore.Driver{CacheDir: filepath.Join(c.dataDir, "cache")}).Open(version)
	if err != nil {
		return err
	}
	return nil
}

func (c *boot) profileHooks() error {
	c.profiles.Publisher = func(gameID, name, ver string) (string, bool) {
		key := share.SourceKeys(gameID)["thunderstore"]
		if key == "" {
			return "", false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		ns, ok, err := thunderstore.Driver{CacheDir: filepath.Join(c.dataDir, "cache")}.Publisher(ctx, key, name, ver, version)
		return ns, ok && err == nil
	}
	c.profiles.PackageLooks = func(gameID string, names []string) map[string]thunderstore.Look {
		key := share.SourceKeys(gameID)["thunderstore"]
		if key == "" {
			return nil
		}
		looks, err := thunderstore.Driver{CacheDir: filepath.Join(c.dataDir, "cache")}.CachedLooks(key, names)
		if err != nil {
			return nil
		}
		return looks
	}
	c.profiles.SourceFacts = func(gameID, kind string, installed map[string]string) map[string]source.Facts {
		e, ok := source.Get(kind)
		if !ok {
			return nil
		}
		cf, ok := e.Source.(source.CachedFacter)
		if !ok {
			return nil
		}
		facts, err := cf.CachedFacts(context.Background(), share.SourceKeys(gameID)[kind], installed)
		if err != nil {
			return nil
		}
		return facts
	}
	c.profiles.ShortcutRenamed = shortcut.Renamed
	c.profiles.ShortcutRemoved = shortcut.Removed
	c.plays.Covers = func(gameID, profileID string) ([]string, error) {
		covers, err := c.profiles.Covers(gameID, profileID)
		if err != nil {
			return nil, err
		}
		if all, err := c.profiles.List(gameID); err == nil {
			for _, p := range all {
				if p.Error != "" || p.ID != profileID {
					continue
				}
				switch p.Cover {
				case "cover.png", "cover.jpg", "cover.webp":
					covers = append([]string{filepath.Join(c.dataDir, "profiles", gameID, profileID, p.Cover)}, covers...)
				}
				break
			}
		}
		return covers, nil
	}
	return nil
}

func (c *boot) startLaunches() error {
	c.launches = launchsvc.NewService(c.home, c.store, c.profiles)
	c.gamesSvc.Running = c.launches.Busy
	c.profiles.Running = c.launches.Running
	c.profiles.GameRunning = c.launches.Busy
	c.bisectSvc = bisect.NewService(c.profiles, c.launches)
	c.modMeta = &meta.Client{CacheDir: filepath.Join(c.dataDir, "cache"), Version: version}
	source.Register(thunderstore.Driver{CacheDir: filepath.Join(c.dataDir, "cache")})
	source.Register(&githubsrc.Driver{CacheDir: filepath.Join(c.dataDir, "cache")})
	c.componentClient = components.NewClient(&http.Client{Timeout: 30 * time.Second})
	game.ConfigureComponents(c.componentClient)
	// The catalog an earlier run cached answers from the start; the network check (Load, below) runs behind the window.
	catalogFrom := "built-in"
	if c.componentClient.UseCached(c.modMeta, updateKey) {
		catalogFrom = "cached"
	}
	log.Printf("catalog: starting with the %s copy, serial %d", catalogFrom, game.CatalogSerial())
	components.UseSignedLists(c.componentClient, c.modMeta, updateKey)
	go c.launches.PrimeHiddenMessages(context.Background())
	game.CacheInstalls(installCacheTTL)
	c.loaders = loadersvc.NewService(c.home, c.store, c.items, c.profiles, c.componentClient)
	c.implemented = game.Implemented()
	loadersvc.Attach(c.loaders)
	c.launches.EnsureLoader = func(ctx context.Context, id, loaderID string, fromStart bool) error {
		_, err := c.loaders.Ensure(ctx, id, loaderID, fromStart)
		return err
	}
	c.loaders.OnReady = func(id string) { c.launches.MaybeSweep(context.Background(), id) }
	c.now = time.Now()
	if err := c.profiles.PurgeTrash(c.now); err != nil {
		log.Printf("purge trash: %v", err)
	}

	return nil
}

func (c *boot) nexusServices() error {
	var err error
	c.savesSvc, err = savessvc.NewService(c.home, c.profiles, c.store, c.modMeta)
	if err != nil {
		return err
	}
	c.savesSvc.Launches = c.launches
	c.launches.OnSavePlayed = c.savesSvc.NotePlayed

	c.nexusClient = nexusmods.New("Mortar", version)
	c.nexusClient.HTTP = nexus.ObservedHTTP(http.DefaultTransport)
	c.nexusServers = nexus.DownloadServers{
		Remember: func(shortNames []string) {
			if err := c.svc.RememberNexusDownloadServers(shortNames); err != nil {
				slog.Warn("could not record Nexus download servers", "err", err)
			}
		},
		Preferred: func() string { return c.svc.Get().NexusPreferredDownloadServer },
	}
	c.nexusSvc = nexussvc.NewService(c.store, c.nexusClient, c.modMeta)
	c.nexusSvc.Profiles = c.profiles
	c.nexusSvc.GitHub = &github.Client{}
	nexussource.CategoryNames = c.nexusSvc.CategoryNames

	return nil
}

func (c *boot) nxmSetup() error {
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
	if h, ok := any(nxmHandler).(interface{ WriteNativeHosts() error }); ok && c.store.Get().NxmHandled {
		if err := h.WriteNativeHosts(); err != nil {
			log.Printf("browser extension host: %v", err)
		}
	}
	c.nxmSvc = nxmsvc.NewService(c.store, nxmHandler)

	return nil
}

func (c *boot) pictureWiring() error {
	c.pictures = modpic.New(c.dataDir, &http.Client{Timeout: 30 * time.Second})
	c.pictures.Icons = c.profiles.PackageIcon
	if err := c.nexusSvc.UseDataDir(c.dataDir); err != nil {
		return err
	}
	c.emit = func(name string, data any) { c.app.Event.Emit(name, data) }
	netstate.OnChange = func() { c.app.Event.Emit("netstate:changed", nil) }
	secret.OnUnlocked = func() {
		c.app.Event.Emit("keyring:unlocked", nil)
		c.app.Event.Emit(nexussvc.ChangedEvent, nil)
	}
	c.plays.Emit = c.emit
	// warmDetails fills the detail dialog's cache after an install, while the account is known to be signed in and
	// online.
	c.warmDetails = func(ctx context.Context, game string, src profile.Source, err error) {
		if err == nil {
			go func() { _, _ = c.nexusSvc.Details(context.WithoutCancel(ctx), game, src.ModID) }()
			go c.pictures.Ensure(context.WithoutCancel(ctx), src.Picture)
		}
	}
	return nil
}

func (c *boot) openQueue() error {
	var err error
	c.queueSvc, err = queue.New(queue.Deps{
		Client:       func() (*nexusmods.Client, error) { return nexussvc.Authed(c.store, c.nexusClient) },
		ModCache:     c.modMeta.Cache(),
		NexusServers: c.nexusServers,
		Premium:      func() bool { return c.store.Get().NexusPremium },
		Install: func(ctx context.Context, game, profileID, path string, src profile.Source) (profile.InstallResult, error) {
			res, err := c.profiles.InstallSource(ctx, game, profileID, path, src)
			c.warmDetails(ctx, game, src, err)
			return res, err
		},
		Stored: func(game, key string) bool {
			_, err := c.items.Path(game, key)
			return err == nil
		},
		SourcesOf:      c.profiles.SourcesOf,
		StoredOverlay:  c.profiles.StoredOverlay,
		AllowUnscanned: c.profiles.AllowUnscanned,
		Stage:          c.profiles.StageGitHub,
		InstallStaged:  c.profiles.InstallStaged,
		InstallRemap:   c.profiles.InstallRemap,
		Newest: func(game, profileID string, modID, current int) int {
			p, err := c.profiles.Get(game, profileID)
			if err != nil {
				return 0
			}
			return profile.NewestFromPage(p, modID, current)
		},
		SamePage: func(game, profileID string, in profile.IncomingFile) (profile.MergeAsk, int, bool) {
			p, err := c.profiles.Get(game, profileID)
			if err != nil {
				return profile.MergeAsk{}, 0, false
			}
			return profile.SamePageAsk(p, in)
		},
		InstallExtra: func(ctx context.Context, game, profileID, entryKey, path string, src profile.Source) (profile.InstallResult, error) {
			res, err := c.profiles.InstallNexusExtra(ctx, game, profileID, entryKey, path, src)
			c.warmDetails(ctx, game, src, err)
			return res, err
		},
		Verify: func(ctx context.Context, id mod.ID, owner, repo string) (bool, error) {
			return github.Verify(ctx, c.modMeta, id.Local(), owner, repo)
		},
		Closure: func(ctx context.Context, gameID string, roots []tstore.Ref, deps thunderstore.Deps) ([]thunderstore.Resolved, error) {
			key := share.SourceKeys(gameID)["thunderstore"]
			if key == "" {
				return nil, fmt.Errorf("%s has no Thunderstore community", gameID)
			}
			d := thunderstore.Driver{CacheDir: filepath.Join(c.dataDir, "cache"), DropIns: components.KnownBrokenFor(ctx, gameID).DropIns()}
			return d.Closure(ctx, key, roots, version, deps)
		},
		ExactDependencies: func(game, profileID string) bool {
			p, err := c.profiles.Get(game, profileID)
			if err != nil {
				return false
			}
			return settings.ResolveAt(c.store.Get(), "dependencyVersions", settings.Scope{Game: game, Profile: profileID}, p.PrefOverrides()) == settings.DependencyExact
		},
		ModsUpTo: c.profiles.ModsUpTo,
		ThunderstoreHost: func() string {
			if h := c.store.Get().ThunderstoreHost; h != settings.ThunderstoreHostAuto {
				return "https://" + h
			}
			return ""
		},
		Held: func(game, profileID, pkg string) string {
			p, err := c.profiles.Get(game, profileID)
			if err != nil {
				return ""
			}
			for _, e := range p.Entries {
				if e.Package && e.Source.Kind == profile.KindThunderstore && strings.EqualFold(e.Source.Name, pkg) {
					return e.Source.Version
				}
			}
			return ""
		},
		InstallPackage: c.profiles.InstallSource,
		Direct:         queue.ResolveDirect,
		GitHub:         &github.Client{},
		OpenURL:        c.openSvc.OpenWeb,
		Running:        c.launches.Running,
		Emit:           c.emit,
		Dir:            c.dataDir,
		Changed:        func(st queue.State) { c.shareSvc.QueueChanged(st) },
		HistoryBatch:   c.profiles.RecordHistoryBatch,
		Remove: func(game, profileID, key string) error {
			_, err := c.profiles.RemoveEntry(game, profileID, key)
			return err
		},
		Everywhere: func(game, modKey string) []queue.EverywhereFailure {
			res, err := c.profiles.UpdateEverywhere(game, modKey, "latest")
			var failed []queue.EverywhereFailure
			for _, sk := range res.Failed() {
				failed = append(failed, queue.EverywhereFailure{Profile: sk.Name, Reason: sk.Reason})
			}
			if err != nil {
				failed = append(failed, queue.EverywhereFailure{Reason: err.Error()})
			}
			return failed
		},
		Parallel:     func() int { return c.store.Get().ParallelDownloads },
		KeepArchives: func() bool { return c.store.Get().KeepDownloadArchives },
		DownloadDir:  func() string { return c.store.Get().ArchiveDir() },
		RetryFetches: func() int {
			switch c.store.Get().AutoRetryDownloads {
			case settings.AutoRetry1:
				return 1
			case settings.AutoRetry3:
				return 3
			default:
				return 0
			}
		},
		VerifyNexusMD5: func() bool { return c.store.Get().VerifyNexusMD5 },
		Track: func(ctx context.Context, gameID string, modID int) {
			if !c.store.Get().AutoTrackNexus || modID <= 0 {
				return
			}
			_ = c.nexusSvc.Track(ctx, gameID, modID)
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *boot) wireQueue() error {
	c.launches.OnState = func() { c.queueSvc.HoldForPlay(c.launches.PausesDownloads()) }
	c.launches.InstallPackage = func(ctx context.Context, gameID, profileID, pkg string) error {
		return c.queueSvc.AddAndWait(ctx, []queue.Request{{Kind: queue.KindInstall, Game: gameID, Profile: profileID, Package: pkg}})
	}
	github.Limiter.Rate = func() int64 { return int64(c.store.Get().DownloadSpeedLimit) << 20 }
	c.nxmSvc.Route = c.queueSvc.Route
	c.notifier = notifications.New()
	desktopnotify.Setup(wailsNotifier{c.notifier}, c.nxmSvc.NotificationIcon, func(key string) bool {
		v, err := c.store.Get().Lookup(key)
		return err == nil && v == "true"
	})

	c.pick = &picker.Service{}
	c.profileSvc = profile.NewService(c.profiles, c.home, c.store)
	c.profileSvc.QueueProfileDeleted = c.queueSvc.SkipProfile
	c.profileSvc.QueueProfileRestored = c.queueSvc.RestoreProfile
	c.savesSvc.Enqueue = c.queueSvc.Add
	c.profileSvc.Version = version
	c.savedGroupsSvc = savedgroups.NewService(c.profiles, c.dataDir)
	c.watchHolds = folderwatch.NewHolds()
	c.devSvc = &devsvc.Service{Version: version, Home: c.home, Settings: c.svc, Profiles: c.profileSvc, CacheDir: filepath.Join(c.dataDir, "cache"), BuildLogDir: filepath.Join(c.dataDir, datadir.BuildLogs), Unwatch: c.watchHolds.Hold}
	c.devSvc.MigrateRecords()
	c.problemsSvc = problems.NewService(c.home, c.store, c.profiles, c.modMeta)
	c.problemsSvc.Runs = c.launches
	c.problemsSvc.SaveFarms = c.savesSvc.FarmTypes
	c.problemsSvc.Throttle = c.queueSvc.SourceSlot
	c.problemsSvc.Installing = c.queueSvc.InstallingInto
	c.problemsSvc.GitHub = &github.Client{}
	c.problemsSvc.DevPackages = c.devSvc.Packages
	c.problemsSvc.Emit = c.emit
	return nil
}

func (c *boot) problemsWiring() error {
	c.nexusPages = func(ctx context.Context, gameID string, ids []int) (map[int]nexusmods.Page, error) {
		details, err := c.nexusSvc.Prime(ctx, gameID, ids)
		pages := make(map[int]nexusmods.Page, len(details))
		for id, d := range details {
			pages[id] = d.Page
		}
		return pages, err
	}
	c.problemsSvc.NexusPages = c.nexusPages
	c.problemsSvc.NexusFiles = c.nexusFiles
	c.supportSvc = support.NewService(version, c.problemsSvc.Environment, c.home, c.profiles.ModsDir)
	c.supportSvc.RecentLog = c.recentLog
	return nil
}

func (c *boot) shareSync() error {
	var err error
	c.shareSvc = sharesvc.NewService(sharesvc.Deps{
		Profiles: c.profiles,
		Meta:     c.modMeta,
		Files: func(ctx context.Context, t nexusmods.Title, modID int) ([]nexusmods.File, error) {
			client, err := nexussvc.Authed(c.store, c.nexusClient)
			if err != nil {
				return nil, err
			}
			files, err := client.Files(ctx, t, modID)
			return files, nexus.Worded(err)
		},
		SignedIn:   func() bool { return c.store.Get().NexusUserID != 0 },
		Premium:    func() bool { return c.store.Get().NexusPremium },
		NexusPages: c.nexusPages,
		CollectionArchive: func(ctx context.Context, link string) ([]byte, error) {
			client, err := nexussvc.Authed(c.store, c.nexusClient)
			if err != nil {
				return nil, err
			}
			archive, err := client.PremiumCollectionArchive(ctx, link)
			return archive, nexus.Worded(err)
		},
		Env:          c.problemsSvc.Environment,
		Queue:        c.queueSvc,
		Unresolvable: c.queueSvc.Unresolvable,
		PackageIcons: func(gameID string, names []string) map[string]string {
			key := share.SourceKeys(gameID)["thunderstore"]
			if key == "" {
				return nil
			}
			icons, err := thunderstore.Driver{CacheDir: filepath.Join(c.dataDir, "cache")}.CachedIcons(key, names)
			if err != nil {
				return nil
			}
			return icons
		},
		PackFolder: func() string { return c.store.Get().PackFolder },
		Stored: func(gameID, key string) bool {
			_, err := c.items.Path(gameID, key)
			return err == nil
		},
		Dismissals: c.problemsSvc,
		Dir:        c.dataDir,
		Emit:       c.emit,
	})
	c.syncSvc, err = syncsvc.New(syncsvc.Deps{
		Source: syncsvc.NewSource(c.profiles, c.shareSvc),
		Folder: func(game string) string { return c.store.Get().SyncFolderFor(game) },
		Dir:    c.dataDir, Quiet: 5 * time.Second, Stall: syncScan, Emit: c.emit,
	})
	if err != nil {
		return err
	}
	c.lanSvc = lan.NewService(lan.Deps{
		Shares: c.shareSvc, Settings: c.store, Store: c.items, Version: version,
		Dir: c.dataDir, Emit: c.emit,
	})
	if err := c.lanSvc.Apply(); err != nil {
		log.Printf("LAN sharing: %v", err)
	}
	// Queue changes reach shareSvc, so links are routed only once both exist.
	c.nxmSvc.Receive(os.Args[1:])
	c.solo = c.plays.Start(os.Args[1:])
	c.shareSvc.Receive(sharesvc.InDir(os.Args[1:], sharesvc.LaunchDir()))
	c.shareSvc.QueueChanged(c.queueSvc.State())

	return nil
}

func (c *boot) quitWiring() error {
	var err error
	c.toolsSvc, err = tools.NewService(c.home, c.store, c.profiles)
	if err != nil {
		return err
	}

	c.intents = &UIIntentService{emit: func(name string, data any) { c.app.Event.Emit(name, data) }}
	c.quitSvc = &QuitService{
		app: c.app, queue: c.queueSvc, lan: c.lanSvc, launch: c.launches, exit: os.Exit, grace: quitGrace, busy: c.queueSvc.Installing,
		show: func() { c.showWindow() }, closed: func() bool { return c.windowClosed() },
	}
	c.quitSvc.WatchSignals()

	return nil
}

func (c *boot) dataServices() error {
	c.archivesSvc = archivesvc.NewService(archivesvc.Deps{
		Dirs: func() []string { return downloadDirs(c.store, c.dataDir) },
		Install: func(ctx context.Context, game, profileID, path string, src profile.Source) (profile.InstallResult, error) {
			if src.Kind == profile.KindPatreon || src.Kind == profile.KindItch {
				return c.profiles.InstallSaved(ctx, game, profileID, path, src)
			}
			if src.ModID > 0 {
				return c.profiles.InstallSource(ctx, game, profileID, path, src)
			}
			return c.profileSvc.InstallArchive(ctx, game, profileID, path)
		},
		HashCache: filepath.Join(c.dataDir, "cache", "archive-hashes.json"),
		Offer:     func(g string) bool { return settings.ToggleOn(c.store.Get().GamePrefs(g).OfferNewDownloads) },
		Seen:      func(g string) int64 { return c.store.Get().GamePrefs(g).LastDownloadsSeen },
		SetSeen:   c.store.RecordDownloadsSeen,
		Keys:      c.items.Keys,
		Profiles:  c.profiles.List,
		NexusMods: func() map[int]bool {
			ids := map[int]bool{}
			for _, h := range c.queueSvc.History() {
				if h.ModID > 0 {
					ids[h.ModID] = true
				}
			}
			return ids
		},
	})
	c.templatesSvc = templates.NewService(templates.Deps{
		Profiles: c.profiles, SavedGroups: c.savedGroupsSvc,
		GameSettings: c.launches.GameSettings, SetGameSettings: c.launches.SetGameSettings,
	}, c.dataDir)

	c.keepSources = []datasvc.KeySource{
		c.savedGroupsSvc.ReferencedStoreKeys, c.templatesSvc.ReferencedStoreKeys,
		func() (map[string][]string, error) { return c.queueSvc.StagedKeys(), nil },
	}
	c.dataSvc = datasvc.NewService(c.items, c.profiles, c.keepSources,
		c.queueSvc.Busy, c.lanSvc.Busy,
		c.launches.AnyBusy,
	)
	c.profileSvc.HealthKeep = func() (map[string][]string, error) { return datasvc.KeepSet(c.profiles, true, c.keepSources) }
	c.profileSvc.HealthJournals = c.launches.LeftoverJournals
	c.profileSvc.HealthShared = c.launches.SharedFiles
	c.profileSvc.HealthRecover = func(game string) error {
		_, err := c.launches.RecoverGameDeploys(context.Background(), game)
		return err
	}
	c.dataSvc.Restart = datasvc.RestartSelf
	c.dataSvc.OnCleanup = func() { c.profileSvc.FlagHealth(game.Implemented()...) }
	c.queueSvc.Landed = func(gameID string) { c.profileSvc.FlagHealth(gameID) }
	c.dataSvc.OnClearCache = c.problemsSvc.ForgetCached
	c.checkSvc = storecheck.New(storecheck.Deps{
		Items: c.items, Source: c.profiles.SourceOf, Add: c.queueSvc.Add, State: c.queueSvc.State,
		NexusMD5: func(ctx context.Context, gameID string, modID, fileID int) (string, error) {
			t, err := game.NexusTitle(gameID)
			if err != nil {
				return "", err
			}
			client, err := nexussvc.Authed(c.store, c.nexusClient)
			if err != nil {
				return "", err
			}
			files, err := client.Files(ctx, t, modID)
			err = nexus.Worded(err)
			for _, f := range files {
				if f.FileID == fileID {
					return f.MD5, err
				}
			}
			return "", err
		},
		ArchiveDir: func() string { return archiveDir(c.store, c.dataDir) },
		Busy:       func() bool { return c.launches.AnyBusy() || c.queueSvc.Active() },
		Emit:       c.emit,
	})
	c.problemsSvc.Damage = c.items.Damaged

	return nil
}

func (c *boot) browseConfig() error {
	c.browseSvc = browse.NewService(version, c.profileSvc)
	// A Thunderstore listing that was stale when searched is replaced in the background; open Browse windows search again.
	thunderstore.OnRefreshed(func(key string) {
		browse.Forget("thunderstore")
		c.emit(browse.RefreshedEvent, key)
	})
	c.packs = &packsvc.Service{Profiles: c.profiles, Queue: c.queueSvc, Folder: func() string { return c.store.Get().PackFolder }, Categories: c.tsClient.Categories}
	c.browseSvc.ShowAdult = func() bool { return c.store.Get().ShowAdultContent }
	c.browseSvc.Installed = c.profiles.Installed
	c.browseSvc.Compat = func(game string) func(ctx context.Context) (smapiweb.CompatIndex, error) {
		info, ok := components.Game(game)
		if !ok {
			return nil
		}
		if st := metadata.For(c.modMeta, info.Metadata).Status; st != nil {
			return st.CompatList
		}
		return nil
	}
	c.browseSvc.HiddenMod = func(game string, it browse.Item) bool {
		gp := c.store.Get().GamePrefs(game)
		return gp.HidesMod(it.Source, it.ID) || gp.HidesAuthor(it.Author)
	}
	c.browseSvc.SourceOrder = func(game string) []string {
		return c.store.Get().GamePrefs(game).SourceOrderIDs()
	}
	c.browseSvc.SourcesOff = func(game string) []string {
		return c.store.Get().GamePrefs(game).SourcesOffIDs()
	}
	c.configSvc = &configsvc.Service{
		Profiles: c.profiles,
		SetJSON: func(game, profileID string, uniqueID mod.ID, sets []profile.ConfigSet) error {
			return c.profileSvc.SetConfigValues(game, profileID, "", uniqueID, sets)
		},
		SetGmcm: func(game, profileID string, uniqueID mod.ID, page string, index int, value string) error {
			_, err := c.profileSvc.SetGmcmOption(game, profileID, uniqueID, page, index, value)
			return err
		},
		Running: c.profiles.Running,
	}
	// Host values the player kept are written into a config once its game has closed.
	c.launches.Unlocked = func() {
		queue.NotifyUnlocked(c.queueSvc)
		c.configSvc.FlushKept()
	}
	return nil
}

func (c *boot) registerServices() error {
	c.gate = &startGate{}
	for _, s := range []application.Service{
		application.NewService(c.gate),
		application.NewService(c.configSvc),
		application.NewService(c.svc),
		application.NewService(c.gamesSvc),
		application.NewService(c.profileSvc),
		application.NewService(c.loaders),
		application.NewService(c.launches),
		application.NewService(c.pick),
		application.NewService(c.openSvc),
		application.NewService(c.savedGroupsSvc),
		application.NewService(c.templatesSvc),
		application.NewService(c.archivesSvc),
		application.NewService(c.savesSvc),
		application.NewService(c.plays),
		application.NewService(c.nexusSvc),
		application.NewService(&itchsource.Service{Store: c.store}),
		application.NewService(&modiosource.Service{Store: c.store}),
		application.NewService(c.nxmSvc),
		application.NewService(c.notifier),
		application.NewService(c.problemsSvc),
		application.NewService(c.queueSvc),
		application.NewService(c.shareSvc),
		application.NewService(c.syncSvc),
		application.NewService(c.lanSvc),
		application.NewService(c.supportSvc),
		application.NewService(c.updates),
		application.NewService(c.bisectSvc),
		application.NewService(c.dataSvc),
		application.NewService(c.toolsSvc),
		application.NewService(c.checkSvc),
		application.NewService(&tidy.Service{Report: c.tidied}),
		application.NewService(c.quitSvc),
		application.NewService(c.intents),
		application.NewService(c.browseSvc),
		application.NewService(&netstate.Service{}),
		application.NewService(c.packs),
		application.NewService(c.devSvc),
	} {
		c.app.RegisterService(s)
	}

	return nil
}

func (c *boot) startBackground(updateCtx, queueCtx context.Context) error {
	if packaged == "" {
		if exe, err := os.Executable(); err == nil {
			packaged = updatesvc.PackagedBy(exe)
		}
	}
	if err := updatesvc.Configure(c.updates, c.app.Updater, version, updateKey, application.System.IsServer(), packaged, func() bool {
		return c.store.Get().IncludeBetaReleases
	}, c.dataDir); err != nil {
		return err
	}
	c.updates.AutoInstall = func() bool { return c.store.Get().AutoInstallMortar() }
	c.updates.StartBackground(updateCtx, c.emit)
	c.savesSvc.Emit = c.emit
	go c.savesSvc.RunScheduledBackups(updateCtx)
	c.settled = make(chan struct{})
	c.problemsSvc.AfterFirstCheck = func() { close(c.settled) }
	c.profileSvc.Emit = c.emit
	go c.profileSvc.RunHealthChecks(updateCtx, c.settled)
	launchsvc.SetLife(c.launches, queueCtx)
	go storecheck.Run(queueCtx, c.checkSvc)
	go c.syncSvc.Run(queueCtx, syncScan)
	go trimWhenQuiet(queueCtx)
	watchLibraryFolders(queueCtx, c.home, c.dataDir, c.store, c.watchHolds, func(name string, data any) {
		if name == syncsvc.FolderEvent {
			c.syncSvc.Kick()
			return
		}
		c.emit(name, data)
	})
	c.waitQueue = queue.Run(queueCtx, c.queueSvc, c.nxmSvc.Assigned)
	c.quitSvc.finish = func() {
		c.shutdown()
		c.stopQueue()
	}
	return nil
}

func (c *boot) serveControl(queueCtx context.Context) error {
	ctl := &control.Services{
		Version: version, Settings: c.store, SettingsSvc: c.svc, Games: c.gamesSvc, Store: c.profiles, Profiles: c.profileSvc,
		Problems: c.problemsSvc, Launches: c.launches, Saves: c.savesSvc, Queue: c.queueSvc, Tools: c.toolsSvc, SavedGroups: c.savedGroupsSvc,
		Nexus: c.nexusSvc, Shares: c.shareSvc, Data: c.dataSvc, Plays: c.plays, Loaders: c.loaders, Dev: c.devSvc, Templates: c.templatesSvc, Archives: c.archivesSvc, Browse: c.browseSvc, Bisect: c.bisectSvc, StoreCheck: c.checkSvc, Lan: c.lanSvc, Sync: c.syncSvc, Updates: c.updates, Nxm: c.nxmSvc, Support: c.supportSvc, Packs: c.packs,
		Emit: func(name string, data any) {
			if name == control.InstallAskEvent {
				c.toPage(name, data)
				return
			}
			c.emit(name, data)
		},
		Quit: c.quitSvc.ConfirmQuit, QuitBlocker: c.quitSvc.BusySummary,
		Handoff: func(args []string) { c.handoffs <- application.SecondInstanceData{Args: args} },
	}
	go func() {
		if err := control.Serve(queueCtx, c.dataDir, version, ctl.Handle); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("control: %v", err)
		}
	}()
	h := wailsHost{c.app}
	c.svc.App = h
	c.loaders.App = h
	c.launches.App = h
	if err := c.launches.RecoverGameSettings(); err != nil {
		log.Printf("game settings restore: %v", err)
		c.app.Event.Emit(launchsvc.SettingsRestoreWarningEvent, launchsvc.SettingsRestoreWarning{Error: err.Error()})
	}
	recovered, err := c.launches.RecoverDeploys(queueCtx)
	if err != nil {
		log.Printf("deploy recovery: %v", err)
	}
	c.profileSvc.FlagHealth(recovered...)
	return nil
}

// attachHost gives the services their app handle and starts what needs them.
func (c *boot) attachHost() error {
	h := wailsHost{c.app}
	for _, id := range c.implemented {
		loadersvc.EnsureExisting(c.loaders, id)
		launchsvc.SweepOnStart(c.launches, id)
	}
	c.pick.App = h
	c.nexusSvc.App = h
	go nexussvc.SettleSignIn(context.Background(), c.nexusSvc)
	c.nxmSvc.App = h
	c.shareSvc.App = h
	c.packs.App = h
	c.supportSvc.App = h
	return nil
}

func (c *boot) windowSetup() error {
	onClick := func(id string, data map[string]any) {
		gameID, profileID, tab := launchsvc.NoticeProfileFromResponse(id, data)
		if profileID != "" && gameID != "" {
			c.toPage(launchsvc.NoticeClickEvent, launchsvc.NoticeClick{Game: gameID, Profile: profileID, Tab: tab})
			return
		}
		c.showWindow()
	}
	c.notifier.OnNotificationResponse(func(result notifications.NotificationResult) {
		onClick(result.Response.ID, result.Response.UserInfo)
	})
	if flatpak.Inside() {
		if err := desktopnotify.RouteClicks(context.Background(), onClick); err != nil {
			slog.Warn("portal notification clicks", "err", err)
		}
	}
	c.app.Event.On(lan.ArrivedEvent, func(e *application.CustomEvent) {
		a, ok := e.Data.(lan.Arrival)
		if !ok || (!c.windowClosed() && c.window.IsVisible()) {
			return
		}
		desktopnotify.Send(desktopnotify.KindOther, i18n.T("{name} wants to share a profile", "name", a.Sender), a.ProfileName, nil)
	})

	// Only the first window honours start-minimised and play mode; a rebuilt one is always asked for.
	c.newWindow = func(hidden bool) *application.WebviewWindow {
		opts := application.WebviewWindowOptions{
			Title:            "Mortar",
			Width:            defaultWindowWidth,
			Height:           defaultWindowHeight,
			MinWidth:         minWindowWidth,
			MinHeight:        minWindowHeight,
			Frameless:        !c.store.Get().SystemTitleBar,
			BackgroundType:   application.BackgroundTypeSolid,
			BackgroundColour: application.NewRGBA(25, 25, 30, 255),
			EnableFileDrop:   true,
			URL:              "/",
			Hidden:           hidden,
		}
		if c.store.Get().RememberWindow {
			if g, ok := loadWindowGeom(c.dataDir, screensOf(c.app)); ok {
				opts.X, opts.Y, opts.Width, opts.Height = g.X, g.Y, g.W, g.H
				if g.Max {
					opts.StartState = application.WindowStateMaximised
				}
			}
		}
		w := c.app.Window.NewWithOptions(opts)
		w.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
			c.app.Event.Emit(picker.DroppedEvent, e.Context().DroppedFiles())
		})
		// Wayland gives an app no say over where a re-shown window goes, so closing to the tray
		// destroys the window and showing builds a fresh one that the compositor places as new.
		w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			if c.store.Get().RememberWindow && !c.plays.PlayMode() {
				x, y := w.Position()
				prev, havePrev := loadWindowGeom(c.dataDir, nil)
				saveWindowGeom(c.dataDir, closingGeom(prev, havePrev, w.IsMaximised(), windowGeom{X: x, Y: y, W: w.Width(), H: w.Height()}))
			}
			if !c.quitSvc.OnWindowClose(c.store.Get().KeepInTray, c.plays.PlayMode()) {
				e.Cancel()
				return
			}
			if !c.store.Get().KeepInTray || c.plays.PlayMode() {
				return
			}
			c.windowMu.Lock()
			c.windowGone = true
			c.windowMu.Unlock()
		})
		return w
	}
	return nil
}

func (c *boot) openWindow() error {
	startupStep("services ready")
	c.window = c.newWindow(c.store.Get().StartMinimised || c.plays.PlayMode() || c.inTray)
	pageLoaded := sync.OnceFunc(func() { startupStep("page loaded") })
	c.window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) { pageLoaded() })
	// Bundled mods are checked behind the open window: the check finds every game's installs, and may download a bridge.
	go func() {
		start := time.Now()
		for _, id := range c.implemented {
			loadersvc.SyncBundled(context.Background(), c.loaders, id)
		}
		log.Printf("bundled mods: %d games checked in %s", len(c.implemented), since(start))
	}()
	return nil
}

func (c *boot) playMode() error {
	c.plays.Window = func(m shortcut.WindowMode) {
		switch m {
		case shortcut.WindowPrompt:
			c.window.SetMinSize(promptWindowWidth, promptWindowHeight)
			c.window.SetSize(promptWindowWidth, promptWindowHeight)
			c.window.Center()
			c.window.Show().Focus()
		case shortcut.WindowHidden:
			c.window.Hide()
		case shortcut.WindowFull:
			c.window.SetMinSize(minWindowWidth, minWindowHeight)
			c.window.SetSize(defaultWindowWidth, defaultWindowHeight)
			c.window.Center()
			c.showWindow()
		}
	}
	if c.solo {
		c.app.Event.On(launchsvc.StateEvent, func(e *application.CustomEvent) {
			st, ok := e.Data.(launchsvc.Status)
			if ok && (st.State == launchsvc.Running || st.State == launchsvc.Idle) && c.plays.Observe(st.Game, st.State == launchsvc.Running) {
				log.Printf("play mode: %s closed; exiting", st.Game)
				c.quitSvc.ConfirmQuit()
			}
		})
	}
	return nil
}

func (c *boot) storeMaintenance() error {
	go func() {
		start := time.Now()
		defer func() {
			log.Printf("store maintenance: done in %s", since(start))
			if c.tidied.Pending() > 0 {
				c.emit(tidy.ReportEvent, nil)
			}
			c.tidied.Close()
		}()
		// An unreadable profile.json stops collection: its keys are unknown, and their items must not be deleted.
		retention := c.store.Get().StoreUnusedFor()
		if keys, err := datasvc.KeepSet(c.profiles, retention != 0, c.keepSources); err != nil {
			log.Printf("store collect skipped: %v", err)
		} else {
			c.items.UnusedFor = retention
			if retention == 0 {
				c.items.UnusedFor = -1
			}
			if err := c.items.Collect(keys, c.now); err != nil {
				log.Printf("store collect: %v", err)
			}
			// Rollback snapshots count here whatever the retention setting, unlike for the age-based pass.
			if withHistory, err := datasvc.KeepSet(c.profiles, true, c.keepSources); err != nil {
				log.Printf("store drop superseded skipped: %v", err)
			} else if err := c.items.DropSuperseded(withHistory); err != nil {
				log.Printf("store drop superseded: %v", err)
			}
			for g := range keys {
				if err := c.profiles.RefreshDependencies(g); err != nil {
					log.Printf("refresh dependencies: %s: %v", g, err)
				}
				// The window may have read the profiles before the refresh wrote them.
				c.emit(control.ChangedEvent, g)
			}
		}
		for _, g := range game.Catalog() {
			if err := c.profiles.FillPackageLooks(g.ID); err != nil {
				log.Printf("package icons and categories: %s: %v", g.ID, err)
			}
		}
	}()
	return nil
}

// tellIfCatalogStale lets the window offer a retry when the catalog in use was signed long ago, whether the refresh
// failed or was skipped.
func (c *boot) tellIfCatalogStale() {
	m := c.componentClient.Manifest()
	if c.app != nil && m.Stale(time.Now()) {
		c.app.Event.Emit(game.CatalogStaleEvent, game.CatalogStale{Issued: m.IssuedAt().Format(time.RFC3339)})
	}
}

func (c *boot) catalogRefresh() error {
	go func() {
		defer c.tellIfCatalogStale()
		failurePath := filepath.Join(c.dataDir, "cache", "components-failure")
		if b, err := safefs.ReadFile(failurePath); err == nil {
			if at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b))); err == nil && time.Since(at) < time.Hour {
				log.Printf("catalog: not checked, the last check failed at %s; serial %d in use", at.Format(time.RFC3339), game.CatalogSerial())
				return
			}
		}
		start := time.Now()
		m, err := c.componentClient.Load(context.Background(), c.modMeta, updateKey)
		log.Printf("catalog: serial %d after %s", m.Serial, since(start))
		if err != nil {
			if errors.Is(err, components.ErrBadSignature) || errors.Is(err, components.ErrRollback) {
				slog.Warn("components manifest refused; keeping the last trusted copy", "err", err)
			} else {
				log.Printf("components manifest unavailable; using bundled copy: %v", err)
			}
			_ = os.MkdirAll(filepath.Dir(failurePath), 0o700)
			_ = safefs.WriteFile(failurePath, []byte(time.Now().Format(time.RFC3339Nano)), 0o600)
			return
		}
		_ = safefs.Remove(failurePath)
		// What was built from the startup catalog follows the fetched one: save layouts, problems checked against it,
		// a newer bridge than the one synced from the bundled copy, package looks, and the window's game list.
		if err := c.savesSvc.Reload(); err != nil {
			log.Printf("saves: reload after the catalog: %v", err)
		}
		c.problemsSvc.ForgetCached()
		for _, id := range game.Implemented() {
			loadersvc.SyncBundled(context.Background(), c.loaders, id)
		}
		for _, g := range game.Catalog() {
			if err := c.profiles.FillPackageLooks(g.ID); err != nil {
				log.Printf("package icons and categories: %s: %v", g.ID, err)
			}
		}
		c.emit(game.CatalogChangedEvent, m.Serial)
	}()
	return nil
}

func (c *boot) windowFuncs(updateCtx context.Context) error {
	c.windowClosed = func() bool {
		c.windowMu.Lock()
		defer c.windowMu.Unlock()
		return c.windowGone
	}
	c.showWindow = func() {
		// Window calls can hop to the main thread, where the closing hook waits on windowMu: only the flag is held.
		c.windowMu.Lock()
		rebuild := c.windowGone
		c.windowGone = false
		c.windowMu.Unlock()
		if rebuild {
			c.window = c.newWindow(false)
		} else {
			c.window.Restore()
		}
		c.window.Show().Focus()
	}
	c.app.Event.On(game.CatalogRetryEvent, func(*application.CustomEvent) {
		_ = safefs.Remove(filepath.Join(c.dataDir, "cache", "components-failure"))
		_ = c.catalogRefresh()
	})
	pongs := make(chan struct{}, 1)
	c.app.Event.On(windowPongEvent, func(*application.CustomEvent) {
		select {
		case pongs <- struct{}{}:
		default:
		}
	})
	c.pageAnswers = func() bool {
		select {
		case <-pongs:
		default:
		}
		c.app.Event.Emit(windowPingEvent, "")
		select {
		case <-pongs:
			return true
		case <-time.After(pageAnswerWait):
			return false
		}
	}
	// toPage hands an event to the page, showing the window first; one closed to the tray has no page to hear it, so
	// the event waits for the rebuilt page to replay it.
	c.toPage = func(name string, data any) {
		if c.windowClosed() {
			c.intents.Queue(name, data)
			c.showWindow()
			return
		}
		c.showWindow()
		c.app.Event.Emit(name, data)
	}
	startModUpdateBackground(updateCtx, c.svc, c.gamesSvc, c.profiles, c.problemsSvc, c.app)

	return nil
}

// trayView is what the tray menu shows, read off the main thread: which games run, the downloads left and each
// game's recent launches. Reading it inside InvokeSync would hold the window's thread on the process table.
type trayView struct {
	ids    []string
	busy   map[string]bool
	left   int
	recent map[string][]launchsvc.RecentLaunch
}

// trayUI owns the system tray and its menu.
type trayUI struct {
	c    *boot
	tray *application.SystemTray
	menu *application.Menu
	// One rebuild at a time, in event order, so a burst of state changes cannot leave an older menu in place.
	rebuildMu sync.Mutex
	// Set and cleared on the main thread with the tray; read by rebuilds started from other goroutines.
	on        atomic.Bool
	shownLeft atomic.Int64
	// Wails runs each listener call on its own goroutine; the remembered values below are read and written in turn.
	seen           sync.Mutex
	shown          bool
	systemTitleBar bool
	language       string
}

func (c *boot) setupTray() error {
	t := &trayUI{c: c}
	// The Play submenus and the window's profile list name each game's profiles, whichever path changed them (the
	// window, the command line, a share or sync). The store calls this under its lock, and reading the profiles takes it
	// again, so the rebuild runs apart.
	// Registered once, whether or not a tray is shown: rebuild does nothing without one. Menus are GTK objects, and
	// rebuild builds them on the main thread one at a time.
	c.app.Event.On(launchsvc.StateEvent, func(*application.CustomEvent) { t.rebuild() })
	// The queue reports every progress step; only a change in how many are left changes the menu.
	c.app.Event.On(queue.ChangedEvent, func(*application.CustomEvent) {
		if left := int64(t.downloadsLeft()); t.shownLeft.Swap(left) != left {
			t.rebuild()
		}
	})
	// A crash notice depends only on its own setting, not on the tray.
	c.launches.NotifyRunEnd = func(n launchsvc.RunEndNotice) {
		if !desktopnotify.Pref("desktopRunCrashed") {
			return
		}
		desktopnotify.Send(desktopnotify.KindRunEnd, n.Title, n.Body, map[string]any{"game": n.Game, "profile": n.Profile})
	}
	c.profiles.ListChanged = func(game string) {
		go t.rebuild()
		c.emit(control.ChangedEvent, game)
	}
	t.sync()
	t.shown = trayWanted(c.store.Get(), c.inTray)
	t.systemTitleBar = c.store.Get().SystemTitleBar
	t.language = c.store.Get().Language
	c.app.Event.On(settings.ChangedEvent, func(*application.CustomEvent) { t.settingsChanged() })
	return nil
}

func (t *trayUI) downloadsLeft() int {
	left := 0
	for _, item := range t.c.queueSvc.State().Items {
		if item.State != queue.StateDone && item.State != queue.StateSkipped && item.State != queue.StateCancelled {
			left++
		}
	}
	return left
}

func (t *trayUI) read() trayView {
	v := trayView{ids: game.Implemented(), busy: map[string]bool{}, recent: map[string][]launchsvc.RecentLaunch{}}
	for _, id := range t.c.launches.BusyAmong(v.ids) {
		v.busy[id] = true
	}
	v.left = t.downloadsLeft()
	for _, id := range v.ids {
		if recent, _ := t.c.launches.RecentLaunches(id, 3); len(recent) > 0 {
			v.recent[id] = recent
		}
	}
	return v
}

func trayGameName(id string) string {
	if g := game.Find(id); g != nil {
		return g.Name()
	}
	return id
}

func (t *trayUI) build(v trayView) {
	if t.menu == nil {
		return
	}
	t.menu.Clear()
	t.menu.Add(i18n.T("Show Mortar")).OnClick(func(*application.Context) {
		t.c.showWindow()
	})
	// A divider opens each group under Show Mortar; Quit's own divider closes the last one.
	running := false
	group := func(open *bool) {
		if !*open {
			t.menu.AddSeparator()
			*open = true
		}
	}
	t.addStops(v, func() { group(&running) })
	if v.left > 0 {
		group(&running)
		t.menu.Add(i18n.T("Downloads: {count}", "count", strconv.Itoa(v.left))).OnClick(func(*application.Context) {
			t.c.toPage("queue:open", nil)
		})
	}
	played := false
	t.addPlay(v, func() { group(&played) })
	t.menu.AddSeparator()
	t.menu.Add(i18n.T("Quit")).OnClick(func(*application.Context) {
		t.c.quitSvc.RequestQuit()
	})
	applyTrayMenu(t.tray, t.menu)
}

func (t *trayUI) addStops(v trayView, group func()) {
	for _, id := range v.ids {
		if !v.busy[id] {
			continue
		}
		group()
		t.menu.Add(i18n.T("Stop {game}", "game", trayGameName(id))).OnClick(func(*application.Context) {
			if err := t.c.launches.Stop(context.Background(), id); err != nil {
				log.Printf("tray: stop %s: %v", id, err)
				desktopnotify.Send(desktopnotify.KindOther, i18n.T("Could not stop {game}", "game", trayGameName(id)), err.Error(), nil)
			}
		})
	}
}

// addPlay puts each game's recent profiles under that game, so two games' "Main" profiles cannot be confused.
func (t *trayUI) addPlay(v trayView, group func()) {
	for _, id := range v.ids {
		if len(v.recent[id]) == 0 {
			continue
		}
		group()
		sub := t.menu.AddSubmenu(i18n.T("Play {game}", "game", trayGameName(id)))
		for _, row := range v.recent[id] {
			item := sub.Add(row.Name)
			if v.busy[id] {
				item.SetEnabled(false)
				continue
			}
			profileID := row.ProfileID
			item.OnClick(func(*application.Context) {
				t.c.plays.Receive([]string{shortcut.Arg(id, profileID)})
				t.c.showWindow()
			})
		}
	}
}

func (t *trayUI) rebuild() {
	if !t.on.Load() {
		return
	}
	t.rebuildMu.Lock()
	defer t.rebuildMu.Unlock()
	v := t.read()
	t.c.gate.run(func() { t.build(v) })
}

func (t *trayUI) sync() {
	c := t.c
	if !trayWanted(c.store.Get(), c.inTray) {
		t.on.Store(false)
		if t.tray != nil {
			t.tray.Destroy()
			t.tray = nil
			t.menu = nil
		}
		return
	}
	if t.tray == nil {
		t.tray = c.app.SystemTray.New().SetIcon(appIcon)
		// Without a label the Linux tray item is titled "Wails", and Windows shows no name on hover.
		t.tray.SetLabel("Mortar")
		t.tray.SetTooltip("Mortar")
		t.menu = c.app.NewMenu()
		t.tray.SetMenu(t.menu)
		t.tray.OnClick(func() {
			c.showWindow()
		})
	}
	t.on.Store(true)
	t.build(t.read())
}

func (t *trayUI) settingsChanged() {
	c := t.c
	t.seen.Lock()
	defer t.seen.Unlock()
	if want := c.store.Get().SystemTitleBar; want != t.systemTitleBar {
		t.systemTitleBar = want
		// A window closed to the tray is rebuilt with the setting when shown.
		if !c.windowClosed() {
			c.window.SetFrameless(!want)
		}
	}
	// Only the tray settings change what the tray is; a rebuild reads every game's launches.
	if want := trayWanted(c.store.Get(), c.inTray); want != t.shown {
		t.shown = want
		c.gate.run(t.sync)
	} else if lang := c.store.Get().Language; lang != t.language {
		// The menu's words are translated when it is built.
		t.language = lang
		go t.rebuild()
	}
	applyLinkChoices(c.store.Get())
	if err := c.lanSvc.Apply(); err != nil {
		log.Printf("LAN sharing: %v", err)
	}
}

// nexusFiles lists each mod's files from the signed-in account, filling gaps and successor links from the cache.
func (c *boot) nexusFiles(ctx context.Context, t nexusmods.Title, ids []int) (map[int][]nexusmods.BatchFile, error) {
	var out map[int][]nexusmods.BatchFile
	var err error
	if client, authErr := nexussvc.Authed(c.store, c.nexusClient); authErr == nil {
		out, err = client.FilesOf(ctx, t, ids)
		err = nexus.Worded(err)
	}
	if out == nil {
		out = map[int][]nexusmods.BatchFile{}
	}
	var missing []int
	for _, id := range ids {
		if len(out[id]) == 0 {
			missing = append(missing, id)
		}
	}
	cached := c.nexusSvc.CachedFiles(t, ids)
	for _, id := range missing {
		if len(cached[id]) > 0 {
			out[id] = cached[id]
		}
	}
	for id, files := range out {
		successors := map[int]int{}
		for _, f := range cached[id] {
			successors[f.FileID] = f.ReplacedBy
		}
		for i := range files {
			files[i].ReplacedBy = cmp.Or(files[i].ReplacedBy, successors[files[i].FileID])
		}
	}
	return out, err
}

// recentLog is the tail of a profile's launch log, for a support report.
func (c *boot) recentLog(gameID, profileID string) string {
	entries, err := c.launches.Lines(gameID, profileID)
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
