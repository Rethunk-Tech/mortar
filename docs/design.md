# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here.

## Sharing: the static page deploy

Remaining: deploying `site/stardew/p/` and `site/download/` at `https://mortar.rethunk.tech/stardew/p` and `https://mortar.rethunk.tech/download/`, held back until the first release is ready. How sharing works: [architecture.md](architecture.md#sharing).

- **Deploy:** `site/stardew/p/index.html` and `site/download/index.html` in this repo, as a DigitalOcean App Platform static site (free tier: three static apps); `rethunk.tech` is on DigitalOcean's nameservers, and `maitre.rethunk.tech` is already a CNAME to an App Platform app, so the subdomain is set up the same way. The share page shows two buttons, since it cannot tell whether a scheme handler exists: open in Mortar (`mortar://stardew/p/<payload>`) and Get Mortar (`/download/`), which also copies the link so the importer can take it after installing.
- **Done when** the link opens the page on the live domain, its button opens Mortar's Import with the link filled in, and no request the page makes carries the fragment.

## Release

Remaining ([architecture.md](architecture.md#release)):

- The updater fixes are offered upstream as wailsapp/wails#6200 (EXDEV staging), #6201 (AppImage) and #6202 (OnUpdateApplied, draft pending a WEP); Mortar pins the fork until they ship in a tagged v3 beta; then pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`. wailsapp/wails#6197 (GTK4 transparency) does not gate this: it only serves the translucent window below.
- The repo turns public at the first release and builds go on its GitHub Releases, since the updater's manifest and assets must be publicly downloadable.
- **Measure on Windows:** how launch arguments order around `%command%`, and whether SMAPI needs `--no-terminal`; one real update through the updater, and `DisplayVersion` after it.

## Startup time per mod

Shows how long each mod adds between pressing Play and the title screen. Measured on the main profile (2026-10-03, 193 SMAPI mods, 417 content packs, 76 s run): SMAPI loads mod DLLs from 3 s to 10 s, calls mods' `Entry` from 10 s to 22 s, then 30 to 45 s, 46 to 60 s and 62 to 70 s pass with no log line at all. SMAPI 4 has no per-mod timing (SMAPI 3's performance monitor is gone), and its log has one-second timestamps, so the bridge measures inside the game. Nothing is passed through environment variables: Mortar writes the bridge's `config.json`.

- **Config** (`mortar-smapi-bridge` `ModConfig.cs`, written by `internal/overlay/overlay.go` `WriteBridgeConfig`): `StartupTimings` (default on) turns on the counters below; `StartupProfile` (default off) adds the sampler for one launch. Performance › Startup › **Measure next launch** sets `StartupProfile`, and Mortar clears it when that run's report arrives.
- **Load early:** Mortar writes `<profile>/mods/SMAPI-config.json` with the bridge's UniqueID in `ModsToLoadEarly` (SMAPI 4.5.2 `Constants.ApiModGroupConfigPath`, merged into SMAPI's settings; keep any other keys a user put there). The bridge's `Entry` then runs before every other mod's.
- **Entry:** in its `Entry`, the bridge Harmony-patches each other loaded mod's `Entry(IModHelper)` override with a prefix and postfix that time it (`Stopwatch`, per mod).
- **Events:** SMAPI's `ManagedEvent<T>.Raise` calls each `ManagedEventHandler<T>.Handler` with its `SourceMod` (SMAPI `Framework/Events/ManagedEvent.cs:98`). The bridge walks SMAPI's `EventManager` with plain `System.Reflection` (SMAPI's reflection helper refuses its own types; plain reflection works, as `ResolveRawCommandQueue` already does) and replaces each handler's delegate with a timed wrapper built per event-args type. It does this in a `GameLaunched` handler at `EventPriority.High + 1` (so later handlers in the same raise are already wrapped) and again on each `UpdateTicking` until the title screen, for late registrations. At the title screen it puts the original delegates back, so play has no overhead.
- **Assets:** a Harmony postfix on the constructors of SMAPI's `AssetEditOperation(Mod, Priority, OnBehalfOf, ApplyEdit)` and `AssetLoadOperation(Mod, OnBehalfOf, Priority, GetData)` records (non-generic) wraps `ApplyEdit` / `GetData` with a timer keyed by `Mod` and `OnBehalfOf`, which gives a time per content pack (Content Patcher edits carry the pack as `OnBehalfOf`). Unwrapped at the title screen like events.
- **Sampler** (`StartupProfile` only): the bridge starts an EventPipe session on its own process from `Entry` with `Microsoft.Diagnostics.NETCore.Client` (sample profiler plus method rundown), stops it at the title screen, and parses it on a background thread with `Microsoft.Diagnostics.Tracing.TraceEvent` (both MIT; add to the bridge's credits and Mortar's licence notices). Each sample is charged to the innermost frame from a mod assembly (frames from SMAPI, the game, MonoGame, Harmony and MonoMod are skipped), so time inside a mod's Harmony patch on game code counts for that mod; samples with no mod frame are "game and SMAPI".
- **Title screen:** the first `RenderedActiveMenu` with `Game1.activeClickableMenu is TitleMenu` after its intro, timed from the process start (`Process.GetCurrentProcess().StartTime`).
- **Report:** the bridge writes `<profile dir>/startup/<run start UTC>.json` atomically (profile dir found as GMCM capture finds it: the nearest folder above the mod holding `profile.json`) as `{schema: 1, smapi, game, processStart, phases: {modsLoaded, entryDone, gameLaunched, titleScreen} (ms from process start), sampled, mods: [{id, name, entryMs, eventMs: {<event>: ms}, assetMs, packs: [{id, name, ms}], sampleMs}], otherMs}`; it keeps the last 10. Mortar attaches the report to the run whose start it matches and adds the pre-bridge phases from the SMAPI log (SMAPI start, `Loading mods...`, `Launching mods...`).
- **Performance › Startup:** a phase bar (DLL loading, Entry, launch, content, title) for the latest run, then a virtualised table of mods sorted by total time (Entry, events, assets and, on a sampled run, sampled time) with each content pack's time under its framework row, and a run picker comparing the last runs. Mods under 50 ms fold into one "Other mods" row.
- **Traps:** tiered compilation is off in Stardew's runtime config, so the sampler sees fully jitted frames; wrapping must keep `ManagedEventHandler.Handler`'s delegate type exactly (`EventHandler<T>`) or the raise throws; a mod whose `Entry` throws is still timed (postfix with `__exception` or a finalizer); the bridge must never let a timing failure reach the game (catch, log once at trace, turn timing off for that run).
- **Done when** a launch of the main profile writes a report whose phases add up to the measured time to the title screen within 1 s, the top rows name the mods behind the silent stretches above, and a sampled run attributes at least 90% of the time between `Entry` and the title screen to named mods or "game and SMAPI".

## Queued for v1

- **Library:** an extra folder to scan for mods; a toggle to show dot-hidden mods; asking before deleting old files on update; new folders in the game's own `Mods` folder offered for moving into a profile.
- **Packages:** an aarch64 Flatpak bundle is blocked without qemu binfmt (or an aarch64 host): `flatpak-builder --arch=aarch64` still runs `build-commands` via the aarch64 SDK's `/bin/sh` (`bwrap: execvp /bin/sh: Exec format error`), even with only `install` of a prebuilt binary.
- **Self-test:** update a mod in place with an older GitHub release in a copied game folder, checking carry-over, `.mortar-old`, the save backup and Roll back.
- **CurseForge** as a third source, after the repository is public: apply for a 3rd-party API key, then build it without caching API data, with a User-Agent on every request, and honouring each author's distribution setting.

## Later

- **UI translations** beyond English: the Lingui machinery and extracted catalogs exist; needs chosen languages and translators. Parked 2026-10-02 (not v1).
- **Steam Deck / gamepad mode** (larger targets, gamepad focus navigation, Game Mode): parked 2026-10-02 (not v1).
- **macOS build**: Stardew runs on macOS, but Mortar has no macOS CI or test machine. Parked 2026-10-02 (not v1).
- **Scheduled save backups** (daily or every N hours while Mortar runs, keep the last N per save, never while the game writes), beside today's before-Play backups: parked 2026-10-03.
- **Accessibility pass** (keyboard-only navigation of every screen, focus order, screen-reader labels on icon buttons, reduced motion everywhere): parked 2026-10-03.
- **Offline mode banner** (clear banner when Nexus/GitHub are unreachable, cached data with "as of" times, network actions disabled with a reason): parked 2026-10-03.

Not in the first release; re-weigh only when asked:

- Game Select hover in the style of Concrete's switcher, where the hovered game grows while the others become strips, with parallax and brightness shifts: eased `flex-grow` and `flex-basis` looked jumpy in WebKitGTK, so v1 has no hover effect; revisit with a measured, transform-based approach.
- Lethal Company, as the second `Game` implementation ([lethal-company.md](lethal-company.md)).
- Translucent window, desktop showing through; on Windows it needs Acrylic measured with the frameless window. The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background except the title bar. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux, built when the desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- In-app mod search and browsing: Mortar links out to Nexus.
- ModDrop as a source (no documented download API); the Xbox app version (WindowsApps folders are locked down).
- Windows code signing.
- macOS, as Stardrop ships for x64 and arm64: needs an Apple developer account for signing and notarization, Mac Steam paths and nxm registration, and a Mac to test on.
- More interface languages than English, as Stardrop (17+), MO2 and r2modman ship; every string already goes through Lingui (English only for v1).
- Portable mode: the data folder beside the executable, switched by a marker file (Move data folder exists).
- Previewing an archive's file tree before installing it (the folder picker shows it only when no manifest is found).
- Bottles as a launcher (Linux): games there are Windows builds in a Wine prefix, so it needs SMAPI's Windows installer run inside the bottle (`bottles-cli run -b <bottle> -e <installer>`) and launches through `bottles-cli run` with `--mods-path`; the Linux SMAPI install would break such a copy. Detection is simple: bottles under `~/.local/share/bottles/bottles` and `~/.var/app/com.usebottles.bottles/data/bottles/bottles`, each searched for `drive_c/Program Files (x86)/Steam` and GOG folders.
- Steam Deck Game Mode play: a `--play` launch from a Steam shortcut starts Mortar minimised or headless, shows only a small controller-friendly prompt when Play is blocked, and exits when the game closes so Steam tracks playtime.
- A profile sync folder (Syncthing, Dropbox, a NAS) holding each profile's `.mortar` state, so another machine is offered the changes, with conflict detection when both sides edited; mod files still come from their sources.
- Needs Nexus's approval through app registration first (below), since it starts downloads outside Nexus's own Mod Manager Download button: an "Add to Mortar" button on Nexus listing tiles.
- Profile templates: a new profile started from a bundle plus game settings and launch options.
- Per-profile save isolation.
- Settings considered and not taken (2026-10-02): new profiles starting as a copy of the open profile or from a bundle; scheduled save backups on a timer while the game runs; an offline mode that never contacts the network.
- Registering Mortar with Nexus (SSO slug; ask then about OAuth, which Vortex uses via `nxm://oauth/callback`, and Collections).
