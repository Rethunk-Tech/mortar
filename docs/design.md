# Mortar: decided, not yet built

Decided work, short enough to build cold. Delete an item in the change that lands it. Screens and styling: [gui-design.md](gui-design.md). Standing rules: [AGENTS.md](../AGENTS.md).

## Release

- Drop the `Rethunk-AI/wails` replace in `go.mod` (branch `mortar/v3-next`) and pin an upstream release once one carries what the branch adds: #6202 (OnUpdateApplied, with WEP #6203), #6199 (service methods fall back to `Options.MarshalError`; `errorkind_test.go` relies on it), #6240 (single-instance lock released at the start of shutdown), #6197 (GTK4 transparency), #6281 and #6282 (session-bus fallbacks), #6283 (GTK4 drop motion), #6285 (server-mode shutdown hooks), #6284 and #6287, WEPs #6243 (local linuxdeploy) and #6244 (`-nopack`), and the fork-only apply-on-exit.

## Ready to build

### From Later, picked 2026-11-01

Each part goes in the repo that owns its concern; Mortar keeps the product side (wording, settings, findings, UI). A library change ships and is tagged before Mortar bumps to it.

- **Develop: GitHub release from the tag.** The release API (create, upload asset) in go-github; Mortar's devsvc builds the notes from the project's CHANGELOG and attaches the packed zip.
- **Develop: dependencies in step both ways.** A project's `ThunderstoreDependency` items and its development profile's mods; manifest dependency parsing stays in go-thunderstore, the sync rules (which side wins, what is offered) in Mortar's devsvc.
- **Develop: development profile over LAN pairing** for co-op tests, through Mortar's existing pairing (`internal/lan`); Mortar only.
- **Upload a modpack to Thunderstore**, as Gale does: the submission API's multipart upload moves out of Mortar's devsvc into go-thunderstore, shared by Develop's Release and the modpack upload; the team token in the keyring (go-keyring) and a preview step in Mortar, reusing ModpackDialog's export zip.
- **Vintage Story dedicated server profiles** (`dotnet VintagestoryServer.dll --dataPath <folder>`), sharing the client profile's Mods and ModConfig; launch through go-gameproc, the profile kind in Mortar.
- **Core Keeper on BepInEx 6 IL2CPP:** the BE6 IL2CPP layout and version reading in go-bepinex; a BepInEx 6 IL2CPP build of mortar-bepinex-bridge (`BasePlugin`, .NET 6, Il2CppInterop) as its own package; the catalog entry (Thunderstore `core-keeper`, pack `BepInEx-BepInExPack_Core_Keeper`). The maintainer owns Core Keeper.
- **GodotModLoader games** (Dome Keeper first; the maintainer owns it): a zipped mod as a new placed-file kind through Mortar's installer layout, records, drift and history; launch arguments in the catalog (`--script res://addons/mod_loader/mod_loader_setup.gd --enable-mods --mods-path <profile>/mods`, r2modman's GodotMLInstaller).
- **Folder games:** Baldur's Gate 3 (an LSPK `.pak` reader and `modsettings.lsx` writer in a new go-larian), Bannerlord (`SubModule.xml` and load order in a new go-bannerlord), Planet Zoo and Graveyard Keeper (catalog entries; Graveyard Keeper gets a BepInEx loader pack). The maintainer owns Planet Zoo and Graveyard Keeper.
- **More stores for catalog games** from known paths (r2modman, Gale, PCGamingWiki, cited by file): GOG for The Witcher 3 (`1207664663`) and Cyberpunk 2077 (`1423049311`); Darkest Dungeon on GOG and Epic and its native Linux build; Hogwarts Legacy on Epic; Starbound's Xbox package; Titanfall 2 on the EA app. Store lookups in go-gamelocator, entries in the catalog.
- **A game version per profile** (Gale #157): a per-profile game folder override; Get an older version opens Steam's console (`steam://open/console`) with `download_depot <app> <depot> <manifest>` copied for the player to paste, then adopts `steamapps/content/app_<app>/depot_<depot>`; no Steam credentials. A per-profile "mod versions up to" date that browse, install and update honour. Depot content paths in go-gamelocator.
- **macOS, the Linux-testable parts:** darwin Steam roots in go-gamelocator (`~/Library/Application Support/Steam`, `GOOS=darwin go vet` in its crosscompile gate), a darwin `Clone` with `clonefile` in go-safefs, `.app` markers (`darwinMarker`) and the BepInEx `run_bepinex` branch in go-bepinex, unit-tested on Linux with a `Platform: "darwin"` view.

### The website on Astro

Stage 2: unique content per game page (`web/src/pages/[game]/index.astro`), beside the loader, sites and stores it shows:

- top mods from the sources' APIs, cached in a data file under `web/src/data/` that `bun run site:refresh` writes (`web/scripts/refresh.ts`);
- an optional hand-written note per game.

### mod.io as a source

**Why:** mod.io is an official mod channel for Blade & Sorcery, BONELAB, Ready or Not, S.T.A.L.K.E.R. 2 and Timberborn, and the only one for Gladio Mori. It is also how Baldur's Gate 3 and Deep Rock Galactic ship mods, and it would bring back Core Keeper (no live source). Neither r2modman nor Gale supports it.

**Access:** every target game has `OPEN` in `api_access_options`: Blade & Sorcery (mod.io game 3852), BONELAB (3809), Ready or Not (3791), S.T.A.L.K.E.R. 2 (5761), Timberborn (3659), Gladio Mori (11453), Baldur's Gate 3 (6715), Deep Rock Galactic (2475) and Core Keeper (5289). All but Deep Rock Galactic also set `DOWNLOADS` (its download URLs carry a changing verification hash). None needs a signed-in user to download. Before shipping, tell mod.io what Mortar does: API Access Terms 3.5 forbids "a competing or replacement product".

**Decided shape:**

- An `internal/source/modio` driver shaped like `internal/source/modrinth`: `Search`, `Sorts` (downloads, updated, newest), `Details`, files, dependencies.
- Sign-in per user, like itch.io (`internal/source/itch/service.go`): the player pastes a read-only OAuth token from mod.io/me/access, Mortar tests it and keeps it in the keyring (`internal/secret`), and Settings › Accounts shows the account. A key shipped in the binary fails: a key is capped at 60 requests per minute across every install sharing it, a user's own token at 120. Tokens have no refresh, so an expired one asks the player to sign in again.
- Downloads: fetch the modfile again right before each download (`binary_url` can expire), check its MD5 (the only hash mod.io gives), send `X-Modio-Platform: windows` for Proton games and `linux` for native ones, and honour `retry-after` on a 429.
- Dependencies come from `?recursive=true`; some creators list soft "credit" links, so a dependency is offered, never installed unasked.
- Placement: the game's manual mod path. The mod.io SDK's cache is synced by the game's own client, which may overwrite it.
- Branding: mod.io's mark and name wherever its mods appear (terms 3.3).
- Catalog: add `modio` to the five games above and Gladio Mori. Baldur's Gate 3, Deep Rock Galactic and Core Keeper are separate entries with their own install layouts.

**Traps:**

- A game whose mod.io download option is authorised-only needs the token for every download, and a paid-only one also needs ownership.
- Requests go to the per-key host `u-{user-id}.modapi.io`.
- Shadows of Doubt moved to Steam Workshop: re-check it.

**Done when:** with a test token, Browse lists Timberborn's mod.io mods with all sources sorted by downloads. Installing one checks its MD5, places it in the profile's mod path, and offers its dependencies. A revoked token shows the sign-in prompt.

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
- **Other loaders and game families:** Bethesda games (plugin load order, LOOT, script extenders, BSAs); Minecraft (Java instances, Fabric, Forge, NeoForge); Resonite on BepisLoader.
- **Runs that need an outside tester:** a launch from a real GOG install and from Flatpak Steam; a Linux launch with Stardew's launch method set to Steam, confirming SMAPI loads the profile's mods; Import from Vortex against a long-lived Vortex folder; timings with the data folder under OneDrive.
- **Translate a mod's description** into the player's language (as StarModsManager does for Stardew), optional and per description; revisit together with the app's own translations.
- **Hosted share service** with short codes and share versioning.
- **Add to Mortar** button on Nexus listing tiles, which needs Nexus's approval first ([nexus-application.md](nexus-application.md)).
- **Windows code signing** with Azure Trusted Signing.
- **GitHub topic-search fallback** (`internal/source/github/fallback.go`) into go-github, once a second consumer needs it.
- **Registering Mortar with Nexus** ([nexus-application.md](nexus-application.md)): once approved, release builds pass `nexussvc.Slug` and `ClientID` as ldflags and the same change removes the API-key sign-in (go-nexusmods' `WithKey`, `internal/nexussvc/apikey.go`), the only sign-in while neither is set.
- **macOS support** (darwin/arm64 and amd64), parked until the maintainer owns or borrows a Mac. The Linux-testable parts are under Ready to build.
  - **Scope:** the catalog's `darwin` path templates and the macOS asset picker exist. Profile copies need a darwin `Clone` using `clonefile` in go-safefs (`clone_stub.go` there). Lethal Company and most BepInEx games ship no macOS build, so the first targets are native-macOS games (check each game's macOS depot before enabling) and, for Windows-only games, a CrossOver or Whisky prefix.
  - **Wails v3:** `wails3 build GOOS=darwin` on the pinned fork ([stack](architecture.md#stack)), on a Mac only (cgo and the macOS SDK). Add `build/darwin/` (Taskfile, `Info.plist`, `icons.icns`, as `build/windows/` and `build/linux/` do) and `darwin: ./build/darwin/Taskfile.yml` under `includes` in `Taskfile.yml`. Check the fork's WKWebView paths first: frameless title bar, `BackgroundTypeSolid`, drag regions, the tray (`tray_other.go`), `nxm://` registration (`Info.plist` `CFBundleURLTypes`, replacing `internal/nxm` runtime registration), single-instance.
  - **Game discovery:** go-gamelocator's `steam/locate_unix.go` (`//go:build !windows`) lists only Linux roots. Add `~/Library/Application Support/Steam` (r2modman's darwin game-directory resolver) and `GOOS=darwin go vet ./...` in the module's crosscompile gate, and read `steamapps/libraryfolders.vdf` as the Linux path does. The install folder holds `<Game>.app` (marker `Contents/MacOS/<exe>` or `Contents/Resources/Data`), so the catalog's `marker` needs a `darwinMarker` like `LinuxMarker` (`internal/game/install.go`, `internal/gamestore/standalone.go`). Gale has no macOS code; r2modman is the only precedent: it launches Steam with `steam_osx -applaunch <id>` (its darwin Steam runner) and a `macos_proxy` launch wrapper (its launch utilities).
  - **Loaders:** BepInEx on macOS starts through the pack's `run_bepinex` script (Doorstop's `libdoorstop.dylib`, injected by the dynamic loader), set as the Steam launch option `"<profile>/run_bepinex" %command%` (r2modman's BepInEx game instructions). `internal/loader/bepinex5` places the Windows `winhttp.dll` pair and needs a darwin branch beside `contributeLinux` in `driver.go` (and thunderpack's). Steam strips dyld injection for hardened binaries, so test a hardened game early. SMAPI's installer swaps the launcher on macOS as on Linux ([games](games.md#smapi)), but `internal/loader/smapi/install.go` hardcodes the Linux names and the bundle layout (`Stardew Valley.app/Contents/MacOS/StardewValley`) differs. MelonLoader ships no macOS build: leave it off, as the driver already refuses Linux.
  - **Packaging:** a `.app` in a `.dmg`, universal or two arch builds; codesign (hardened runtime) and notarise (`notarytool`) with an Apple Developer ID certificate (paid account, a maintainer decision); self-update through `wails3 updater manifest` with a `darwin` artifact in `Taskfile.yml`; a Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap. Gatekeeper blocks an unnotarised app, so do not ship unsigned.
  - **Testing:** no macOS runners in workflows (billable minutes), so CI stays Linux and Windows; manual testing needs a lender with a Mac, or a one-off macOS VM or hosted runner the maintainer approves. The pure-Go parts (Steam roots, markers, loader branches) are unit-tested on Linux with a `Platform: "darwin"` view, as `internal/loader/thunderpack/thunderpack_test.go` does for Windows.
  - **Done when:** `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://` links; a Stardew Valley profile installs SMAPI, launches through Steam and loads a mod (`SMAPI-latest.txt`); the Steam root and `.app` game folders are found without a manual folder choice; the Homebrew cask installs and updates it; `bun run gate` passes on Linux with the new darwin unit tests.
