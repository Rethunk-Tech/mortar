# Mortar: decided, not yet built

Decided work, short enough to build cold. Delete an item in the change that lands it. Screens and styling: [gui-design.md](gui-design.md). Standing rules: [AGENTS.md](../AGENTS.md).

## Release

- Drop the `Rethunk-AI/wails` replace in `go.mod` (branch `mortar/v3-next`) and pin an upstream release once one carries what the branch adds: #6202 (OnUpdateApplied, with WEP #6203), #6199 (service methods fall back to `Options.MarshalError`; `errorkind_test.go` relies on it), #6240 (single-instance lock released at the start of shutdown), #6197 (GTK4 transparency), #6281 and #6282 (session-bus fallbacks), #6283 (GTK4 drop motion), #6285 (server-mode shutdown hooks), #6284 and #6287, WEPs #6243 (local linuxdeploy) and #6244 (`-nopack`), and the fork-only apply-on-exit.

## Ready to build

### Update version binding

A freshness gap in what the signing pipeline signs ([security](security.md)). It needs the signing step, so it is not a client-only change.

- **Update version binding.** Wails' updater verifies an Ed25519 signature over the artifact digest only (`Rethunk-AI/wails/v3/pkg/updater/verify.go:137`, `:175`); `Verification` carries no version or platform. Whoever controls `releases/latest/download/manifest.json` can serve an older signed build, or another platform's, under a higher version number. `TestUpdaterE2EDowngradeIgnored` covers only a lower version number.
  - **Decided shape:** the release step signs `version|platform|digest` instead of the digest (`wails3 task release:manifest`, `docs/release.md`), the fork's verifier rebuilds that message from the release it checked, and `updatesvc.Check` (`internal/updatesvc/service.go:187`) keeps refusing a release without a signature. Mortar and the fork change together, and older builds must still read the new manifest, so the manifest carries both signatures for one release cycle.
  - **Done when:** an e2e test installs nothing when a validly signed older artifact is relabelled with a higher version.

### From Later, picked 2026-11-01

Each part goes in the repo that owns its concern; Mortar keeps the product side (wording, settings, findings, UI). A library change ships and is tagged before Mortar bumps to it.

- **Folder games:** Planet Zoo and Graveyard Keeper (catalog entries; Graveyard Keeper gets a BepInEx loader pack; each waits for a cited mods folder or marker, which no r2modman, Gale or Vortex source gives). The maintainer owns Planet Zoo and Graveyard Keeper.
- **More stores, the ones without a citable id** (built: GOG for The Witcher 3 and Cyberpunk 2077, the EA app for Titanfall 2): Darkest Dungeon on GOG (`1450711444`) needs a second marker (`_windowsnosteam/win64/Darkest.exe`), which is a new catalog field and so needs `minVersion` and the oldest-tag decode check; Darkest Dungeon's native Linux build needs its executable's name; Hogwarts Legacy and Darkest Dungeon on Epic need their Epic AppName (PCGamingWiki gives only the store slug); Starbound on Xbox needs its package Name (only the Store id `9mtsbwpwpqvq` is known). Each waits for a source.
- **macOS, what is left after the Linux-testable parts** (built: darwin Steam roots, `clonefile`, `darwinMarker` support, BepInEx launch-script placement with the Steam launch option, SMAPI's app-bundle paths and mod deployment, a darwin vet in the gate): no catalog entry has a `darwinMarker` yet (add each from a cited source); and SMAPI's macOS installer folder name, `executable_name` without `.app`, and the launch-option form are unverified until a Mac run.

### Companion split (after the 0.6.0 tag)

The BepInEx companion keeps its channel and query, perf (needs Harmony's patch registry), the startup patcher, intro skip, console, configsync and log-source tags (always on). The stream overlays leave the bridges. Lethal Company's DevLaunch intro flags stay beside the companion's.

- **Companion (mortar-bepinex-bridge):** one runtime-detected DLL for every Mono game. Game rows stay in `Games/<Game>/` files, picked by `Application.productName`; no `#if`, no per-game zips once retired. `status` gains a `features` list (missing means the 0.5.x set). Before it: fix the stale TODO.md and README.md paths and HUMANS.md's Nexus ids; drop IL2CPP's LogTag link; move `IntroSkip.Find` and `Overlay.Probe` to a neutral helper; log the "alive after scene" line from `Plugin`, since bisect reads it (`internal/launchsvc/bisect.go:35`).
- **Overlays:** `mortar-mods/bepinex5/lethal-company/mortar-lc-streamoverlay` (Lethal Company only, MonoMod, Thunderstore) and `mortar-mods/smapi/stardew-valley/mortar-sdv-streamoverlay` (out of `mortar-smapi-bridge`). Both read `startup/overlay.json` and serve `/state` byte-identical to today's. Valheim's overlay is dropped.
- **Mortar:** the catalog keeps `"bridge"` pinned at 0.5.x for old Mortars (they install only that name, `internal/loadersvc/service.go:154`); new component names point at the split builds. A profile holds a second hidden Mortar bundle: `applyBundledTo` replaces only the same component (`internal/profile/ops.go:1043`), `bridgeFolder` matches the bridge entry (`internal/launchsvc/service.go:1857`). The overlay mod is added while the overlay setting is on and removed when it is off; the overlay settings show only for games that have an overlay component. A profile upgraded with the setting on keeps its overlay in the same sync. The 0.5.x companion and an overlay mod never load together (both bind the overlay port).
- **Acceptance:** `/state` matches 0.5.0 for LC and Stardew fixtures; the companion DLL has no `Overlay` namespace; one build and test run in the companion's `verify.sh`; old-tag `components.Decode` reads the new catalog; bisect, perf stop, console and configsync pass the sandbox regress.

### The Sims 4 placement speed

- Decide how placement copes with large profiles on Windows: every Play places every file again (`internal/deploy/place.go`), several times slower than on Linux (`MORTAR_SCALE=1 go test ./internal/loader/folder -run TestPlaceAndPurgeAThreeThousandFileProfile -v`). Options: show progress; leave files unchanged since the last Play in place; link files where the volume allows (needs an answer for a mod that rewrites its own file).

### Lethal Company content packs under `mortar/<kind>/`

Mortar's side is built (`internal/devsvc/content.go`, the catalog's `devTemplate.content`). Left:

- Mortar-Posters, Mortar-Boombox and Mortar-Television are published to Thunderstore, and the known-broken redirects from LethalPosters, LethalPaintings, Custom Boombox Music and TVLoader to them follow. New mod offers a content tick only once its loader is there.
- The per-kind `assetKinds` entries for posters, paintings, television, boombox, suits and sounds leave the catalog once no supported build predates the `content` entry's `minVersion`.

Done when: a pack with posters + boombox + tv in `plugins/mortar/`, suits in `plugins/moresuits/` and a sound pack loads all three in game; a legacy-layout pack still loads; a code project's ticked content ships in its package.

### Lethal Company facts to verify

Record each in [games.md](games.md#lethal-company) once seen.

- Profile paths with spaces or non-ASCII characters (BepInEx flags `Assembly.LoadFile` issues on some Mono builds, `Entrypoint.cs:94,142`).

## Later

Parked; take up when asked.

- **Scrap items without Unity** (Lethal Company): a bundle cannot be built outside the Unity Editor (AssetsTools.NET and UnityPy only patch existing bundles; a new MonoBehaviour's script hash cannot be generated). The path is a runtime loader mod in mortar-mods on LethalLib: `NetworkPrefabs.CloneNetworkPrefab` (deterministic hash from assembly and name) of template prefabs built once in Unity, an `Item` from `ScriptableObject.CreateInstance` passed to `Items.RegisterScrap`, filled from a folder (`item.json`, `texture.png`, `icon.png`, sounds). Mortar's wizard gains a Scrap item kind that writes the folder. Sholiver-LethalCardLoader proves the PNG+JSON half. Arbitrary GLB meshes (glTFast with an HDRP material mapper copied from a vanilla scrap) come after and risk the game's Burst/Collections versions and grip offsets.

- **Translucent window** through the fork's GTK4 `setTransparent()` fix (upstream #6197). Traps: a fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque, and stacked alphas compound toward opaque, so images and overlays each need their own alpha. Windows needs Acrylic with the frameless window. The AppImage is the GTK 3 build (RGBA visual) and needs the same check.
- **Frosted glass on Linux** (needs GNOME 51): in the fork beside `setTransparent()`, bind Wayland `ext_background_effect_manager_v1` and set the toplevel surface's blur region to the whole window, updated on resize, only while the translucent window is on. The AppImage's GTK 3 build reaches the surface through `gdk_wayland_window_get_wl_surface`.
- **Other loaders and game families:** Bethesda games (plugin load order, LOOT, script extenders, BSAs); Minecraft (Java instances, Fabric, Forge, NeoForge).
- **Runs that need an outside tester:** a launch from a real GOG install and from Flatpak Steam; a Linux launch with Stardew's launch method set to Steam, confirming SMAPI loads the profile's mods; Import from Vortex against a long-lived Vortex folder; timings with the data folder under OneDrive.
- **Translate a mod's description** into the player's language (as StarModsManager does for Stardew), optional and per description; revisit together with the app's own translations.
- **Hosted share service** with short codes and share versioning.
- **Structured logging:** move the `log.Print*`/`log.Fatal*` calls (heaviest in `main.go` and `internal/nxmsvc/service.go`) to `slog` with attributes. `slog.SetDefault` in `main.go` already routes `log.*` through the handler as plain messages, so the change is how lines read in run logs and support bundles; plain calls convert by codemod, formatted ones by hand.
- **Add to Mortar** button on Nexus listing tiles, which needs Nexus's approval first ([nexus-application.md](nexus-application.md)).
- **Windows code signing** with Azure Trusted Signing.
- **GitHub topic-search fallback** (`internal/source/github/fallback.go`) into go-github, once a second consumer needs it.
- **macOS support** (darwin/arm64 and amd64), parked until the maintainer owns or borrows a Mac. The Linux-testable parts are under Ready to build.
  - **Scope:** the catalog's `darwin` path templates and the macOS asset picker exist. go-safefs already has a darwin `Clone` (`clonefile`), so profile copies need no new code. Lethal Company and most BepInEx games ship no macOS build, so the first targets are native-macOS games (check each game's macOS depot before enabling) and, for Windows-only games, a CrossOver or Whisky prefix.
  - **Wails v3:** `wails3 build GOOS=darwin` on the pinned fork ([stack](architecture.md#stack)), on a Mac only (cgo and the macOS SDK). Add `build/darwin/` (Taskfile, `Info.plist`, `icons.icns`, as `build/windows/` and `build/linux/` do) and `darwin: ./build/darwin/Taskfile.yml` under `includes` in `Taskfile.yml`. Check the fork's WKWebView paths first: frameless title bar, `BackgroundTypeSolid`, drag regions, the tray (`tray_other.go`), `nxm://` registration (`Info.plist` `CFBundleURLTypes`, replacing `internal/nxm` runtime registration), single-instance.
  - **Game discovery:** go-gamelocator's Unix Steam locator (`//go:build !windows`) lists only Linux roots. Add `~/Library/Application Support/Steam` (r2modman's darwin game-directory resolver) and `GOOS=darwin go vet ./...` in the module's crosscompile gate, and read `steamapps/libraryfolders.vdf` as the Linux path does. The install folder holds `<Game>.app` (marker `Contents/MacOS/<exe>` or `Contents/Resources/Data`), so the catalog's `marker` needs a `darwinMarker` like `LinuxMarker` (`internal/game/install.go`, `internal/gamestore/standalone.go`). Gale has no macOS code; r2modman is the only precedent: it launches Steam with `steam_osx -applaunch <id>` (its darwin Steam runner) and a `macos_proxy` launch wrapper (its launch utilities).
  - **Loaders:** BepInEx on macOS starts through the pack's `run_bepinex` script (Doorstop's `libdoorstop.dylib`, injected by the dynamic loader), set as the Steam launch option `"<profile>/run_bepinex" %command%` (r2modman's BepInEx game instructions). `internal/loader/bepinex5` places the Windows `winhttp.dll` pair and needs a darwin branch beside `contributeLinux` in `driver.go` (and thunderpack's). Steam strips dyld injection for hardened binaries, so test a hardened game early. SMAPI's installer swaps the launcher on macOS as on Linux ([games](games.md#smapi)), but `internal/loader/smapi/install.go` hardcodes the Linux names and the bundle layout (`Stardew Valley.app/Contents/MacOS/StardewValley`) differs. MelonLoader ships no macOS build: leave it off, as the driver already refuses Linux.
  - **Packaging:** a `.app` in a `.dmg`, universal or two arch builds; codesign (hardened runtime) and notarise (`notarytool`) with an Apple Developer ID certificate (paid account, a maintainer decision); self-update through `wails3 updater manifest` with a `darwin` artifact in `Taskfile.yml`; a Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap. Gatekeeper blocks an unnotarised app, so do not ship unsigned.
  - **Testing:** no macOS runners in workflows (billable minutes), so CI stays Linux and Windows; manual testing needs a lender with a Mac, or a one-off macOS VM or hosted runner the maintainer approves. The pure-Go parts (Steam roots, markers, loader branches) are unit-tested on Linux with a `Platform: "darwin"` view, as `internal/loader/thunderpack/thunderpack_test.go` does for Windows.
  - **Done when:** `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://` links; a Stardew Valley profile installs SMAPI, launches through Steam and loads a mod (`SMAPI-latest.txt`); the Steam root and `.app` game folders are found without a manual folder choice; the Homebrew cask installs and updates it; `bun run gate` passes on Linux with the new darwin unit tests.
