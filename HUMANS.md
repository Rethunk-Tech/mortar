# Mortar runbook

How to run, build and gate Mortar. What it is and the rules it keeps: [AGENTS.md](AGENTS.md); how it works: [docs/architecture.md](docs/architecture.md); decided work not yet built: [docs/design.md](docs/design.md).

## Prerequisites

- Go (the version in `go.mod`) and [bun](https://bun.sh)
- The Wails CLI, `wails3`, built inside the `Rethunk-AI/wails` fork's module at the version `go.mod` pins, since `go install` ignores `replace`: `go mod download github.com/wailsapp/wails/v3`, then `(v=$(go env GOVERSION); cd "$(go list -m -f '{{.Replace.Dir}}' github.com/wailsapp/wails/v3)" && GOTOOLCHAIN=$v go build -o "$(go env GOPATH)/bin/wails3" ./cmd/wails3)` (built with Mortar's Go version; the fork's own `go.mod` asks for an older one)
- GTK4 and WebKitGTK 6.0 development packages: `sudo dnf install gtk4-devel webkitgtk6.0-devel gcc-c++ pkgconf-pkg-config`
- `golangci-lint` and `govulncheck` (the gate's lint and vuln steps), and `lefthook` and `gitleaks` (the git hooks; pre-commit runs gitleaks on staged changes), on PATH

```sh
bun install
lefthook install
```

## Run and build

```sh
wails3 dev      # app with the Vite dev server
wails3 build    # production binary in bin/mortar
wails3 task selftest -- start [--copy-data]   # sandboxed server-mode Mortar at http://127.0.0.1:9455
wails3 task selftest -- restart               # rebuild and restart after changes
wails3 task selftest -- stop
bun run --cwd frontend e2e                    # UI smoke against the running self-test server
```

`selftest` runs `scripts/selftest.sh`: a `server`-tagged binary, its own HOME, a copied (never linked) Stardew folder and a minimal Steam library, so the real data, game and Steam config are never touched. Optional `--copy-data` copies live Mortar profiles and settings into that sandbox once. The sandbox lives under `/var/tmp/mortar-selftest` unless `MORTAR_SELFTEST_DIR` is set; the script does not delete it.

`e2e` (Playwright, Chromium only) opens every profile tab and every Settings page of the running self-test server and fails on a console error, a failed request, or a main-thread task over 200 ms. It reports itself skipped when no server is running, so it is not part of the gate or CI; run it after each wave of UI changes (`restart` first).

The frontend build first runs `scripts/gen-credits.ts`, which rewrites `frontend/src/settings/generated/credits.json` (the licence list on Settings › About) from `frontend/package.json` and `go.mod`, then extracts and compiles the Lingui catalogs in `frontend/src/locales/`, so a build never shows a message id in place of its text; commit those files when they change.

## Command line

With Mortar running, `mortar <command>` asks the open app and prints a table, or JSON with `--json`; `mortar help` lists every command. `<profile>` is a profile id or name.

| Command | Does |
| --- | --- |
| `games` | supported games and whether each is configured |
| `profiles <game>`, `profile create\|rename\|copy\|delete\|repair ...` | list and manage profiles |
| `trash list [--game stardew]`, `trash restore\|delete\|empty ...` | recently deleted profiles (`delete` and `empty` need `--yes`) |
| `profile list <game> <profile> --format md\|text` | enabled mods: name, version, Nexus link when known |
| `profile compare <game> <A> <B>` | mods only in A, only in B, version or enabled differences |
| `play <game> <profile> --check` | pre-Play summary; exits 3 when anything is wrong |
| `mods tag\|untag\|category\|note\|skip-version ...` | the Mods selection-bar setters |
| `settings export\|import <file>`, `settings reset [key] [--game id]` | portable settings and restore defaults |
| `profile match <game> <profile> <link-or-file>` | preview a friend's share against a profile |
| `profile history <game> <profile>`, `profile revert <game> <profile> <eventId>` | restore points; revert to one |
| `history <game> --all` | recent changes across that game's profiles |
| `profile load-order <game> <profile>` | enabled mods in SMAPI load order |
| `mods <game> <profile>`, `mods enable\|disable\|pin\|unpin\|remove ... <mod id>...`, `mod ... <mod id>` | list, change and inspect mods (a mod id is `<format>:<local id>`; a bare id means the game's own format, SMAPI's UniqueID for Stardew) |
| `install <game> <profile> <archive>` | install a local archive |
| `conflicts`, `problems [--format text]`, `updates`, `saves <game> <profile>` | what the Problems, Mods and Saves tabs show |
| `problems dismissed`, `problems dismiss <index>`, `problems restore <token\|index>` (`--profile`, `--game stardew`) | dismiss and restore Problems-tab warnings like the GUI |
| `share`, `export <game> <profile> [file]`, `open <link\|file>` | share links and `.mortar` files |
| `launch <game> <profile> [--preset NAME] [--wait]`, `status`, `stop <game>`, `runs`, `logs`, `logs search <query> [--profile <name>]` | play and read past runs |
| `launchers`, `launchers add\|remove <id> <folder>` | what Settings › Launchers shows and changes |
| `tools <game>`, `tools run <game> <profile> <tool>` | configured external tools; start one |
| `bundles <game>`, `bundles apply <game> <bundle> <profile>` | list saved bundles; copy one into a profile |
| `nexus untrack <game> --all\|--unused` | untrack that game's Nexus mods (`--yes` skips the prompt) |
| `update game profile <mod id>...\|--all` | queue selected or all available mod updates |
| `queue retry\|skip [id]`, `queue pause\|resume\|clear` | control queued downloads |
| `backups list`, `backups create <save>`, `backups restore <name> [save...]` | list, pin a Manual backup of one save, or restore |
| `backups usage`, `backups trim --keep N` | save backup sizes; keep the newest N per save (pinned ones stay) |
| `history usage <game>`, `history trim <game> <profile> --keep N` | profile history sizes; keep the newest N changes |
| `templates list\|save\|delete\|new ...` | profile templates; `new` starts a profile from one |
| `library extra <game>`, `library hidden <game> <profile>` | mods in the extra mods folder; dot-hidden mods inside a profile |
| `library old-files <game> <profile> [--keep\|--delete KEY]`, `library strays <game> [<profile> --move FOLDER...] [--dismiss FOLDER]` | files an update dropped; folders in the game's Mods folder that Mortar does not track |
| `queue retry-failed`, `data location`, `archive preview <path>`, `archive downloads <game>` | retry every failed download; where the data folder is and whether it is portable; an archive's contents; archives in the downloads folder |
| `queue`, `doctor`, `version`, `completion bash\|zsh\|fish` | the download queue, the environment, shell completion |

`--json` writes failures to stderr as `{"error":"...","code":...}`. Exit code 2 means usage or confirmation was required, 3 means Mortar was not running or `play --check` found issues, and 1 means another failure. `mortar doctor` falls back to read-only offline checks when Mortar is not running: it exits 1 when those checks find problems and 3 when they find none, never 0.

```sh
mortar conflicts stardew "Profile 2"
mortar mods stardew bf8012eb5944d3ad --json | jq -r '.[] | select(.enabled | not) | .id'
source <(mortar completion bash)
```

## Gate

```sh
bun run gate    # scripts/gate.sh: bindings first, then every other check in parallel; prints each failing step's log
```

`main.go` embeds `frontend/dist`, so a fresh clone needs `bun run bindings && bun run --cwd frontend build` once before the gate, as CI's setup does. The lefthook pre-push hook runs `gate` ([rethunk-gate-cli](https://github.com/Rethunk-Tech/rethunk-gate-cli)), which runs these steps plus actionlint. Where CI runs it: [AGENTS.md](AGENTS.md#verify).

### Stardew regression run (opt-in, not in the gate)

```sh
scripts/selftest.sh regress     # about 40 s warm, needs the real Stardew install and a display
```

One command in a throwaway `/var/tmp/mortar-regress-XXXXXX` sandbox on a free port: builds under the shared flock, copies the real data, sets the launch method to direct, hashes the game folder (type, mode, path, link target or sha256, sorted), launches the first profile through the CLI (`MORTAR_REGRESS_PROFILE` picks another) and waits for SMAPI's "Loaded N mods" and "Loaded M content packs" (`MORTAR_REGRESS_TIMEOUT`, default 180 s). It requires N + M plus the mods SMAPI names as skipped to equal the profile's enabled count, stops the game by its exe-verified pid, waits for Mortar to go idle, re-hashes and requires an empty diff. It exits non-zero on any failure and prints a short summary; a pass deletes only its own directory, a failure keeps it (`game-before.txt`, `game-after.txt`, `server.log`).

### Lethal Company regression run (opt-in, not in the gate)

```sh
scripts/selftest.sh regress --game lethal-company     # about 50 s, needs Lethal Company and Proton - Experimental in the real Steam library
```

Same shape as the Stardew run, in `/var/tmp/mortar-regress-lc-XXXXXX`: copies the game, copies Steam's `steamclient.so` files and `.steam` links into the sandbox home (Proton needs the library, not a running Steam; Steam is never started), creates the Proton prefix once, installs BepInEx 5 (`MORTAR_REGRESS_BEPINEX`) and LethalConfig, ShipLoot and MoreCompany from Thunderstore, hashes the game folder and launches the profile directly through a Proton wrapper set as its launch prefix. It waits for BepInEx's "Chainloader startup complete" (`MORTAR_REGRESS_TIMEOUT`, default 240 s) and requires the three plugins' "Loading" lines. It then stops every process of the sandbox prefix by pid, each verified by its `WINEPREFIX` or `STEAM_COMPAT_DATA_PATH`, waits for Mortar to go idle, re-hashes and requires an empty diff. The Thunderstore index and downloads are cached after the first run in `/var/tmp/mortar-regress-cache` (`MORTAR_REGRESS_CACHE`), so a rerun needs no network; `MORTAR_REGRESS_OFFLINE=1` proves it with a dead proxy. The run also builds the local [mortar-bepinex-bridge](https://github.com/Rethunk-Tech/mortar-bepinex-bridge) (`MORTAR_REGRESS_BRIDGE_REPO`, default the sibling checkout; skipped when absent) and hands it to the server through `MORTAR_LOCAL_BRIDGES`, a folder holding `<game>.zip` that stands in for a release in `loadersvc.ensureBridge`, then requires BepInEx's "Loading [Mortar BepInEx Bridge" and the plugin's "Listening on 127.0.0.1" lines. The shipped catalog has no Lethal Company bridge until the repository publishes a release; then one `components.source.json` entry (game `lethal-company`, name `bridge`, kind `bridge`, asset `MortarBepInExBridge-{version}.zip`, source `github.com/Rethunk-Tech/mortar-bepinex-bridge`) is all it takes. A pass deletes only its own directory; a failure keeps it and prints the path (`LogOutput.log`, `game-before.txt`, `game-after.txt`, `server.log`, `wineboot.log`).

## Release

```sh
wails3 task linux:create:appimage     # bin/mortar-linux-x86_64.AppImage and the portable bin/mortar-linux-amd64
wails3 task linux:nfpm                # .deb, .rpm, Arch package, bin/mortar-aur-linux-amd64
wails3 task linux:flatpak             # bin/mortar-linux-x86_64.flatpak (needs flatpak-builder)
wails3 task linux:build:arm64         # bin/mortar-aur-linux-arm64 and its .deb, .rpm, Arch package
wails3 build GOOS=windows             # bin/mortar.exe (ARCH=arm64 for Windows on ARM; wails3 task windows:package ARCH=arm64 also writes bin/mortar-arm64-installer.exe)
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

`linux:build:arm64` cross-compiles on an x86_64 machine with no emulator registered: it needs `zig`, `docker` (to download the arm64 Ubuntu packages it links against, extracted under `tmp/`), `nfpm` and `qemu-aarch64`, which checks that every shared library resolves. The arm64 AppImage is built only in CI.

`scripts/package-matrix.sh [tag]` downloads a release's Linux assets and, in throwaway containers (podman, else docker), installs and smoke-runs the `.deb` (Debian, Ubuntu), `.rpm` (Fedora), Arch package and AppImage (extract-and-run), then the Flatpak bundle on the host in a temporary `FLATPAK_USER_DIR`. Each case runs `mortar version`, checks the binary, `.desktop`, metainfo, icon and MIME files land, uninstalls and checks nothing is left; it prints a pass/fail table and exits non-zero on any failure. `scripts/windows-installer-smoke.ps1 -Installer <exe>` does the same for the Windows installer (silent install, `mortar version`, Start Menu shortcut, autostart Run key, silent uninstall, leftovers); it needs a Windows machine or the test VM and an x64 installer.

`release:manifest` refuses a `VERSION` other than `build/config.yml`'s `info.version`, needs `bin/mortar-windows-amd64.exe` to exist (copy `bin/mortar.exe` to it first; `release.yml` does that copy) and adds `mortar-windows-arm64.exe` and the arm64 AppImage when present, and writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. The app reads the manifest from the latest release, or the latest pre-release when Settings › Updates includes beta releases; packaged Linux installs leave updating to the package manager. Where the key lives: [docs/architecture.md](docs/architecture.md#release).

### Cutting a release in CI

`.github/workflows/release.yml` runs on a `v*` tag: the gate on the amd64 leg (skipped when `ci.yml` already passed on that commit), then the AppImage, nfpm packages, Flatpak bundle (SDK cached between runs), the Windows exes and per-user NSIS installers for amd64 and arm64 (`bin/mortar-amd64-installer.exe`, `bin/mortar-arm64-installer.exe`, both cross-built on x86_64; the arm64 installer is arm64-native) on x86_64, the AppImage and nfpm packages again on an `ubuntu-24.04-arm` runner, the signed `manifest.json`, the rendered AUR `PKGBUILD` and `SRCINFO` (the release asset for `.SRCINFO`) and Flathub manifest and `SUBMISSION.md`, and the GitHub release with all of those files.

Its notes are user-facing: `build/release/notes.sh` keeps the `feat`, `fix` and `perf` commits since the previous `v*` tag outside internal scopes, in sentence case without scopes, grouped New and Fixed, at most 15 lines each, and links the full changelog.

Releases are immutable once published, so the release is created as a draft, every draft asset is downloaded and checked against the staged files and the manifest's digests (`build/release/verify.sh`), and only then published; a rerun finishes the existing draft. After publishing, `verify.sh --public` fetches the manifest, its assets, the pinned component assets and the newest components manifest without a token: a failure is a warning while a repo is private and fails the run once it is public.

A manual dispatch (Actions › Release › Run workflow) is the dry run: both legs, the signed manifest, package-manager sources, the notes (in the run summary) and `verify.sh --staged` on the staged files, for `build/config.yml`'s version, with no GitHub release.

CI reads the repository secret `MORTAR_UPDATE_KEY`, which holds the private key file's PEM contents, not its path:

```sh
gh secret set MORTAR_UPDATE_KEY --repo Rethunk-Tech/mortar < ~/.config/mortar-release/updater.key
```

An optional organisation secret adds a release output (`gh secret set NAME --org Rethunk-Tech --repos Rethunk-Tech/mortar`); it is skipped with a notice in the run while unset:

- `MORTAR_REPO_GPG_KEY` (the armored private key of "Mortar packages <security@rethunk.tech>", fingerprint `3283604606CAE2295D476F9883BC8751EE6F773D`): after publishing, the `package-repo` job runs `scripts/package-repo.sh` on the release's `.deb`, `.rpm` and Arch packages and uploads the signed repository tree as the `package-repo` artifact and force-pushes it as the single commit of the orphan `packages` branch, which the site's `packages` component serves at https://mortar.rethunk.tech/packages and redeploys on push. `package-repo.yml` (Actions › Package repository) rebuilds it for an existing tag. The tree holds `deb/` (apt-ftparchive `Packages`, `Release`, `InRelease`, `Release.gpg`), `rpm/<arch>/` (createrepo_c metadata with a signed `repomd.xml.asc`; the packages are signed copies of the release assets, signed by `rpmsign` in a throwaway Fedora container, so `mortar.repo` sets both `repo_gpgcheck=1` and `gpgcheck=1`), `arch/<arch>/` (`repo-add` database with package and database `.sig` files), `flatpak/` (the release's Flatpak bundle imported into an OSTree repo with signed commits and summary, plus `tech.rethunk.Mortar.flatpakref` and `mortar.flatpakrepo` carrying the public key and naming Flathub for the runtime), the client snippets `mortar.list`, `mortar.repo` and `pacman.conf`, and the public key as `mortar-archive-keyring.asc` and `.gpg`. The script refuses a key whose fingerprint differs from the committed `build/linux/repo/mortar-archive-keyring.asc`. It holds only the latest release. Locally: `MORTAR_REPO_KEY=<private key file> MORTAR_REPO_URL=<base url> scripts/package-repo.sh <asset dir> <out dir>` (needs podman or docker and gpg).

Per release:

1. Set `info.version` in `build/config.yml`, the only place the version is set, then run `go run ./cmd/version` to copy it into the Windows resources, the Linux metainfo, and the AUR `PKGBUILD` (`main.go` reads it from `build/config.yml` itself). Then run `build/release/notes.sh --metainfo build/linux/tech.rethunk.Mortar.metainfo.xml v1.2.3 HEAD`, which writes the same user-facing notes the release will carry into that version's `<release><description>` (software centres and Flathub show it). Gate (it fails on any copy that differs, and `appstreamcli validate --strict --no-net` checks the metainfo), commit and push `main`.
2. `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal that version with a leading `v`, or the release stops before building.

### Flathub

CI only attaches a single-file `.flatpak` for people who sideload, built by `build/linux/flatpak/tech.rethunk.Mortar.yml` from the CI binary. Listing on Flathub is a separate submission that builds the tagged source. `build/linux/flathub/render.sh OUT_DIR` writes it: `tech.rethunk.Mortar.yml` is the sideload manifest's header and finish-args (the one list of permissions; `render.test.ts` keeps the Flathub template free of its own) above `build/linux/flathub/tech.rethunk.Mortar.yml`'s modules, which build the wails3 CLI, the TypeScript bindings, the frontend (`vite build`; the catalogs, credits and notices are committed) and Mortar with `-X main.packaged=flatpak` inside the GNOME SDK with the Go and Node 24 SDK extensions and no network.

`go-sources.json` lists every proxy.golang.org file a clean build of Mortar and the wails3 CLI reads on amd64 and arm64, served to the build by `GOPROXY=file://`; `node-sources.json` comes from [flatpak-node-generator](https://github.com/flatpak/flatpak-builder-tools/tree/master/node) over `yarn.lock`, bun's lock written in yarn v1 format (the generator reads no bun lock), for yarn's offline mirror. Rendering needs network, Go, bun, jq and `uvx`; `--local` builds this checkout's HEAD instead of the tag, for `flatpak run org.flatpak.Builder` tests (the sandboxed builder sees only home and `/var/tmp`, so test from a clone there). Each release attaches the four files with a `flathub-` prefix and `flathub-SUBMISSION.md` with the PR text:

1. [Flathub app requirements](https://docs.flathub.org/docs/for-app-authors/requirements): AppStream metainfo (`tech.rethunk.Mortar.metainfo.xml`), 128×128 and 256×256 icons, screenshots, AGPL-3.0 license text.
2. Fork [flathub/flathub](https://github.com/flathub/flathub), open a PR that adds `tech.rethunk.Mortar`, then maintain the app repo Flathub creates.
3. Flathub builds from source only ([requirements](https://docs.flathub.org/docs/for-app-authors/requirements): no network during the build, every dependency a manifest source with a public URL), which is what the rendered manifest does.
4. finish-args grant network, Wayland/X11 with ipc, DRI, native and Flatpak Steam libraries, read-only Heroic (`xdg-config/heroic`) and Lutris game configs (`xdg-data/lutris`), read-write Flatpak Lutris and Heroic app data, default `~/GOG Games` and `~/Games/Heroic` install trees, `xdg-download`, removable media (`/run/media`, `/mnt`) for second-drive Steam libraries, the Secret Service (`org.freedesktop.secrets`, Nexus sign-in) and `org.freedesktop.Flatpak` (`flatpak-spawn --host`, which starts Steam and the game outside the sandbox). Launch at login and profile shortcuts go through the Background and DynamicLauncher portals; the game-library paths and `flatpak-spawn` are the linter exceptions `SUBMISSION.md` requests.

### winget

`build/windows/winget/` holds the last submitted manifests for `RethunkTech.Mortar`. After a release publishes, `build/windows/winget/bump.sh 1.2.3 <winget-pkgs checkout>` verifies the release's `SHA256SUMS` signature against the repo key, writes `manifests/r/RethunkTech/Mortar/1.2.3/` with both installers' hashes, and the update goes to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) as a PR (`winget validate` first). Copy the new manifests back over `build/windows/winget/` once it merges. Scoop and the Homebrew tap update themselves from the release ([scoop-bucket](https://github.com/Rethunk-Tech/scoop-bucket), [homebrew-tap](https://github.com/Rethunk-Tech/homebrew-tap)).

### AUR mortar-bin

`build/linux/aur/PKGBUILD` installs `mortar-aur-linux-amd64` or `mortar-aur-linux-arm64` (the packaged build, updater off) from the GitHub release, plus the tagged desktop entry and icon. Publishing:

1. `git clone ssh://aur@aur.archlinux.org/mortar-bin.git`
2. Copy the release's `PKGBUILD` and `SRCINFO` (saved as `.SRCINFO`) in; both carry real sums. The committed `PKGBUILD` keeps `SKIP` sums and its `pkgver` follows `build/config.yml` through `go run ./cmd/version`. If the source archive was not publicly downloadable when the release rendered them, its sum is `SKIP`: run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`.
3. `git add PKGBUILD .SRCINFO && git commit -m "mortar-bin $pkgver" && git push`

## Publishing components

`components.source.json` is the allowlisted input for the signed loader and bridge manifest. The generator resolves its GitHub releases, hashes each asset, refreshes the embedded fallback and verifies a signature when `MORTAR_UPDATE_KEY` names the private key file:

```sh
wails3 task components:refresh
```

The source's `games` array is the game catalog and is copied into the manifest as is: each game's `stores`, `loaders`, `sources`, `metadata`, `paths` and `r2modmanFolder` come from there, and a key the `GameInfo` type does not know is dropped, so add a new field to `internal/components` before using it in the source. Without `MORTAR_UPDATE_KEY` the generator writes an unsigned manifest, so a dry run needs no key: `go run ./cmd/components -output "$T/components.json" -bundled "$T/bundled.json" -signature "$T/sig"` into a temp dir `$T` (it still reads the components' GitHub releases) leaves the repo untouched, and `bundled.json` should equal `internal/components/components.json` except for `serial`. The generator resolves `github.com` components only and the kinds `loader` and `bridge`.

The repository secret `MORTAR_UPDATE_KEY` contains the PEM contents, as for releases. Run the Components workflow manually from Actions, or dispatch it from a component release:

```sh
gh workflow run components.yml --repo Rethunk-Tech/mortar
gh api repos/Rethunk-Tech/mortar/dispatches -f event_type=components
```

The workflow writes the secret to a temporary file, runs the generator with `GOTMPDIR=/var/tmp TMPDIR=/var/tmp`, and creates a `components-<serial>` release (not marked latest) holding `components.json` and `components.json.sig`, each published once because releases are immutable. Until `Rethunk-Tech/mortar-smapi-bridge` is public the generator fails with a 404 looking it up, since the workflow's token reads only this repo.

## Adding a game

A game is catalog data first and code only where its behaviour differs. Each step below names the registry that answers to a catalog reference; `TestEveryCatalogReferenceResolves` (`internal/source/all/catalog_test.go`) fails when an entry names a source, loader, metadata id, store field, path role or path token nothing answers to.

### The catalog entry

Add the game to `games` in `components.source.json`, then publish as described under [Publishing components](#publishing-components), which also covers how `games` reaches the signed manifest and the embedded fallback. `GameInfo.Validate` (`internal/components/components.go`) rejects an entry without an id, name, marker or loader, an enabled game with no store, a marker or GOG folder holding a path separator, a path template that does not start with a token, and a repeated source.

- `id`: the slug; permanent, because profiles, settings and links store it.
- `name`, `enabled`, `marker`: the display name, whether the game is selectable (false lists it as coming later), and a file every install holds, at its root or one `game` folder down. A `.exe` marker makes the build a Windows build.
- `stores`: the game's id in each store that sells it, keyed by the store driver's key: `steam.appId`, `gog.productId` and `gog.folder`, `lutris.slug` and `lutris.keyword`.
- `paths`: folders outside the install by role (`saves`, `errorLogs`, `startupPreferences`), one template per platform. A template starts with a token: `{appData}`, `{localAppData}`, `{localLow}`, `{documents}`, `{xdgConfig}`, `{xdgData}`, `{home}` or `{install}`. The runtime gives each token its folder.
- `sources`: each mod site with `key`, the site's name for the game (Nexus domain, Thunderstore community, a Modrinth facet such as `categories:fabric`, an itch.io search keyword), and `gameId` where the site's API wants a number. Catalog order is the order browse lists them.
- `loaders`: the loader ids the game can run (`smapi`, `bepinex5`; the test's `knownLoaders` list names the ids Mortar has), each with an optional `companion`, the name of the catalog component (kind `bridge`) installed beside it so the running game can be queried.
- `deploy`: `redirect` when the loader points the game at the profile's folder (SMAPI's `--mods-path`), so nothing is placed, or `profile` when the profile holds the loader and its mods (BepInEx), so only the loader's install-side files (the Doorstop proxy) are placed into the install for the launch and removed after.
- `targets`: the places mod files go, each with a profile `root` (`{profileMods}`, `{profile}` or a folder below `{profile}`) and optionally `maxDepth` to cap how deep a file may sit.
- `importIds`: the game's Vortex id (its extension's `GAME_ID`) and MO2 name (its plugin's `GameName`); leave it out for a game neither manager supports.
- `metadata`: provider ids (`smapi-updates`, `smapi-compat`, `stardew-dataset`); none for a game without them.
- `r2modmanFolder`: r2modman's name for the game (the Thunderstore ecosystem schema's `internalFolderName`), which maps an imported r2modman profile to the game.

### A catalog-only game, end to end

A game that runs on a loader Mortar has, is sold by a store Mortar has, and is modded from sources Mortar has needs no Go. The worked example is Lethal Company: Steam, BepInEx 5, Thunderstore, Nexus and GitHub.

1. **Entry.** Add the game to `games` in `components.source.json` with `enabled: false` while it is checked. Lethal Company's entry names `stores.steam.appId`, `loaders: [{id: "bepinex5", companion: "bridge"}]`, `sources` for `thunderstore` (`key` is the community slug), `nexus` (domain `key` and numeric `gameId`) and `github`, `r2modmanFolder`, `paths.saves`, `deploy: "profile"` and the `profile` target.
2. **Companion and loader components.** A loader whose release Mortar installs (SMAPI) or a companion (the bridge) is a `components` entry for the game with a GitHub source, an asset pattern and a version policy; BepInEx ships as a Thunderstore package and needs none.
3. **Manifest.** Regenerate the embedded manifest as described under [Publishing components](#publishing-components).
4. **Check.** `go test ./internal/components ./internal/source/all` runs `GameInfo.Validate` and `TestEveryCatalogReferenceResolves` on the entry.
5. **Try it.** `MORTAR_ENABLE_GAMES=<id>` (the self-test sets it) enables a game the shipped catalog lists as disabled. `scripts/selftest.sh seed` fills the sandbox with a profile for it when the game is installed there; browse, install, a profile and its Problems tab are then exercised in the sandbox's browser.
6. **Enable.** Flip `enabled` to true and publish the manifest.

### Drivers

A new store, site, runtime, provider, host, installer, deployer, framework or loader is one Go package or file plus its registration; a game that uses only existing ones needs none of this.

1. **Store** (`internal/gamestore`): implement `Store` (`Key`, `Launchers`, `Discover`) in `drivers.go`, add it to `All()`, and give `components.GameStores` the matching field. `Key` is the name used in the catalog's `stores`.
2. **Source** (`internal/source`): implement `Source` (`ID`, `Name`, `Modes`) in `internal/source/<id>/`, call `source.Register` in a package variable, and import the package in `internal/source/all/all.go`. Add the capabilities the site supports, each found by type assertion: `Searcher` (browse), `Categorizer` (category names that a query's include and exclude lists match), `Gated` (`Unavailable()` says what the player must set up first, such as itch.io's API key; browse greys the source out with that reason), `Schemer` (a URL scheme the site's links open Mortar with), `PageLinker` (`ModPageURL(gameKey, id)`), `Hoster` (web hosts, so a bare URL traces back to the site), and `LinkOptIn` (a scheme claimed from the system only when the player turns it on; Thunderstore's `ror2mm` is off by default). A searcher sets `Item.Adult` from the site's flag, since browse hides adult hits unless the player opts in. Tests serve the site from a fake HTTP server.
3. **Runtime** (`internal/runtime`): implement `Runtime` (`ID`, `Detect`, `Resolve`) and add it to `drivers` before `native`, which claims whatever nothing else does. `Resolve` expands a catalog path template for the install.
4. **Metadata provider** (`internal/metadata`): add the capability (`Updates`, `Status` or `Dataset`) and the case for its id in `For`.
5. **Host** (`internal/host`): implement `Host` (`ID`, `Match`, `Fetch`) and add it to `For` ahead of the direct host, which takes any HTTPS URL.
6. **Pack format** (`internal/pack`): implement `Format` so another manager's list reads into a `Draft` and pass it to `pack.Read` where the importer lists its formats; a format writes nothing.
7. **Installer** (`internal/installer`): implement `Installer` (`ID`, `Detect`, `Layout`) for an archive shape, call `installer.Register`, and add its id to `detectionOrder` before `plain`, which takes any archive. `Layout` returns files with a catalog target and a path inside it; every path is validated to stay inside its target.
8. **Deployer** (`internal/deploy`): implement `Deployer` (`Plan`, `Apply`, `Purge`, `Recover`) and call `deploy.Register`; the catalog's `deploy` names the id.
9. **Framework** (`internal/framework`): implement `Framework` (`ID`, `Matches`, `Analyze`) in a package that calls `framework.Register`; its analyzers run when an enabled mod matches. Content Patcher is the one driver.
10. **Loader** (`internal/loader`): implement `Loader` (`ID`, `Formats`, `Status`, `Install`, `Contribute`) in `internal/loader/<id>/`, call `loader.Register`, and import the package where the other loaders are imported. Add the capabilities it supports, found by type assertion: `Vanilla` (a launch without mods), `Owner` (which process is this loader's), `WithLogs` (the loader's log, its readiness line and analyzers), `WithConfig` (writable settings folders), `WithLaunchSettings` (launch options the loader keeps in the profile's files, shown in Edit profile), `WithPlayerLog` (the Unity player log its analyzers also read), `WithCompanion` (the bridge mod's folder, id and state file), `Console`, `Releases`, `ProcessNames` and `SteamExe`. A loader that redirects the game to the profile (SMAPI) declares no install-side files; one that needs files beside the executable (BepInEx's Doorstop `winhttp.dll` and `doorstop_config.ini`) declares them in its plan and the deployer places them.

### Self-test

```sh
go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime
scripts/selftest.sh start
```

`scripts/selftest.sh` serves a sandboxed Mortar in a browser at `http://127.0.0.1:$PORT` (default 9455), with a minimal Steam library holding copies of Stardew Valley and, when installed, Lethal Company; `setup` creates the library only. Add the game's app id and folder to the script to put a copy of a new game in the sandbox.
