# Mortar: decided, not yet built

Decided work, short enough to build cold. Delete an item in the change that lands it; the code and its commits are the record. Screens and styling: [gui-design.md](gui-design.md). Standing rules: [AGENTS.md](../AGENTS.md).

## Release

- Drop the `Rethunk-AI/wails` replace in `go.mod` and pin a tagged beta once one carries what branch `mortar/v3-next` adds to upstream master: #6202 (OnUpdateApplied, with WEP #6203), #6199 (service methods fall back to `Options.MarshalError`; `errorkind_test.go` relies on it), #6240 (single-instance lock released at the start of shutdown), #6197 (GTK4 transparency), #6281 and #6282 (session-bus fallbacks), #6283 (GTK4 drop motion), #6284 (tray Menu property), #6285 (server-mode shutdown hooks), #6287 (pre-Run main-thread queue), WEPs #6243 (local linuxdeploy) and #6244 (`-nopack`), and the fork-only apply-on-exit.
- **NOMAD-only** (Windows Steam with Stardew and SMAPI): set `"<game>\StardewModdingAPI.exe" %command%`, Play once from an account whose user name has a space, and check that `%APPDATA%\StardewValley\ErrorLogs\SMAPI-latest.txt` names the profile's `mods` folder as one path. Otherwise the Steam hand-off needs another form ([architecture.md](architecture.md#launch)). In the same sitting, check that a direct launch with **SMAPI console window** on shows SMAPI's output.

## Ready to build

### The website on Astro

**Stage 2:** unique content per game page, beside the loader, sites and stores it already shows (`web/src/pages/[game]/index.astro:64-73`):

- top mods from the sources' APIs, cached in a data file under `web/src/data/` that `bun run site:refresh` writes (`web/scripts/refresh.ts` writes only `rival-facts.json` and `steam-art.json` today);
- an optional hand-written note per game.

### mod.io as a source

**Why:** mod.io is an official mod channel for Blade & Sorcery, BONELAB, Ready or Not, S.T.A.L.K.E.R. 2 and Timberborn, and the only one for Gladio Mori. Outside the catalog it is how Baldur's Gate 3 (Larian supports mods only through mod.io) and Deep Rock Galactic ship mods, and it would bring back Core Keeper (dropped in 0.3.9 for lack of a live source). Neither r2modman (it declined, issue #942) nor Gale supports it.

**First, before code:** create a mod.io API key at mod.io/me/access and read each target game's `api_access_options` with `GET /v1/games/{id}`. Only a game with the `OPEN` bit allows third-party access, so build for those alone. When requesting the key, write to mod.io about what Mortar does: API Access Terms 3.5 forbids "a competing or replacement product".

**Decided shape:**

- A `internal/source/modio` driver shaped like `internal/source/modrinth`: `Search`, `Sorts` (downloads, updated, newest), `Details`, files and dependencies.
- **Sign-in per user**, like itch.io (`internal/source/itch/service.go:32`). The player pastes a read-only OAuth token from mod.io/me/access, Mortar tests it and keeps it in the keyring (`internal/secret`), and Settings › Accounts shows the account. A key shipped in the binary does not work: a user key is capped at 60 requests per minute across every install sharing it, while a user's own token gets 120. mint, a Deep Rock Galactic manager, works the same way (`trumank/mint` `src/providers/modio.rs:300-305`). Tokens have no refresh, so an expired token asks the player to sign in again.
- **Downloads:** fetch the modfile again right before each download, since `binary_url` can expire. Check the file's MD5 (the only hash mod.io gives), and send `X-Modio-Platform: windows` for Proton games and `linux` for native ones. Honour `retry-after` on a 429.
- **Dependencies** come from `?recursive=true`, but some creators list soft "credit" links, so a dependency is offered, never installed unasked.
- **Placement:** install into the game's manual mod path, never the mod.io SDK's cache, which the game's own client syncs and may overwrite.
- **Branding:** show mod.io's mark and name wherever its mods appear (terms 3.3).
- Catalog: add `modio` to the five games above and Gladio Mori. Baldur's Gate 3, Deep Rock Galactic and Core Keeper are separate catalog entries with their own install layouts, sized per game.

**Traps:**

- A game that sets `AUTHORISED_DOWNLOADS` needs the token for every download, and `PAID_DOWNLOADS` also needs ownership.
- Requests go to the per-key host `u-{user-id}.modapi.io`.
- Re-check Shadows of Doubt: it moved to Steam Workshop in January 2026.

**Done when:** with a test token, Browse lists Timberborn's mod.io mods with all sources sorted by downloads. Installing one checks its MD5, places it in the profile's mod path, and offers its dependencies. A revoked token shows the sign-in prompt.

### The Sims 4 before a release carries it

- **Verify** on a real install and record in [sims4.md](sims4.md): the `Options.ini` section header and whether key names match case-insensitively, and EA's statement of the `.ts4script` depth limit.
- **Process names:** the `sims4` catalog entry has no `gameProcesses`, so running-game detection falls back to the marker (`internal/game/catalog_only.go:35`) and misses `TS4_DX9_x64.exe`; add both executables.
- **NOMAD-only** (Steam under Proton, EA App on Windows): a modded profile launches with its script mods running; the game keeps `scriptmodsenabled` in the profile's `Options.ini` copy after a restart; Tray files appear in the game's library; the player's own Mods, saves and `Options.ini` are identical after quitting.
- **Decide** how Windows placement copes with large profiles: every Play places every file again (`internal/deploy/place.go`), which is several times slower on Windows than on Linux (`MORTAR_SCALE=1 go test ./internal/loader/folder -run ThreeThousand -v`). Options: show progress; leave files unchanged since the last Play in place; link files where the volume allows, which needs an answer for a mod that rewrites its own file.
- **Measure** the same test with the Mods folder inside a signed-in OneDrive folder (`MORTAR_SCALE_MODS=<folder>`, `MORTAR_SCALE_KB=<size>`); it needs a person with a Microsoft account.

### First real launches of the newer loaders

No owned game has launched these yet. The first real launch settles: the Wine `Z:` form of the folder flags for ReturnOfModding, GDWeave, Lovely and Northstar (r2modman passes host paths); `wsock32=native,builtin` for Northstar (`internal/loader/thunderpack/specs.go:139`) and whether it needs more beside the game than everything but `R2Northstar/mods` (its `Placed`, `specs.go:131`); for MelonLoader, `--melonloader.basedir` under Proton, the `version=native,builtin` override and the `N Mods loaded.` ready line (`driver.go` `Ready`).

## Later

Parked; take up when asked.

- **Translucent window** through the fork's GTK4 `setTransparent()` fix (upstream #6197). Trap: a fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque, and stacked alphas compound toward opaque, so images and overlays each need their own alpha. Windows needs Acrylic with the frameless window.
- **Frosted glass on Linux** once the desktop runs GNOME 51: in the fork beside `setTransparent()` (`v3/pkg/application/linux_cgo.go`), bind Wayland `ext_background_effect_manager_v1` and set the toplevel surface's blur region to the whole window, updated on resize, only while the translucent window is on.
- **Vintage Story dedicated server** profiles (`dotnet VintagestoryServer.dll --dataPath <folder>`), keeping Mods and ModConfig as for a client so a group's server and clients share one mod list.
- **Bridge for IL2CPP games:** a BepInEx 6 IL2CPP build of the BepInEx bridge (`BasePlugin`, .NET 6, Il2CppInterop) with its own packages.
- **MelonLoader beyond 64-bit Windows builds:** a 32-bit game needs `MelonLoader.x86.zip` (read the executable's PE machine type), a native Linux build `MelonLoader.Linux.x64.zip`. A game with MelonLoader as a second loader needs `installerGame` (`internal/profile/layout.go:30`) and the Thunderstore `router` (`internal/installer/thunderstore.go:62`) to take the profile's own loader first.
- **GodotModLoader games** (Brotato, Dome Keeper, Random Access Mayhem, Pair-a-Dice): mods load only as `.zip` files from `--mods-path`, so a zipped mod becomes a new kind of placed file through the installer layout, placed-file record, drift and history. The loader pack's `addons/mod_loader` must sit beside the game's executable for a launch (`mod_loader_setup.gd` reads `game_base_dir/addons/mod_loader`), whose first run may write `override.cfg` there; launch with `--script res://addons/mod_loader/mod_loader_setup.gd --enable-mods --mods-path <profile>/mods` (r2modman `GodotMLInstaller.ts`, `GodotMLGameInstructions.ts`). Parked on demand: in October 2026 Thunderstore listed 9 Brotato mods, 8 Random Access Mayhem and none for the other two, Nexus 2 for Dome Keeper, and CurseForge none of the four; Brotato's mods are on the Steam Workshop.
- **More folder games:** Baldur's Gate 3 (a `modsettings.lsx` writer from each `.pak`'s `meta.lsx`), Mount & Blade II: Bannerlord (install-side target and load order), Planet Zoo (no source names its mods folder), and a loader pack for BepInEx games with no Thunderstore community (Graveyard Keeper).
- **More stores for catalog games** whose folder or marker no source gives: GOG for The Witcher 3 (`1207664663`) and Cyberpunk 2077 (`1423049311`); Darkest Dungeon on GOG and Epic (`_windowsnosteam/win64/Darkest.exe`, a second marker) and its native Linux build; Hogwarts Legacy on Epic; Starbound's Xbox package; Titanfall 2 on the EA app.
- **Northstar through Steam's own Proton:** `game/start.go` `command` runs `plan.Exe` directly, so it would run the launcher through the Steam Proton build and the install's `compatdata`.
- **Other loaders and game families:** Bethesda games (plugin load order, LOOT, script extenders, BSAs); Minecraft (Java instances, Fabric, Forge, NeoForge); Resonite on BepisLoader.
- **Runs that need an outside tester:** a launch from a real GOG install and from Flatpak Steam; a Linux launch with Stardew's launch method set to Steam, confirming SMAPI loads the profile's mods; Import from Vortex against a long-lived Vortex folder; timings with the data folder under OneDrive.
- **Upload a modpack to Thunderstore** from Mortar, as Gale does (its `ApiKeyPref` and modpack page): a team token in the OS keyring, the submission API's multipart upload behind a preview step. Today **Export as Thunderstore modpack…** writes the zip for a manual upload.
- **A game version per profile** (Gale #157), e.g. Lethal Company v50 for an old modpack beside the latest: fetch the older build with Steam's depot downloader into a per-version folder the profile launches from. With it, a per-profile "mod versions up to" date (Gale #454) that browse, install and update honour, so the mods match that game build.
- **Translate a mod's description** into the player's language (as StarModsManager does for Stardew), optional and per description; revisit together with the app's own translations.
- **Hosted share service** with short codes and share versioning; an **Add to Mortar** button on Nexus listing tiles, which needs Nexus's approval first ([nexus-application.md](nexus-application.md)).
- **Settings:** an offline mode that never contacts the network.
- **Windows code signing** with Azure Trusted Signing; until then installers stay unsigned and are checked against the signed `SHA256SUMS`.
- **Registering Mortar with Nexus:** SSO and OAuth PKCE are built behind `nexussso.Slug` / `ClientID` (`internal/nexussso/sso.go:18-23`). Once Nexus approves ([nexus-application.md](nexus-application.md)), release builds pass both as ldflags.
- **UI translations** (strings already go through Lingui).
- **macOS support** (darwin/arm64 and amd64). Parked until the maintainer owns or borrows a Mac; written to be picked up cold.
  - **Why little changes:** the catalog already carries `darwin` path templates (`internal/components/components.go:335`, `internal/runtime/runtime.go:107,161`), `internal/github/pick.go:27` picks macOS assets, and `tray_other.go` builds on darwin. Profile copies need a darwin `cloneFile` using `clonefile`: today darwin builds `internal/datadir/clone_stub.go:7`, which returns `EOPNOTSUPP`. Lethal Company and most BepInEx games ship no macOS build, so the first playable targets are native-macOS games (Stardew Valley; check each catalog game's macOS depot before enabling) and, for Windows-only games, a CrossOver or Whisky prefix.
  - **Wails v3:** darwin builds come from the pinned Rethunk-AI fork (`go.mod:53`, [stack](architecture.md#stack)) through `wails3 build GOOS=darwin`; add `build/darwin/` (Taskfile, `Info.plist`, `icons.icns`, as `build/windows/` and `build/linux/` do) and `darwin: ./build/darwin/Taskfile.yml` beside `Taskfile.yml:14-15`. cgo and the macOS SDK are needed, so the build runs on a Mac only (no cross build from Linux). Check the fork's WKWebView paths before relying on them: frameless title bar (`Frameless: true`, `main.go`), `BackgroundTypeSolid`, drag regions, the tray (`tray_other.go`), `nxm://` registration (`Info.plist` `CFBundleURLTypes`, replacing `internal/nxm` runtime registration), and single-instance.
  - **Game discovery:** `internal/steam/locate_unix.go:7` (`//go:build !windows`) lists only Linux roots; add `~/Library/Application Support/Steam` (r2modman: `src/r2mm/manager/darwin/DarwinGameDirectoryResolver.ts:38`, `SteamDirectory.vue:61`, `GameDirectory.vue:130`), read `steamapps/libraryfolders.vdf` as the Linux path does, and the install folder holds `<Game>.app` (marker `Contents/MacOS/<exe>` or `Contents/Resources/Data`), so `marker` in `components.source.json` needs a `darwinMarker` like `LinuxMarker` (`internal/game/install.go:75`, `internal/gamestore/standalone.go:23`). Gale has no macOS code (only `cfg(target_os = "windows"|"linux")`, `src-tauri/src/profile/launch/platform.rs:42,90`; no macOS release asset), so r2modman is the only precedent: it launches Steam with `steam_osx -applaunch <id>` (`src/r2mm/launching/runners/darwin/SteamGameRunner_Darwin.ts:56`) and a `macos_proxy` launch wrapper (`src/r2mm/launching/LaunchUtils.ts:155`).
  - **Loaders:** BepInEx on macOS starts through the pack's `run_bepinex.sh` (Doorstop `libdoorstop.dylib` via `DYLD_INSERT_LIBRARIES`), set as the Steam launch option `"<profile>/run_bepinex.sh" %command%` (r2modman: `src/r2mm/launching/instructions/instructions/loader/BepInExGameInstructions.ts:30,61`); `internal/loader/bepinex5` places the Windows `winhttp.dll` pair and needs a darwin branch beside the Linux one (`internal/loader/bepinex5/driver.go:114`, `contributeLinux` `:135`; thunderpack's at `internal/loader/thunderpack/thunderpack.go:284`), and Steam strips `DYLD_*` for hardened binaries, so test a hardened game early. SMAPI's installer swaps the launcher on macOS as on Linux ([architecture](architecture.md#smapi)); `internal/loader/smapi/install.go:32-33,88` hardcodes the Linux names and the bundle layout (`Stardew Valley.app/Contents/MacOS/StardewValley`) differs. MelonLoader ships no macOS build: leave it off, as the driver already refuses Linux (`internal/loader/melonloader/driver.go:113-114`).
  - **Packaging:** a `.app` in a `.dmg`, universal or two arch builds; codesign (hardened runtime) and notarise (`notarytool`) with an Apple Developer ID certificate (paid Apple Developer account, a maintainer decision); self-update through `wails3 updater manifest` with a `darwin` artifact in `Taskfile.yml:90`; a Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap (it holds `Formula/` today). Without notarisation Gatekeeper blocks the app and the cask would need `xattr` quarantine removal, so do not ship unsigned.
  - **Testing:** the maintainer owns no Mac. Fleet rule: no macOS runners in workflows (billable minutes), so CI stays Linux and Windows; testing needs a lender with a Mac, or a one-off macOS VM or hosted runner run by dispatch the maintainer approves. The pure-Go parts (Steam roots, markers, loader branches) are unit-tested on Linux with a `Platform: "darwin"` view as `internal/loader/thunderpack/thunderpack_test.go:164` does for Windows.
  - **Done when:** `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://` links; a Stardew Valley profile installs SMAPI, launches through Steam and loads a mod (`SMAPI-latest.txt`); the Steam root and `.app` game folders are found without a manual folder choice; the Homebrew cask installs and updates it; `bun run gate` still passes on Linux with the new darwin unit tests.
