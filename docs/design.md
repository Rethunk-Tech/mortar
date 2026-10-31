# Mortar: decided, not yet built

Decided work, short enough to build cold. Delete an item in the change that lands it; the code and its commits are the record. Screens and styling: [gui-design.md](gui-design.md). Standing rules: [AGENTS.md](../AGENTS.md).

## Release

- Drop the `Rethunk-AI/wails` replace in `go.mod` (now `v3.0.0-mortar.1`, branch `mortar/v3-next` rebuilt on v3.0.0) and pin an upstream release once one carries what the branch adds: #6202 (OnUpdateApplied, with WEP #6203), #6199 (service methods fall back to `Options.MarshalError`; `errorkind_test.go` relies on it), #6240 (single-instance lock released at the start of shutdown), #6197 (GTK4 transparency), #6281 and #6282 (session-bus fallbacks), #6283 (GTK4 drop motion), #6285 (server-mode shutdown hooks), #6284 and #6287 (merged on upstream master after v3.0.0), WEPs #6243 (local linuxdeploy) and #6244 (`-nopack`), and the fork-only apply-on-exit.

## Ready to build

### The website on Astro

**Stage 2:** unique content per game page, beside the loader, sites and stores it shows (`web/src/pages/[game]/index.astro`):

- top mods from the sources' APIs, cached in a data file under `web/src/data/` that `bun run site:refresh` writes (`web/scripts/refresh.ts`);
- an optional hand-written note per game.

### mod.io as a source

**Why:** mod.io is an official mod channel for Blade & Sorcery, BONELAB, Ready or Not, S.T.A.L.K.E.R. 2 and Timberborn, and the only one for Gladio Mori. Outside the catalog it is how Baldur's Gate 3 and Deep Rock Galactic ship mods, and it would bring back Core Keeper (no live source today). Neither r2modman (declined) nor Gale supports it.

**Access:** every target game allows third-party API access (`api_access_options` has `OPEN`): Blade & Sorcery (mod.io game 3852), BONELAB (3809), Ready or Not (3791), S.T.A.L.K.E.R. 2 (5761), Timberborn (3659), Gladio Mori (11453), Baldur's Gate 3 (6715), Deep Rock Galactic (2475) and Core Keeper (5289). All but Deep Rock Galactic also set `DOWNLOADS`; its download URLs carry a changing verification hash. None needs a signed-in user to download. Before shipping, tell mod.io what Mortar does: API Access Terms 3.5 forbids "a competing or replacement product".

**Decided shape:**

- A `internal/source/modio` driver shaped like `internal/source/modrinth`: `Search`, `Sorts` (downloads, updated, newest), `Details`, files and dependencies.
- **Sign-in per user**, like itch.io (`internal/source/itch/service.go`). The player pastes a read-only OAuth token from mod.io/me/access, Mortar tests it and keeps it in the keyring (`internal/secret`), and Settings › Accounts shows the account. A key shipped in the binary does not work: a user key is capped at 60 requests per minute across every install sharing it, a user's own token at 120 (mint, a Deep Rock Galactic manager, does the same). Tokens have no refresh, so an expired token asks the player to sign in again.
- **Downloads:** fetch the modfile again right before each download, since `binary_url` can expire. Check the file's MD5 (the only hash mod.io gives), and send `X-Modio-Platform: windows` for Proton games and `linux` for native ones. Honour `retry-after` on a 429.
- **Dependencies** come from `?recursive=true`, but some creators list soft "credit" links, so a dependency is offered, never installed unasked.
- **Placement:** install into the game's manual mod path, never the mod.io SDK's cache, which the game's own client syncs and may overwrite.
- **Branding:** show mod.io's mark and name wherever its mods appear (terms 3.3).
- Catalog: add `modio` to the five games above and Gladio Mori. Baldur's Gate 3, Deep Rock Galactic and Core Keeper are separate catalog entries with their own install layouts.

**Traps:**

- A game whose mod.io download option is authorised-only needs the token for every download, and a paid-only one also needs ownership.
- Requests go to the per-key host `u-{user-id}.modapi.io`.
- Shadows of Doubt moved to Steam Workshop: re-check it.

**Done when:** with a test token, Browse lists Timberborn's mod.io mods with all sources sorted by downloads. Installing one checks its MD5, places it in the profile's mod path, and offers its dependencies. A revoked token shows the sign-in prompt.

### The Sims 4 placement speed

- **Decide** how Windows placement copes with large profiles: every Play places every file again (`internal/deploy/place.go`), several times slower on Windows than on Linux (`MORTAR_SCALE=1 go test ./internal/loader/folder -run TestPlaceAndPurgeAThreeThousandFileProfile -v`). Options: show progress; leave files unchanged since the last Play in place; link files where the volume allows, which needs an answer for a mod that rewrites its own file.

### Lethal Company content packs under `mortar/<kind>/`

Mortar's side is built (`internal/devsvc/content.go`, the catalog's `devTemplate.content`). Left:

- The BepInEx 5 template (`Rethunk-Tech/mortar-bepinex5-template`, `Directory.Build.targets` `ContentFile`/`CopyContent`) gets a minor tag and the catalog pins it; until then a code project's ticked `content/` folders exist but its build does not ship them.
- The three mods (Mortar-Posters, Mortar-Boombox, Mortar-Television, #5) scan every package folder under `plugins/` for `mortar/<kind>/` and keep reading the old layouts (`*/LethalPosters/posters`, `*/LethalPosters/tips`, `*/LethalPaintings/paintings`, `*/Custom Songs`, `BepInEx/Custom Songs`, `*/Television Videos`), loading a file present in both once; the known-broken redirects (#10) follow. New mod offers the content pack only once one of them is on Thunderstore.
- The per-kind `assetKinds` entries for posters, paintings, television, boombox, suits and sounds leave the catalog once no supported build predates the `content` entry's `minVersion`.

Done when: a pack with posters + boombox + tv in `plugins/mortar/`, suits in `plugins/moresuits/` and a sound pack loads all three in game; a legacy-layout pack still loads; a code project's ticked content ships in its package.

### Lethal Company facts to verify

Record each in [games.md](games.md#lethal-company) once seen.

- Profile paths with spaces or non-ASCII characters (BepInEx flags `Assembly.LoadFile` issues on some Mono builds, `Entrypoint.cs:94,142`).

## Later

Parked; take up when asked.

- **Develop tab:** a GitHub release from the tag, keeping a project's ThunderstoreDependency items and its development profile's mods in step both ways (Browse's Add as dependency covers adding one), sending the development profile over LAN pairing for co-op tests, and watching the source folders below `src/<Name>` (only that folder is watched).

- **Scrap items without Unity** (Lethal Company): a bundle cannot be built outside the Unity Editor (AssetsTools.NET and UnityPy only patch existing bundles; a new MonoBehaviour's script hash cannot be generated), so the path is a runtime loader mod in mortar-mods on LethalLib: `NetworkPrefabs.CloneNetworkPrefab` (deterministic hash from assembly and name) of template prefabs built once in Unity, an `Item` made with `ScriptableObject.CreateInstance` and passed to `Items.RegisterScrap`, filled from a folder (`item.json`, `texture.png`, `icon.png`, sounds); Mortar's wizard gains a Scrap item kind that writes the folder. Sholiver-LethalCardLoader proves the PNG+JSON half. Arbitrary GLB meshes (glTFast with an HDRP material mapper copied from a vanilla scrap) come after, risking the game's Burst/Collections versions and grip offsets.

- **Translucent window** through the fork's GTK4 `setTransparent()` fix (upstream #6197). Trap: a fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque, and stacked alphas compound toward opaque, so images and overlays each need their own alpha. Windows needs Acrylic with the frameless window. The AppImage is the GTK 3 build (RGBA visual), so it needs the same check there.
- **Frosted glass on Linux** once the desktop runs GNOME 51: in the fork beside `setTransparent()`, bind Wayland `ext_background_effect_manager_v1` and set the toplevel surface's blur region to the whole window, updated on resize, only while the translucent window is on; the AppImage's GTK 3 build reaches the surface through `gdk_wayland_window_get_wl_surface`.
- **Vintage Story dedicated server** profiles (`dotnet VintagestoryServer.dll --dataPath <folder>`), keeping Mods and ModConfig as for a client so a group's server and clients share one mod list.
- **Bridge for IL2CPP games:** a BepInEx 6 IL2CPP build of the BepInEx bridge (`BasePlugin`, .NET 6, Il2CppInterop) with its own packages.
- **MelonLoader beyond 64-bit Windows builds:** a 32-bit game needs `MelonLoader.x86.zip` (read the executable's PE machine type), a native Linux build `MelonLoader.Linux.x64.zip`. A game with MelonLoader as a second loader needs `installerGame` (`internal/profile/layout.go`) and the Thunderstore `router` (`internal/installer/thunderstore.go`) to take the profile's own loader first.
- **GodotModLoader games** (Brotato, Dome Keeper, Random Access Mayhem, Pair-a-Dice): mods load only as `.zip` files from `--mods-path`, so a zipped mod becomes a new kind of placed file through the installer layout, placed-file record, drift and history. The loader pack's `addons/mod_loader` must sit beside the game's executable for a launch (`mod_loader_setup.gd` reads `game_base_dir/addons/mod_loader`), whose first run may write `override.cfg` there; launch with `--script res://addons/mod_loader/mod_loader_setup.gd --enable-mods --mods-path <profile>/mods` (r2modman's GodotMLInstaller and GodotMLGameInstructions). Parked for lack of mods: Thunderstore lists them only for Brotato and Random Access Mayhem, Nexus for Dome Keeper, and Brotato's are on the Steam Workshop.
- **More folder games:** Baldur's Gate 3 (a `modsettings.lsx` writer from each `.pak`'s `meta.lsx`), Mount & Blade II: Bannerlord (install-side target and load order), Planet Zoo (no source names its mods folder), and a loader pack for BepInEx games with no Thunderstore community (Graveyard Keeper).
- **More stores for catalog games** whose folder or marker no source gives: GOG for The Witcher 3 (`1207664663`) and Cyberpunk 2077 (`1423049311`); Darkest Dungeon on GOG and Epic (`_windowsnosteam/win64/Darkest.exe`, a second marker) and its native Linux build; Hogwarts Legacy on Epic; Starbound's Xbox package; Titanfall 2 on the EA app.
- **Other loaders and game families:** Bethesda games (plugin load order, LOOT, script extenders, BSAs); Minecraft (Java instances, Fabric, Forge, NeoForge); Resonite on BepisLoader.
- **Runs that need an outside tester:** a launch from a real GOG install and from Flatpak Steam; a Linux launch with Stardew's launch method set to Steam, confirming SMAPI loads the profile's mods; Import from Vortex against a long-lived Vortex folder; timings with the data folder under OneDrive.
- **Upload a modpack to Thunderstore** from Mortar, as Gale does: a team token in the OS keyring, the submission API's multipart upload behind a preview step. **Export as Thunderstore modpack…** writes the zip for a manual upload.
- **A game version per profile** (Gale #157), e.g. Lethal Company v50 for an old modpack beside the latest: fetch the older build with Steam's depot downloader into a per-version folder the profile launches from. With it, a per-profile "mod versions up to" date (Gale #454) that browse, install and update honour, so the mods match that game build.
- **Translate a mod's description** into the player's language (as StarModsManager does for Stardew), optional and per description; revisit together with the app's own translations.
- **Hosted share service** with short codes and share versioning; an **Add to Mortar** button on Nexus listing tiles, which needs Nexus's approval first ([nexus-application.md](nexus-application.md)).
- **Settings:** an offline mode that never contacts the network.
- **Windows code signing** with Azure Trusted Signing.
- **GitHub topic-search fallback** (`source/github/fallback.go`) into go-github, once a second consumer needs it.
- **Registering Mortar with Nexus:** once Nexus approves ([nexus-application.md](nexus-application.md)), release builds pass `nexussvc.Slug` and `ClientID` as ldflags, and the same change removes the API-key sign-in (go-nexusmods' `WithKey`, `internal/nexussvc/apikey.go`), which is the only sign-in while neither is set.
- **macOS support** (darwin/arm64 and amd64). Parked until the maintainer owns or borrows a Mac; written to be picked up cold.
  - **Scope:** the catalog's `darwin` path templates and the macOS asset picker exist. Profile copies need a darwin `Clone` using `clonefile` in go-safefs (`clone_stub.go` there). Lethal Company and most BepInEx games ship no macOS build, so the first playable targets are native-macOS games (check each catalog game's macOS depot before enabling) and, for Windows-only games, a CrossOver or Whisky prefix.
  - **Wails v3:** darwin builds come from the pinned fork ([stack](architecture.md#stack)) through `wails3 build GOOS=darwin`; add `build/darwin/` (Taskfile, `Info.plist`, `icons.icns`, as `build/windows/` and `build/linux/` do) and `darwin: ./build/darwin/Taskfile.yml` under `includes` in `Taskfile.yml`. cgo and the macOS SDK are needed, so the build runs on a Mac only. Check the fork's WKWebView paths before relying on them: frameless title bar, `BackgroundTypeSolid`, drag regions, the tray (`tray_other.go`), `nxm://` registration (`Info.plist` `CFBundleURLTypes`, replacing `internal/nxm` runtime registration), and single-instance.
  - **Game discovery:** go-gamelocator's `steam/locate_unix.go` (`//go:build !windows`) lists only Linux roots; add `~/Library/Application Support/Steam` (r2modman's darwin game-directory resolver) and, in the same change, `GOOS=darwin go vet ./...` to the module's crosscompile gate, read `steamapps/libraryfolders.vdf` as the Linux path does, and the install folder holds `<Game>.app` (marker `Contents/MacOS/<exe>` or `Contents/Resources/Data`), so `marker` in the catalog needs a `darwinMarker` like `LinuxMarker` (`internal/game/install.go`, `internal/gamestore/standalone.go`). Gale has no macOS code, so r2modman is the only precedent: it launches Steam with `steam_osx -applaunch <id>` (its darwin Steam runner) and a `macos_proxy` launch wrapper (its launch utilities).
  - **Loaders:** BepInEx on macOS starts through the pack's `run_bepinex` script (Doorstop's `libdoorstop.dylib`, injected by the dynamic loader), set as the Steam launch option `"<profile>/run_bepinex" %command%` (r2modman's BepInEx game instructions); `internal/loader/bepinex5` places the Windows `winhttp.dll` pair and needs a darwin branch beside `contributeLinux` in `driver.go` (and thunderpack's), and Steam strips dyld injection for hardened binaries, so test a hardened game early. SMAPI's installer swaps the launcher on macOS as on Linux ([games](games.md#smapi)); `internal/loader/smapi/install.go` hardcodes the Linux names and the bundle layout (`Stardew Valley.app/Contents/MacOS/StardewValley`) differs. MelonLoader ships no macOS build: leave it off, as the driver already refuses Linux.
  - **Packaging:** a `.app` in a `.dmg`, universal or two arch builds; codesign (hardened runtime) and notarise (`notarytool`) with an Apple Developer ID certificate (paid account, a maintainer decision); self-update through `wails3 updater manifest` with a `darwin` artifact in `Taskfile.yml`; a Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap. Without notarisation Gatekeeper blocks the app and the cask would need `xattr` quarantine removal, so do not ship unsigned.
  - **Testing:** the maintainer owns no Mac. Fleet rule: no macOS runners in workflows (billable minutes), so CI stays Linux and Windows; testing needs a lender with a Mac, or a one-off macOS VM or hosted runner run by dispatch the maintainer approves. The pure-Go parts (Steam roots, markers, loader branches) are unit-tested on Linux with a `Platform: "darwin"` view as `internal/loader/thunderpack/thunderpack_test.go` does for Windows.
  - **Done when:** `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://` links; a Stardew Valley profile installs SMAPI, launches through Steam and loads a mod (`SMAPI-latest.txt`); the Steam root and `.app` game folders are found without a manual folder choice; the Homebrew cask installs and updates it; `bun run gate` still passes on Linux with the new darwin unit tests.
