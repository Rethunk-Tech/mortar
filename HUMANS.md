# Mortar runbook

How to run, build and gate Mortar. Rules and what it is: [AGENTS.md](AGENTS.md); how it works: [docs/architecture.md](docs/architecture.md); decided work not yet built: [docs/design.md](docs/design.md).

## Prerequisites

- Go (the version in `go.mod`) and [bun](https://bun.sh).
- `wails3`, built inside the `Rethunk-AI/wails` fork's module at the version `go.mod` pins (`go install` ignores `replace`): `go mod download github.com/wailsapp/wails/v3`, then `(v=$(go env GOVERSION); cd "$(go list -m -f '{{.Replace.Dir}}' github.com/wailsapp/wails/v3)" && GOTOOLCHAIN=$v go build -o "$(go env GOPATH)/bin/wails3" ./cmd/wails3)`. Mortar's Go version builds it because the fork's own `go.mod` asks for an older one.
- GTK4 and WebKitGTK 6.0 development packages: `sudo dnf install gtk4-devel webkitgtk6.0-devel gcc-c++ pkgconf-pkg-config`.
- On PATH: `golangci-lint` and `govulncheck` (gate steps), `lefthook` and `gitleaks` (git hooks; pre-commit runs gitleaks on staged changes).

```sh
bun install
lefthook install
```

## Run and build

```sh
wails3 dev      # app with the Vite dev server
wails3 build    # production binary in bin/mortar
wails3 task selftest ARGS="start [--copy-data]"   # sandboxed server-mode Mortar at http://127.0.0.1:9455
wails3 task selftest ARGS=restart               # rebuild and restart after changes
wails3 task selftest ARGS=stop
bun run --cwd frontend e2e                    # UI smoke in its own sandbox
```

- `selftest` (`scripts/selftest.sh`; its header lists every subcommand and variable) runs a `server`-tagged binary with its own HOME, copied (never linked) game folders and a minimal Steam library, so real data, game and Steam config are never touched. `--copy-data` copies live profiles and settings in once (`MORTAR_SELFTEST_DATA` names another Mortar data folder). The sandbox lives under `/var/tmp/mortar-selftest` unless `MORTAR_SELFTEST_DIR` is set. `MORTAR_ENABLE_GAMES` (set from `MORTAR_SELFTEST_ENABLE`) switches on games the bundled manifest lists as disabled. `stop` leaves the folder; `destroy` deletes it, and `reap --yes` deletes idle marked sandboxes under the base dir.
- `e2e` (Playwright, Chromium only; [docs/testing.md](docs/testing.md)) starts its own sandbox, opens every profile tab and Settings page and fails on a console error, a failed request or a main-thread task over 200 ms. It is in neither the gate nor CI; run it after each wave of UI changes.
- The frontend build first runs `scripts/gen-credits.ts` (rewrites `frontend/src/settings/generated/credits.json` from `frontend/package.json` and `go.mod`), then extracts and compiles the Lingui catalogs in `frontend/src/locales/`. Commit those files when they change.

## Command line

With Mortar running, `mortar <command>` asks the open app and prints a table, or JSON with `--json`. `mortar help` lists every command (source: `internal/cli/cli.go`). Output flags (`--json`, `-v`) may precede the verb (`mortar --json install ...`). `<profile>` is a profile id or name; a mod id is `<format>:<local id>` (a bare id uses the game's own format, SMAPI's UniqueID for Stardew). A destructive verb is refused (exit 2) until you pass `--yes`; `install` takes `--allow-unscanned` to install although the antivirus flagged the archive (recorded in history).

`--json` writes failures to stderr as `{"error":"...","kind":"...","code":...}`, plus a `detail` object when the failure has typed fields (a flagged install names `game`, `key`, `name`, `file`, `scanner`). `-v`/`--verbose` prints the raw error. Exit 2 means usage or confirmation was required, 3 means Mortar was not running or `play --check` found issues, 1 any other failure. `mortar doctor` falls back to read-only offline checks when Mortar is not running: it exits 1 when they find problems and 3 when they find none, never 0. `uninstall-cleanup` is not listed by `mortar help`: it needs no running app (packages and the Windows uninstaller run it). Window-launch arguments, not CLI verbs: `--release-links`, `--handle-links <id>`, `--play=<game>/<profile>`, `--steam-session`, `--firewall-allow` (`--firewall-allow=any` on Windows).

```sh
mortar conflicts stardew "Profile 2"
mortar mods stardew bf8012eb5944d3ad --json | jq -r '.[] | select(.enabled | not) | .id'
source <(mortar completion bash)
```

## Gate

```sh
bun run gate    # scripts/gate.sh: bindings first, then every other check in parallel; prints each failing step's log
```

- govulncheck reruns locally only when `go.mod` or `go.sum` changed since its last passing run (stamp in `tmp/gate/`); CI and `MORTAR_GATE_VULNCHECK=1` always run it.
- `main.go` embeds `frontend/dist`, so a fresh clone needs `bun run bindings && bun run --cwd frontend build` once before the gate, as CI's setup does.
- The lefthook pre-push hook runs `gate` ([rethunk-gate-cli](https://github.com/Rethunk-Tech/rethunk-gate-cli)): these steps plus actionlint.
- Where CI runs it: [AGENTS.md](AGENTS.md#verify). What each package's tests cover and the slow, fuzz and property runs: [docs/testing.md](docs/testing.md).

### Stardew regression run (opt-in, not in the gate)

```sh
scripts/selftest.sh regress     # needs the real Stardew install and a display
```

In a throwaway `/var/tmp/mortar-regress-XXXXXX` sandbox on a free port it builds under the shared flock, copies the real data, sets the launch method to direct, hashes the game folder and launches the first profile through the CLI (`MORTAR_REGRESS_PROFILE` picks another). It waits for SMAPI's "Loaded N mods" and "Loaded M content packs" (`MORTAR_REGRESS_TIMEOUT`) and requires N + M plus the mods SMAPI names as skipped to equal the profile's enabled count. It then stops the game by its exe-verified pid, waits for Mortar to go idle, re-hashes and requires an empty diff. A pass deletes its directory; a failure keeps it (`game-before.txt`, `game-after.txt`, `server.log`) and exits non-zero.

### Lethal Company regression run (opt-in, not in the gate)

```sh
scripts/selftest.sh regress --game lethal-company     # needs Lethal Company and Proton - Experimental in the real Steam library
```

Same shape, in `/var/tmp/mortar-regress-lc-XXXXXX`:

- **Setup:** copies the game, Steam's `steamclient.so` files and `.steam` links into the sandbox home (Proton needs the library, not a running Steam; Steam is never started). Every Proton process runs under `bwrap --unshare-net`: a copy registered under its Wine pid in a running Steam's loopback IPC leaves Steam on "Waiting for Lethal Company to shut down" for good. It creates the Proton prefix once, installs BepInEx 5 (`MORTAR_REGRESS_BEPINEX`) plus LethalConfig, ShipLoot and MoreCompany from Thunderstore, and launches the profile through a Proton wrapper set as its launch prefix (`MORTAR_REGRESS_PROTON` names the Proton folder; `MORTAR_REGRESS_MORTAR_PROTON=1` runs the copy as a folder Steam did not install, through Mortar's own Proton runtime).
- **Checks:** BepInEx's "Chainloader startup complete" (`MORTAR_REGRESS_TIMEOUT`), the three plugins' "Loading" lines and, for the bridge, "Loading [Mortar BepInEx Bridge" and "Listening on 127.0.0.1". It stops the game with `mortar stop`, reaps what is left of the sandbox prefix by pid (each verified by its `WINEPREFIX` or `STEAM_COMPAT_DATA_PATH`), waits for idle, re-hashes and requires an empty diff.
- **Cache:** the Thunderstore index and downloads are cached (`MORTAR_REGRESS_CACHE`); `MORTAR_REGRESS_OFFLINE=1` proves a rerun needs no network, with a dead proxy.
- **Bridge:** profiles get the catalog's published [mortar-bepinex-bridge](https://github.com/Rethunk-Tech/mortar-bepinex-bridge) release, which an offline run cannot download and so skips the bridge check. `MORTAR_REGRESS_LOCAL_BRIDGE=1` builds the checkout beside this repo instead (`MORTAR_REGRESS_BRIDGE_REPO=<checkout>` names another and implies it), handed over through `MORTAR_LOCAL_BRIDGES` (a folder holding `<game>.zip`, standing in for the release in `loadersvc.ensureBridge`). The summary says `published` or `local`.
- **Launches** run on the sandbox's hidden display under the session's launch cap ([docs/testing.md](docs/testing.md#sandbox-game-launches)).
- **r2modman import:** `MORTAR_REGRESS_R2_CODE=<key>` adds a second profile imported with `mortar profile import --game lethal-company`. Its downloads need the network (no archive cache). BepInEx must count as many plugins as it logs "Loading [" lines, and the purge must again leave the game folder identical. `MORTAR_REGRESS_R2_DISABLE` lists package ids (`Namespace-Name`) to switch off first. `MORTAR_REGRESS_R2_EXPORT=1` publishes an r2modman code to thunderstore.io (refused together with the matrix, which uses all the session's launches).
- **Trap:** a mod that writes into the game folder at runtime fails the hash (Mortar purges only what it placed). RugbugRedfern-Skinwalkers creates an empty `Dissonance_Diagnostics` folder, so `019c9378-d2e6-4fec-5a2f-e4b22f8cf1d1` passes with `MORTAR_REGRESS_R2_DISABLE=RugbugRedfern-Skinwalkers`.
- A pass deletes its directory; a failure keeps it and prints the path (`LogOutput-<step>.log`, `game-before.txt`, `game-after-<step>.txt`, `server.log`, `wineboot.log`, `r2-*.json`).

## Release

Building the packages and cutting a release, step by step: [docs/release.md](docs/release.md).

## Publishing components

`components.source.json` is the allowlisted input for the signed loader and bridge manifest. The generator resolves its GitHub releases (`github.com` components of kind `loader` and `bridge` only), hashes each asset, refreshes the embedded fallback and verifies a signature when `MORTAR_UPDATE_KEY` names the private key file:

```sh
wails3 task components:refresh
```

- The source's `games` array is the game catalog, copied as is; a key `GameInfo` does not know is dropped, so add the field to `internal/components` first.
- Without `MORTAR_UPDATE_KEY` the generator writes an unsigned manifest, so a dry run needs no key: `go run ./cmd/components -output "$T/components.json" -bundled "$T/bundled.json" -signature "$T/sig"` into a temp dir `$T` (it still reads the components' GitHub releases) leaves the repo untouched, and `bundled.json` should equal `internal/components/components.json` except for `serial`.
- The repository secret `MORTAR_UPDATE_KEY` holds the PEM contents, as for releases. Run the Components workflow from Actions, or dispatch it from a component release:

  ```sh
  gh workflow run components.yml --repo Rethunk-Tech/mortar
  gh api repos/Rethunk-Tech/mortar/dispatches -f event_type=components
  ```

- The workflow creates a `components-v2-<serial>` release (not marked latest) holding `components.json` and `components.json.sig`, each published once (releases are immutable). Its token reads only this repo, so a component on a private repo fails the run with a 404.

## Translations

Translators work in Hosted Weblate, which opens pull requests against `main`. The Lingui catalogs are split for them: `frontend/src/locales/<locale>/{app,mods,settings,profiles,console}.po`, merged into one `messages.ts` per locale at compile (`frontend/lingui.config.ts`).

Weblate project settings:

- One component per catalog. Version control **GitHub pull request**, repository `https://github.com/Rethunk-Tech/mortar.git`, branch `main`. File format **gettext PO file**, file mask `frontend/src/locales/*/<name>.po`, source language English. Leave the template for new translations empty.
- A sixth component holds the text Go shows through the OS (tray, notifications, native dialogs): file mask `internal/i18n/locales/*/native.po`, same settings. A Go test keeps `en/native.po` equal to the `i18n.T` calls.
- Translation license **MIT** on every component (translations ship inside the AGPL-3.0-only app, and MIT lets the strings be reused elsewhere).
- Component flag `icu-message-format`, so Weblate checks placeholders and plurals.
- Project setting **Translation propagation** on: a message used in several folders sits in each of their catalogs.
- Glossary: upload `frontend/src/locales/glossary.tbx` to the project glossary; it explains each recurring term once.
- **Adding new translation:** contact maintainers. Weblate must not run msgmerge; `lingui extract` keeps every locale's catalogs in step with the source.

Starting a language: add its code to `availableLocales` in `frontend/src/i18n/locales.ts`, copy `en/native.po` to `internal/i18n/locales/<locale>/native.po` with empty msgstr and its `Language` and `Plural-Forms` header (a test keeps the two lists equal; the language setting and Go's lookup read the embedded one), run `bun run --cwd frontend i18n:extract` and commit the new catalogs; Weblate picks them up. A regional locale (`es-AR`) needs only the messages where its region differs: empty ones fall back to its base language (`es`), then English. It reaches players with the next release.

## Adding a game

A game is catalog data first and code only where its behaviour differs. `TestEveryCatalogReferenceResolves` (`internal/source/all/catalog_test.go`) fails when an entry names a source, loader, metadata id, store field, path role or path token nothing answers to.

`wails3 task components:audit` lists Thunderstore communities with 20 or more packages that the catalog lacks, and catalog games Nexus hosts for their loader (or CurseForge hosts, with `MORTAR_CURSEFORGE_KEY` set) that have no source there, and fails when a catalog game's community or loader pack has gone from Thunderstore; run it before a release.

### The catalog entry

Add the game to `games` in `components.source.json`, then publish as under [Publishing components](#publishing-components). `GameInfo` (`internal/components/components.go`) holds the full field list and `GameInfo.Validate` rejects an incomplete entry. Fields and their traps:

| Field | Meaning |
| --- | --- |
| `id` | the slug; permanent, because profiles, settings and links store it |
| `name`, `enabled` | display name; false lists the game as coming later |
| `minVersion` | the oldest Mortar that can run the entry (`0.3.6`): set it when the entry needs driver code or catalog fields an older build lacks, and older builds leave the game out |
| `marker`, `markerDir`, `linuxMarker`, `linuxStart` | a file every install holds, at its root or one `game` folder down. `markerDir` is the slash path below the install root that holds it (The Sims 4's `Game/Bin`); the install folder stays the root. A `.exe` marker makes a Windows build. `linuxMarker` names a native executable (Valheim's `valheim.x86_64`); an install holding it and not `marker` runs natively. `linuxStart` is the slash path of the file a native Linux start actually runs when that is not `linuxMarker` (Vintage Story's `run.sh`) |
| `art` | slash path below the install of an image the game ships, used when no store has art |
| `stores` | the game's id per store driver key: `steam.appId`, `gog.productId`, `gog.folder`, `lutris.slug`, `lutris.keyword`, `bottles.folder`, `ea.folder`, `ea.registry` (the HKLM key whose "Install Dir" the installer writes, read on Windows to find a game on another drive; Vortex reads The Sims 4 the same way), `epic.appNames` (one per product sold), `xbox.package`. Empty for a game no store app tracks (an itch.io download), which then sets `manualFolder: true` and the player browses to its folder |
| `templates` | starter profiles for "Starter templates" in the New profile dialog: `id`, `name`, `description`, `packages` (`source`, `ref` (Nexus mod id, GitHub `owner/repo` or Thunderstore `Namespace-Name`), `name`) |
| `paths` | folders and files outside the install, by role, one template per platform; each starts with a token: `{appData}`, `{localAppData}`, `{localLow}`, `{documents}`, `{xdgConfig}`, `{xdgData}`, `{home}` or `{install}`. Roles: `saves`, `errorLogs`, `startupPreferences`; `mods` (a `folder` game's mod folder, filled for a launch, then emptied with the player's files returned); `options` (one settings file; each profile holds its own copy, seeded on first launch and swapped in per launch, `requiredSettings` written into that copy or only reported per the game's settings mode); `tray` (shared by every profile, filled by a target with `role: "tray"`); `userData` (the base `caches` paths resolve under). Any other role a `requiredSettings` or `caches` entry names must be listed too |
| `sources` | each mod site: `key` (the site's name for the game: Nexus domain, Thunderstore community, Modrinth facet such as `categories:fabric`, CurseForge's Mods class id, itch.io search keyword) and `gameId` where the API wants a number. CurseForge may list `classes` (numeric ids a search covers; must contain `key`; merged in the player's sort order). A link and handoff source (`patreon`, `itch`) is just `{"id": "..."}` with no `key`: browse does not search it, a pasted address resolves to it and Mortar opens the page. Catalog order is browse order |
| `loaders` | ids the game can run (`smapi`, `bepinex5`, `melonloader`, `returnofmodding`, `gdweave`, `lovely`, `northstar`, `rivet`, `umm`, `shimloader`, `folder` for a game with none; the test's `knownLoaders` names them), each with optional `companion` (a catalog component of kind `bridge`). BepInEx also takes `package` (a community's own pack, e.g. `denikson-BepInExPack_Valheim`; the standard pack when absent), `major` (6 for IL2CPP on `BepInEx-BepInExPack_IL2CPP`; 5 when absent), `proxy` (the pack's Doorstop proxy when it is not `winhttp.dll`; the doctor's Proton check and a launch before the profile holds a pack read it) and `routes`, from the game's r2modman `installRules` beyond the standard ones: `dir` (a profile-root folder), `extensions` (files ending in one go there wherever they sit) and `track` (`subdir` keeps a package's files below a folder named for it, `subdir-no-flatten` also keeps their path from the package's top); a package folder named like `dir`'s last segment goes there too, so keep r2modman's rule order. MelonLoader takes `routes` with `default` (the route for a file no folder or extension claims); with no `routes` it takes 0.7's recursive layout, and its `package` is `LavaGang-MelonLoader` so a mod's dependency on it is the loader's own. Every Thunderstore-installed loader (`returnofmodding`, `gdweave`, `lovely`, `northstar`, `rivet`, `umm`, `shimloader`) names its Thunderstore `package`, which the loader installs from the game's community; Northstar and Shimloader also take `routes`, and Shimloader `placeDir` (the slash path below the install of the folder its files are placed in, when that is not `markerDir`). A bridge installs only from the game's own `kind: bridge` component, taken from the bridge's GitHub release, never Thunderstore, so a game without one gets no startup timing, in-game performance, stream overlay, intro skip or console commands even when its loader names a `companion`; a loader entry's `devTemplate` (`repo` "owner/name", `ref` a tag) is the GitHub template the Develop tab starts a new mod from |
| `deploy` | `redirect` when the loader points the game at the profile's folder (SMAPI's `--mods-path`), or `profile` when the profile holds the loader and its mods (BepInEx; only the Doorstop proxy is placed into the install for a launch) |
| `targets` | where mod files go: a profile `root` (`{profileMods}`, `{profile}` or a folder below it) and optional `maxDepth`. `keepWhole` (`folder` games) lists extensions that make an archive one entry; an archive with none splits per file. A target with `role` instead of `root` places files at install into that path role's folder, shared by every profile; `extensions` routes file types there, and `sources` (source ids) sends every archive from those sources there whole, below a folder named for the mod, before any loader rule runs (Cities: Skylines II's `pdx` target takes `paradox`) |
| `importIds` | Vortex id (the extension's `GAME_ID`) and MO2 name (the plugin's `GameName`); omit when neither supports it |
| `metadata` | provider ids (`smapi-updates`, `smapi-compat`, `stardew-dataset`) |
| `saveFiles`, `saveCompanions`, `saveCompanionPaths`, `saveCopies`, `saveCopyFolders`, `saveLinked`, `saveUserFolders` | for saves kept as files: name patterns in the saves folder or one below (`!` first excludes), same-stem companion extensions, slash paths under the saves folder of files or folders that belong to one save (`{stem}` is the save file's name without its extension: Cat Mail Co.'s `CatLibSaves/{stem}`; a restore puts them back only when the backup holds them), and patterns for the game's own copies beside a save (`{name}`, `{stem}`, `{ext}`) that backups list and restore. `saveCopyFolders` are folders beside the saves folder that also hold those copies; `saveLinked` are slash paths beside the saves folder of files that belong to one save (`{uuid}` is the world's id). `saveUserFolders` puts the `saveFiles` one folder per platform user below the saves folder (Cities: Skylines II's `Saves/<Steam id>`). Without `saveFiles` a save is a folder holding a file of its own name |
| `folderNames` | other names a `paths` folder goes by on some installs (The Sims 4's user folder follows the game's language); the one that exists and changed last wins |
| `requiredSettings` | `{path, section, key, value, message}` in the game's own `key = value` files; `path` is a path role, `section` optional. Problems reports another value; Mortar writes only into a profile's own copy of an `options` file |
| `newProfileSeparateSaves` | a new profile starts with its own saves folder |
| `caches` | `{role, path}` rebuildable files the game recreates; deleted once on the first launch after a profile's mod set changes |
| `knownBroken` | curated mods or Thunderstore packages that do not work: `id`, optional `versions` (space-separated constraints), `reason`, optional `replacement` |
| `titleScene` | Unity scene of the main menu, where a BepInEx startup measurement ends; without it the first scene ends it |
| `graphics` | player choice of graphics API (`explanation`, `choices` with `id`/`label`/`args`, `recommended`, `reason`, `source`) |
| `r2modmanFolder` | r2modman's name for the game (the Thunderstore schema's `internalFolderName`) |

The Sims 4 (`sims4` in `components.source.json`) is the worked `folder` game: loader `folder`, `deploy` `profile`, a `mods` target with `keepWhole` `["ts4script"]` (a script mod stays whole, a package-only archive splits per file) and a `tray` target with `role` `tray`.

### A catalog-only game, end to end

A game on a loader, store and sources Mortar already has needs no Go: `game.Find` answers for every catalog entry, and only a game the entry cannot describe is a type in `internal/game` listed in `coded` (Stardew Valley, for the process names SMAPI's install leaves). Lethal Company is the worked example (Steam, BepInEx 5, Thunderstore, Nexus, CurseForge, GitHub).

1. **Entry.** Add it to `games` with `enabled: false` while checking. Lethal Company's names `stores.steam.appId`, `loaders: [{id: "bepinex5", companion: "bridge"}]`, `sources` for `thunderstore` (`key` is the community slug), `nexus` (domain `key` and numeric `gameId`) and `github`, `r2modmanFolder`, `paths.saves`, `deploy: "profile"` and the `profile` target.
2. **Components.** A loader whose release Mortar installs (SMAPI) or a companion (the bridge) is a `components` entry with a GitHub source, an asset pattern and a version policy. BepInEx ships as a Thunderstore package and needs none.
3. **Manifest.** Regenerate the embedded manifest ([Publishing components](#publishing-components)).
4. **Check.** `go test ./internal/components ./internal/source/all` runs `GameInfo.Validate` and `TestEveryCatalogReferenceResolves`.
5. **Try it.** `TestACatalogOnlyGameNeedsNoCode` (`internal/game`: a test-only Content Warning entry is selectable, found on Steam and given BepInEx) and `TestACatalogOnlyGameBrowsesAndAdds` (`internal/browse`) are the pattern: put the entry in a test manifest selected with `components.Use` (never the shipped catalog). In the sandbox, `MORTAR_ENABLE_GAMES=<id>` ([Run and build](#run-and-build)) switches on a game the bundled manifest lists as disabled, so the entry must be in `components.source.json` and the regenerated bundled manifest first. `scripts/selftest.sh` copies Stardew Valley, Lethal Company and Valheim (when installed) into its Steam library and `seed` fills profiles for the first two; a new game needs its app id and folder added to the script.
6. **Enable.** Flip `enabled` to true and publish the manifest.

### Drivers

A new store, site, runtime, provider, host, installer, deployer, framework or loader is one Go package or file plus its registration. A game that uses only existing ones needs none of this.

| Driver | Package | Implement and register |
| --- | --- | --- |
| Store | `internal/gamestore` | `Store` (`Key`, `Launchers`, `Discover`) in `drivers.go`; add to `All()` and give `components.GameStores` the matching field. `Key` is the name in the catalog's `stores` |
| Source | `internal/source` | `Source` (`ID`, `Name`, `Modes`) in `internal/source/<id>/`; `source.Register` in a package variable; import in `internal/source/all/all.go`. Capabilities by type assertion: `Searcher`, `Categorizer`, `Gated` (`Unavailable()` says what the player must set up, e.g. itch.io's API key; browse greys the source out), `Schemer`, `PageLinker` (`ModPageURL(gameKey, id)`), `Hoster`, `LinkOptIn` (scheme claimed only when the player opts in; Thunderstore's `ror2mm`). A searcher sets `Item.Adult` from the site's flag, since browse hides adult hits unless opted in. Tests serve the site from a fake HTTP server |
| Runtime | `internal/runtime` | `Runtime` (`ID`, `Detect`, `Resolve`) in `drivers` before `native`, which claims whatever nothing else does |
| Metadata provider | `internal/metadata` | the capability (`Updates`, `Status` or `Dataset`) and its id's case in `For` |
| Host | `internal/host` | `Host` (`ID`, `Match`, `Fetch`) in `For` ahead of the direct host, which takes any HTTPS URL |
| Pack format | `internal/pack` | `Format` reading another manager's list into a `Draft`; pass it to `pack.Read` where the importer lists formats. Writes nothing |
| Installer | `internal/installer` | `Installer` (`ID`, `Detect`, `Layout`); `installer.Register`; id in `detectionOrder` before `plain`, which takes any archive. Every `Layout` path is validated to stay inside its target |
| Framework | `internal/framework` | `Framework` (`ID`, `Matches`, `Analyze`); `framework.Register`. Content Patcher is the one driver |
| Loader | `internal/loader` | `Loader` (`ID`, `Formats`, `Status`, `Install`, `Contribute`) in `internal/loader/<id>/`; `loader.Register`; import in `internal/loader/all/all.go`. Capabilities by type assertion: `Vanilla`, `Owner`, `WithLogs`, `WithConfig`, `WithLaunchSettings` (shown in Edit profile), `WithPlayerLog`, `WithCompanion`, `Console`, `Releases`, `ProcessNames`, `SteamExe`. A loader that redirects the game to the profile (SMAPI) declares no install-side files; one that needs files beside the executable (BepInEx's Doorstop `winhttp.dll`, `doorstop_config.ini`) declares them in its plan and the deployer places them |

### Self-test

```sh
go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime ./internal/game ./internal/browse
scripts/selftest.sh start
```

`scripts/selftest.sh setup` creates the sandbox's Steam library only.
