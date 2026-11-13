package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/cli"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/devsvc"
	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/loadersvc"
	"github.com/Rethunk-Tech/mortar/internal/nativehost"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/nxm"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/picker"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
	"github.com/Rethunk-Tech/mortar/internal/syncsvc"
	"github.com/Rethunk-Tech/mortar/internal/updatesvc"
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

// updateKey verifies release signatures; its private half never enters the repository (docs/release.md).
//
//go:embed build/updater/public.key
var updateKey []byte

// registerEvents declares the custom events for the binding generator and the runtime's payload checks.
func registerEvents() {
	application.RegisterEvent[uint64](game.CatalogChangedEvent)
	application.RegisterEvent[launchsvc.Status](launchsvc.StateEvent)
	application.RegisterEvent[launchsvc.Lines](launchsvc.LineEvent)
	application.RegisterEvent[launchsvc.Progress](launchsvc.ProgressEvent)
	application.RegisterEvent[launchsvc.Crash](launchsvc.CrashEvent)
	application.RegisterEvent[launchsvc.LiveRun](launchsvc.LiveEvent)
	application.RegisterEvent[loadersvc.Progress](loadersvc.ProgressEvent)
	application.RegisterEvent[[]string](picker.DroppedEvent)
	application.RegisterEvent[settings.Settings](settings.ChangedEvent)
	application.RegisterEvent[string](control.ChangedEvent)
	application.RegisterEvent[string](windowPingEvent)
	application.RegisterEvent[string](windowPongEvent)
	application.RegisterEvent[control.InstallAsk](control.InstallAskEvent)
	application.RegisterEvent[shortcut.Request](shortcut.RequestedEvent)
	application.RegisterEvent[nexussvc.Account](nexussvc.ChangedEvent)
	application.RegisterEvent[string](nexussvc.TrackedEvent)
	application.RegisterEvent[nexussvc.SSOState](nexussvc.SSOEvent)
	application.RegisterEvent[queue.State](queue.ChangedEvent)
	application.RegisterEvent[queue.Progress](queue.ProgressEvent)
	application.RegisterEvent[queue.EverywhereFailed](queue.EverywhereFailedEvent)
	application.RegisterEvent[nxmsvc.Arrival](nxmsvc.ArrivedEvent)
	application.RegisterEvent[nxmsvc.Rejection](nxmsvc.RejectedEvent)
	application.RegisterEvent[string](ModsFolderEvent)
	application.RegisterEvent[Folder](ExtraFolderEvent)
	application.RegisterEvent[string](DownloadsEvent)
	application.RegisterEvent[Folder](devsvc.SourceEvent)
	application.RegisterEvent[[]syncsvc.Offer](syncsvc.OffersEvent)
	application.RegisterEvent[[]syncsvc.Stall](syncsvc.StalledEvent)
	application.RegisterEvent[sharesvc.Arrival](sharesvc.ArrivedEvent)
	application.RegisterEvent[lan.Arrival](lan.ArrivedEvent)
	application.RegisterEvent[lan.Expired](lan.ExpiredEvent)
	application.RegisterEvent[lan.TransferProgress](lan.TransferProgressEvent)
	application.RegisterEvent[lan.PairedPeer](lan.PairedEvent)
	application.RegisterEvent[launchsvc.NoticeClick](launchsvc.NoticeClickEvent)
	application.RegisterEvent[launchsvc.BackupWarning](launchsvc.BackupWarningEvent)
	application.RegisterEvent[launchsvc.SteamOption](launchsvc.SteamOptionEvent)
	application.RegisterEvent[launchsvc.SettingsRestoreWarning](launchsvc.SettingsRestoreWarningEvent)
	application.RegisterEvent[launchsvc.RescuedNotice](launchsvc.RescuedEvent)
	application.RegisterEvent[launchsvc.RescuedNotice](launchsvc.DiscardRescuedEvent)
	application.RegisterEvent[updatesvc.Release](updatesvc.StagedEvent)
	application.RegisterEvent[updatesvc.ModUpdateDigestNotice](updatesvc.ModUpdateDigestEvent)
	application.RegisterEvent[profile.GameModsProgress](profile.GameModsProgressEvent)
	application.RegisterEvent[string](quitRequestedEvent)
}

func main() {
	components.SetRunning(appversion.ForCatalog(version))
	applyMemoryLimit(os.Getenv)
	if err := runShowingFailures(); err != nil {
		os.Exit(1)
	}
}

// runShowingFailures is run with a failed start or a panic shown in a dialog, since a windowsgui build has no
// console and would otherwise end without a word.
func runShowingFailures() error {
	defer func() {
		if r := recover(); r != nil {
			showStartupError(i18n.T("Mortar stopped while starting:\n\n{error}\n\nThe details are in crash.log in Mortar's data folder.", "error", fmt.Sprint(r)), logFolder())
			panic(r)
		}
	}()
	err := run()
	if err != nil {
		slog.Error("run", "err", err)
		if _, shown := errors.AsType[shownError](err); !shown {
			showStartupError(startupFailureText(err), logFolder())
		}
	}
	return err
}

// registerDoctorLoaders lets each game's loader, as the catalog in use names it, contribute its checks to the Proton
// install checks.
func registerDoctorLoaders() {
	doctor.LoaderOf = func(gameID string) doctor.OverrideLoader {
		for _, l := range game.Loaders(gameID) {
			if w, ok := l.(doctor.OverrideLoader); ok {
				return w
			}
		}
		return nil
	}
}

// syncScan is how often the sync folder is scanned, and how long a payload may lag its version file before the
// profile shows as waiting on it.
const syncScan = time.Minute

// steamSessionPatience is how long a Steam session sent to a running Mortar waits for its game to start, through
// any pre-Play dialog, before it gives up and lets Steam see the session end.
const steamSessionPatience = 15 * time.Minute

// serverShutdownTimeout bounds how long server mode waits for in-flight requests once Mortar quits.
const serverShutdownTimeout = time.Second

// queueQuitWait bounds how long quit waits for the download workers to stop.
const queueQuitWait = 5 * time.Second

// installCacheTTL is how long a game's discovered installs are reused; a game its launcher installs or removes shows
// up within it, and the setup screen's launcher list always looks again.
const installCacheTTL = 30 * time.Second

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "--release-links" {
		return releaseLinks()
	}
	if len(os.Args) > 1 && (os.Args[1] == lan.AllowFirewallFlag || os.Args[1] == lan.AllowFirewallAny) {
		return lan.AllowFirewall(os.Args[1] == lan.AllowFirewallAny)
	}
	if len(os.Args) > 2 && os.Args[1] == "--handle-links" {
		return claimLinks(os.Args[2])
	}
	// Every client that sets no User-Agent of its own, the CLI and the native host included, names Mortar.
	http.DefaultTransport = &httpkit.Transport{Base: http.DefaultTransport, UserAgent: appversion.UserAgent(version)}
	registerDoctorLoaders()
	if cli.Is(os.Args[1:]) {
		os.Exit(cli.Run(version, os.Args[1:], os.Stdout, os.Stderr))
	}
	if nativehost.Invoked(os.Args[1:]) {
		return serveNativeHost()
	}
	registerEvents()
	if err := prepareServerMode(); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// application.New exits a second instance after forwarding its arguments, so it runs before anything that
	// writes to disk: a second instance must not clean, collect or rewrite the running instance's data.
	c := &boot{home: home}
	c.initInstanceState()
	defer c.closeReady()
	go c.serveHandoffs()
	if done, err := c.preflight(); done || err != nil {
		return err
	}
	updateCtx, stopUpdates := context.WithCancel(context.Background())
	defer stopUpdates()
	queueCtx, stopQueue := context.WithCancel(context.Background())
	c.stopQueue = stopQueue
	if err := runPhases(
		c.newApp,
		c.openLogs,
		c.openStores,
		c.profileHooks,
		c.startLaunches,
		c.nexusServices,
		c.nxmSetup,
		c.pictureWiring,
		c.openQueue,
		c.wireQueue,
		c.problemsWiring,
		c.shareSync,
		c.quitWiring,
		c.dataServices,
		c.browseConfig,
		c.registerServices,
		func() error { return c.startBackground(updateCtx, queueCtx) },
	); err != nil {
		return err
	}
	defer func() {
		stopQueue()
		c.queueSvc.StopWait(c.waitQueue, queueQuitWait)
	}()
	if err := runPhases(
		func() error { return c.serveControl(queueCtx) },
		c.attachHost,
		c.windowSetup,
		c.openWindow,
		c.playMode,
		c.storeMaintenance,
		func() error { return c.catalogRefresh(updateCtx) },
		func() error { return c.windowFuncs(updateCtx) },
		c.setupTray,
	); err != nil {
		return err
	}

	c.closeReady()
	err = c.app.Run()
	stopQueue()
	if errors.Is(err, context.DeadlineExceeded) {
		// Requests abandoned at the end of serverShutdownTimeout; the shutdown hooks have already run.
		return nil
	}
	return err
}

// runPhases runs each startup phase in order and stops at the first error.
func runPhases(phases ...func() error) error {
	for _, p := range phases {
		if err := p(); err != nil {
			return err
		}
	}
	return nil
}

// withLinks runs fn with the settings and the system's link handler, the player's link choices applied.
func withLinks(fn func(*settings.Store, nxm.Handler) error) error {
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
	applyLinkChoices(store.Get())
	return fn(store, h)
}

func releaseLinks() error { return withLinks(nxmsvc.ReleaseLinks) }

// claimLinks is the installer's link checkbox: Mortar takes one source's links.
func claimLinks(id string) error {
	return withLinks(func(store *settings.Store, h nxm.Handler) error { return nxmsvc.ClaimLinks(store, h, id) })
}

// serveNativeHost runs Mortar as the browser extension's native messaging host: each nxm link or Nexus collection
// page link is handed to a new Mortar process, which forwards it to the running one like any launch, or starts Mortar
// when none runs. The browser waits for the reply, so the child is not waited for.
func serveNativeHost() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe = hostenv.Launchable(exe)
	return nativehost.ServeFrom(os.Args[1:], os.Stdin, os.Stdout, func(link string) error {
		if !source.IsLink(link) && !sharesvc.IsCollectionURL(link) {
			return fmt.Errorf("not an nxm or collection link: %q", link)
		}
		return nativehost.Start(exe, link)
	})
}

// applyLinkChoices tells the source registry which link schemes the user chose to claim from the system.
func applyLinkChoices(s settings.Settings) {
	choice := map[string]bool{}
	if s.ThunderstoreHandleLinks != nil {
		choice["thunderstore"] = *s.ThunderstoreHandleLinks
	}
	if s.CurseForgeHandleLinks != nil {
		choice["curseforge"] = *s.CurseForgeHandleLinks
	}
	if s.VSModDBHandleLinks != nil {
		choice["vsmoddb"] = *s.VSModDBHandleLinks
	}
	source.SetHandleLinks(choice)
}
