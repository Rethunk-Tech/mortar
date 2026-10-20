# Mortar: decided, not yet built

Decided work, short enough to build cold. Delete an item in the change that lands it; the code and its commits are the record. Screens and styling: [gui-design.md](gui-design.md). Standing rules: [AGENTS.md](../AGENTS.md).

## Release

- Drop the `Rethunk-AI/wails` replace in `go.mod` and pin a tagged beta once one carries what branch `mortar/v3-next` adds to upstream master: #6202 (OnUpdateApplied, with WEP #6203), #6199 (service methods fall back to `Options.MarshalError`; `errorkind_test.go` relies on it), #6240 (single-instance lock released at the start of shutdown), #6197 (GTK4 transparency), #6281 and #6282 (session-bus fallbacks), #6283 (GTK4 drop motion), #6284 (tray Menu property), #6285 (server-mode shutdown hooks), #6287 (pre-Run main-thread queue), WEPs #6243 (local linuxdeploy) and #6244 (`-nopack`), and the fork-only apply-on-exit.
- **NOMAD-only** (Windows Steam with Stardew and SMAPI): set `"<game>\StardewModdingAPI.exe" %command%`, Play once from an account whose user name has a space, and check that `%APPDATA%\StardewValley\ErrorLogs\SMAPI-latest.txt` names the profile's `mods` folder as one path. Otherwise the Steam hand-off needs another form ([architecture.md](architecture.md#launch)). In the same sitting, check that a direct launch with **SMAPI console window** on shows SMAPI's output.

## Ready to build

### The website on Astro

Built on the local `site-astro` branch only. Nothing on it is merged, pushed or deployed until the maintainer has looked through it in `astro preview` and approved it.

**Why:** the generator (`scripts/site.ts` and `scripts/site/*.ts`, about 1,566 lines of string templates) writes 690 pages that are committed under `site/`, so every copy edit churns hundreds of files and the gate fails on stale output (`scripts/gate.sh:126`). Astro keeps generation at build time, gives the pages components and Markdown, and keeps the output out of git.

**Look:** [design/landing-mock/index.html](../design/landing-mock/index.html) is the approved landing page. Its parts, in order:

- **Hero:** a two-line headline, "Download for <your system>" with a version line, and Join the Discord. Beside it, an endless tilted wall of game capsule art that fades out above the metrics.
- **Metrics line:** each figure links to /games/, /sources/, /news/ or /mission/ and counts up the first time it is seen.
- **Demo:** a looping preview, then the interactive demo.
- **How it works:** three steps.
- **Game picker band:** search under the heading, with eight game tiles.
- **Feature cards:** six cards that tilt toward the pointer, with a screenshot that rises on hover.
- **Steam Deck and Your PCs:** large split screenshots.
- **Comparison:** grouped (Reach, Sharing, Everyday). Only supported icons are shown, in brand colour, and columns size to their content.
- **What's new:** picture tiles for the latest release, plus cards for the two before it.
- **Download:** OS cards with your system highlighted, and package-manager chips feeding one copyable command.
- **FAQ:** a Discord button beside the questions.
- **Footer:** a "Ready to mod?" band, then the footer columns.

The rules behind it:

- **Type and colour:** Atkinson Hyperlegible Next, self-hosted through `@fontsource`. Sections sit `--gap` (200px) apart and the content is 82rem wide. No gradients beyond alpha fades.
- **Motion:** sections blur in, the headline settles in, the nav shows the current section and a progress line, and screenshots open in a lightbox. "/" focuses game search. There is a skip link and sand focus rings. All motion is off under `prefers-reduced-motion`.

**Build, stage 1:**

- **Pages:** one layout component replaces `scripts/site/chrome.ts` and the `<!-- x:start -->` blocks. The hand pages in `site/*/index.html` become `.astro` pages.
- **Game pages:** `[game].astro` and `[game]/p.astro` come from `getStaticPaths()` over `playableGames()` (`scripts/site/games.ts`). `site/share/app.js` and `share.test.ts` move along with the share page.
- **Content collections:**
  - FAQ (`scripts/site/faq.md`) and the comparison pages (`scripts/site/why/*.md`).
  - The guide, split by heading out of `docs/user-guide.md` into pages with a sidebar.
  - The bot documents.
  - A new /news collection: one post per release, drafted from `build/release/notes/<tag>.md`, plus how-to posts. It has its own RSS feed.
- **Data:** counts, the version and the rival figures are read at build time from code, `scripts/site/rival-facts.json` and `steam-art.json`. `--refresh` stays. No figure is typed by hand.
- **New pages:** /sources/ (the mod sites) and /mission/ (free, no account, AGPL).
- **SEO:**
  - A generated Open Graph image per page (Satori).
  - JSON-LD: FAQPage, BreadcrumbList, and SoftwareApplication on every page.
  - The sitemap from `@astrojs/sitemap`.
  - Plausible: the maintainer's Business plan has room. The mortar.rethunk.tech site is added only after approval, with a line in the privacy page.
  - Search Console, verified by a DNS record the maintainer adds.
- **Pipeline:**
  - `site/` output leaves git. The gate step becomes `astro check`, `astro build` and the link and picture checks.
  - `scripts/site-downloads.ts` reads the build output.
  - `package-repo.yml` builds Astro before Pagefind (`:93`) and the copy into the `packages` branch (`:105`). App Platform keeps `source_dir: site` (`.do/app.yaml:22`).
- **Interactive demo:** the real frontend over mocked bindings and the e2e seed profile, as a JS island loaded on click. It has three views: the Mods list, a mod's details, and Problems.

**Stage 2:** unique content per game page:

- generated facts: loader, sites, stores, and top mods from the sources' APIs, cached in a data file;
- an optional hand-written note per game.

**Traps:**

- Astro under Bun.
- Biome's `.astro` support.
- Keeping the gate's test time under budget.
- The release-only deploy: a site change ships only with a tag.
- The share page reads its payload in the browser, so it must stay static.
- Pagefind indexes the built output.

**Done when:**

1. Every current URL still resolves, and the new URLs are added on top.
2. Every claim on the old pages appears on the new ones, checked by a text diff of claims.
3. Lighthouse scores 90 or more.
4. axe passes.
5. The share page, search, downloads and RSS work.
6. The maintainer approves it after viewing it locally.

### The Sims 4 before a release carries it

- **Verify** on a real install and record in [sims4.md](sims4.md): the `Options.ini` section header and whether key names match case-insensitively, a localized Documents folder name ("Die Sims 4"), a DX9 or legacy executable, the stock `Resource.cfg` text, EA's statement of the `.ts4script` depth limit, and `gameProcesses`.
- **NOMAD-only** (Steam under Proton, EA App on Windows): a modded profile launches with its script mods running; the game keeps `scriptmodsenabled` in the profile's `Options.ini` copy after a restart; Tray files appear in the game's library; the player's own Mods, saves and `Options.ini` are identical after quitting.
- **Decide** how Windows placement copes with large profiles: every Play places every file again (`internal/deploy/place.go`), which is several times slower on Windows than on Linux (`MORTAR_SCALE=1 go test ./internal/loader/folder -run ThreeThousand -v`). Options: show progress; leave files unchanged since the last Play in place; link files where the volume allows, which needs an answer for a mod that rewrites its own file.
- **Measure** the same test with the Mods folder inside a signed-in OneDrive folder (`MORTAR_SCALE_MODS=<folder>`, `MORTAR_SCALE_KB=<size>`); it needs a person with a Microsoft account.

### First real launches of the newer loaders

No owned game has launched these yet. The first real launch settles: the Wine `Z:` form of the folder flags for ReturnOfModding, GDWeave, Lovely and Northstar (r2modman passes host paths); `wsock32=native,builtin` for Northstar and whether it needs more beside the game than everything but `R2Northstar/mods` (`thunderpack.go` `Spec.Placed`); for MelonLoader, `--melonloader.basedir` under Proton, the `version=native,builtin` override and the `N Mods loaded.` ready line (`driver.go` `Ready`).

## Later

Parked; take up when asked.

- **Translucent window** through the fork's GTK4 `setTransparent()` fix (upstream #6197). Trap: a fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque, and stacked alphas compound toward opaque, so images and overlays each need their own alpha. Windows needs Acrylic with the frameless window.
- **Frosted glass on Linux** once the desktop runs GNOME 51: in the fork beside `setTransparent()` (`v3/pkg/application/linux_cgo.go`), bind Wayland `ext_background_effect_manager_v1` and set the toplevel surface's blur region to the whole window, updated on resize, only while the translucent window is on.
- **Vintage Story dedicated server** profiles (`dotnet VintagestoryServer.dll --dataPath <folder>`), keeping Mods and ModConfig as for a client so a group's server and clients share one mod list.
- **Bridge for IL2CPP games:** a BepInEx 6 IL2CPP build of the BepInEx bridge (`BasePlugin`, .NET 6, Il2CppInterop) with its own packages.
- **MelonLoader beyond 64-bit Windows builds:** a 32-bit game needs `MelonLoader.x86.zip` (read the executable's PE machine type), a native Linux build `MelonLoader.Linux.x64.zip`. A game with MelonLoader as a second loader needs `installerGame` (`internal/profile/layout.go`) and `thunderstoreRules` (`internal/installer/thunderstore.go` `route`) to take the profile's own loader first.
- **GodotModLoader games** (Brotato, Dome Keeper, Random Access Mayhem, Pair-a-Dice): mods load only as `.zip` files from `--mods-path`, so a zipped mod becomes a new kind of placed file through the installer layout, placed-file record, drift and history. The loader pack's `addons/mod_loader` must sit beside the game's executable for a launch (`mod_loader_setup.gd` reads `game_base_dir/addons/mod_loader`), whose first run may write `override.cfg` there; launch with `--script res://addons/mod_loader/mod_loader_setup.gd --enable-mods --mods-path <profile>/mods` (r2modman `GodotMLInstaller.ts`, `GodotMLGameInstructions.ts`). Parked on demand: in October 2026 Thunderstore listed 9 Brotato mods, 8 Random Access Mayhem and none for the other two, Nexus 2 for Dome Keeper, and CurseForge none of the four; Brotato's mods are on the Steam Workshop.
- **More folder games:** Baldur's Gate 3 (a `modsettings.lsx` writer from each `.pak`'s `meta.lsx`), Mount & Blade II: Bannerlord (install-side target and load order), Planet Zoo (no source names its mods folder), and a loader pack for BepInEx games with no Thunderstore community (Graveyard Keeper).
- **More stores for catalog games** whose folder or marker no source gives: GOG for The Witcher 3 (`1207664663`) and Cyberpunk 2077 (`1423049311`); Darkest Dungeon on GOG and Epic (`_windowsnosteam/win64/Darkest.exe`, a second marker) and its native Linux build; Hogwarts Legacy on Epic; Starbound's Xbox package; Titanfall 2 on the EA app.
- **Northstar through Steam's own Proton:** `game/start.go` `command` runs `plan.Exe` directly, so it would run the launcher through the Steam Proton build and the install's `compatdata`.
- **Other loaders and game families:** Bethesda games (plugin load order, LOOT, script extenders, BSAs); Minecraft (Java instances, Fabric, Forge, NeoForge); Resonite on BepisLoader.
- **Runs that need an outside tester:** a launch from a real GOG install and from Flatpak Steam; a Linux launch with Stardew's launch method set to Steam, confirming SMAPI loads the profile's mods; Import from Vortex against a long-lived Vortex folder; timings with the data folder under OneDrive.
- **Upload a modpack to Thunderstore** from Mortar, as Gale does (its `ApiKeyPref` and modpack page): a team token in the OS keyring, the submission API's multipart upload behind a preview step. Today **Export as Thunderstore modpack…** writes the zip for a manual upload.
- **A game version per profile** (Gale #157), e.g. Lethal Company v50 for an old modpack beside the latest: fetch the older build with Steam's depot downloader into a per-version folder the profile launches from. With it, a per-profile "mod versions up to" date (Gale #454) that browse, install and update honour, so the mods match that game build.
- **Translate a mod's description** into the player's language (as StarModsManager does for Stardew), optional and per description; revisit together with the app's own translations.
- **Hosted share service** with short codes and share versioning; the **Xbox app** version (WindowsApps folders are locked down); an **Add to Mortar** button on Nexus listing tiles, which needs Nexus's approval first ([nexus-application.md](nexus-application.md)).
- **Settings:** an offline mode that never contacts the network.
- **EA App libraries on another drive** are found only once the player adds the folder (`internal/gamestore/drivers.go` `eaDirs`).
- **Windows code signing** with Azure Trusted Signing; until then installers stay unsigned and are checked against the signed `SHA256SUMS`.
- **Registering Mortar with Nexus:** SSO and OAuth PKCE are built behind `nexussso.Slug` / `ClientID`, waiting for Nexus approval ([nexus-application.md](nexus-application.md)).
- **Discord release post** drafted beside the release notes, once the new Discord thread exists.
- **UI translations** (strings already go through Lingui).
- **macOS support** (darwin/arm64 and amd64). Parked until the maintainer owns or borrows a Mac; written to be picked up cold.
  - **Why little changes:** the catalog already carries `darwin` path templates (`internal/components/components.go:293`, `internal/runtime/runtime.go:105,159`), `internal/github/pick.go:27` picks macOS assets, `internal/datadir/materialize.go` clones with `clonefile` ([architecture](architecture.md#profile-operations)), and `tray_other.go` builds on darwin. Lethal Company and most BepInEx games ship no macOS build, so the first playable targets are native-macOS games (Stardew Valley; check each catalog game's macOS depot before enabling) and, for Windows-only games, a CrossOver or Whisky prefix.
  - **Wails v3:** darwin builds come from the pinned Rethunk-AI fork (`go.mod:53`, [stack](architecture.md#stack)) through `wails3 build GOOS=darwin`; add `build/darwin/` (Taskfile, `Info.plist`, `icons.icns`, as `build/windows/` and `build/linux/` do) and `darwin: ./build/darwin/Taskfile.yml` beside `Taskfile.yml:14-15`. cgo and the macOS SDK are needed, so the build runs on a Mac only (no cross build from Linux). Check the fork's WKWebView paths before relying on them: frameless title bar (`Frameless: true`, `main.go`), `BackgroundTypeSolid`, drag regions, the tray (`tray_other.go`), `nxm://` registration (`Info.plist` `CFBundleURLTypes`, replacing `internal/nxm` runtime registration), and single-instance.
  - **Game discovery:** `internal/steam/locate_unix.go:7` (`//go:build !windows`) lists only Linux roots; add `~/Library/Application Support/Steam` (r2modman: `src/r2mm/manager/darwin/DarwinGameDirectoryResolver.ts:38`, `SteamDirectory.vue:61`, `GameDirectory.vue:130`), read `steamapps/libraryfolders.vdf` as the Linux path does, and the install folder holds `<Game>.app` (marker `Contents/MacOS/<exe>` or `Contents/Resources/Data`), so `marker` in `components.source.json` needs a `darwinMarker` like `LinuxMarker` (`internal/game/install.go:75`, `internal/gamestore/standalone.go:23`). Gale has no macOS code (only `cfg(target_os = "windows"|"linux")`, `src-tauri/src/profile/launch/platform.rs:42,90`; no macOS release asset), so r2modman is the only precedent: it launches Steam with `steam_osx -applaunch <id>` (`src/r2mm/launching/runners/darwin/SteamGameRunner_Darwin.ts:56`) and a `macos_proxy` launch wrapper (`src/r2mm/launching/LaunchUtils.ts:155`).
  - **Loaders:** BepInEx on macOS starts through the pack's `run_bepinex.sh` (Doorstop `libdoorstop.dylib` via `DYLD_INSERT_LIBRARIES`), set as the Steam launch option `"<profile>/run_bepinex.sh" %command%` (r2modman: `src/r2mm/launching/instructions/instructions/loader/BepInExGameInstructions.ts:30,61`); `internal/loader/bepinex5/bepinex5.go` places the Windows `winhttp.dll` pair and needs a darwin branch beside the Linux one (`internal/loader/thunderpack/thunderpack.go:251`), and Steam strips `DYLD_*` for hardened binaries, so test a hardened game early. SMAPI's installer swaps the launcher on macOS as on Linux ([architecture](architecture.md#smapi)); `internal/loader/smapi/install.go:32-33,88` hardcodes the Linux names and the bundle layout (`Stardew Valley.app/Contents/MacOS/StardewValley`) differs. MelonLoader (`internal/loader/melonloader/driver.go:98`) ships no macOS build: leave it off.
  - **Packaging:** a `.app` in a `.dmg`, universal or two arch builds; codesign (hardened runtime) and notarise (`notarytool`) with an Apple Developer ID certificate (paid Apple Developer account, a maintainer decision); self-update through `wails3 updater manifest` with a `darwin` artifact in `Taskfile.yml:90`; a Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap (it holds `Formula/` today). Without notarisation Gatekeeper blocks the app and the cask would need `xattr` quarantine removal, so do not ship unsigned.
  - **Testing:** the maintainer owns no Mac. Fleet rule: no macOS runners in workflows (billable minutes), so CI stays Linux and Windows; testing needs a lender with a Mac, or a one-off macOS VM or hosted runner run by dispatch the maintainer approves. The pure-Go parts (Steam roots, markers, loader branches) are unit-tested on Linux with a `Platform: "darwin"` view as `internal/loader/thunderpack/thunderpack_test.go:139` does for Windows.
  - **Done when:** `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://` links; a Stardew Valley profile installs SMAPI, launches through Steam and loads a mod (`SMAPI-latest.txt`); the Steam root and `.app` game folders are found without a manual folder choice; the Homebrew cask installs and updates it; `bun run gate` still passes on Linux with the new darwin unit tests.
