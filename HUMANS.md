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
bun run --cwd frontend e2e                    # UI smoke against the running self-test server
```

- `selftest` (`scripts/selftest.sh`) runs a `server`-tagged binary with its own HOME, copied (never linked) game folders and a minimal Steam library, so real data, game and Steam config are never touched. `--copy-data` copies live profiles and settings in once (`MORTAR_SELFTEST_DATA` names another Mortar data folder). The sandbox lives under `/var/tmp/mortar-selftest` unless `MORTAR_SELFTEST_DIR` is set (`MORTAR_SELFTEST_BASE` defaults to `/var/tmp`; `MORTAR_SELFTEST_PORT` to 9455; `MORTAR_SELFTEST_STEAM` to the host Steam library; `MORTAR_SELFTEST_ENABLE` overrides the games `MORTAR_ENABLE_GAMES` turns on, default `lethal-company`). `stop` leaves the folder; `destroy` deletes it, and `reap --yes` deletes idle marked sandboxes under the base dir.
- `e2e` (Playwright, Chromium only) opens every profile tab and Settings page of the running server and fails on a console error, a failed request or a main-thread task over 200 ms. It skips itself when no server is running, so it is in neither the gate nor CI; run it after each wave of UI changes (`restart` first).
- The frontend build first runs `scripts/gen-credits.ts` (rewrites `frontend/src/settings/generated/credits.json` from `frontend/package.json` and `go.mod`), then extracts and compiles the Lingui catalogs in `frontend/src/locales/`. Commit those files when they change.

## Command line

With Mortar running, `mortar <command>` asks the open app and prints a table, or JSON with `--json`. Output flags (`--json`, `-v`) may precede the verb (`mortar --json install ...`). `mortar help` lists every command; the source is `internal/cli/cli.go`. `<profile>` is a profile id or name; a mod id is `<format>:<local id>` (a bare id uses the game's own format, SMAPI's UniqueID for Stardew).

| Area | Commands |
| --- | --- |
| Games, profiles | `games`; `game steam-launch-option\|launch-preset-templates\|reset-install\|steam-status`; `profiles <game>`; `profile create\|from-save\|rename\|copy\|delete\|repair\|set\|shortcut\|steam`; `profile list <game> <profile> --format md\|text`; `profile compare <game> <A> <B>`; `profile match <game> <profile> <link-or-file>`; `profile collection <game> <profile> [--update\|--unlink]`; `profile load-order\|health\|changes\|good\|backup\|restore`; `profile import`; `profile export <game> <profile> <file.zip> --format modpack`; `profile export <game> <profile> <file.r2z> --format r2z`; `profile farm export\|check\|fix`; `loader versions\|install\|pin`; `trash list [--game stardew]`, `trash restore\|delete\|empty ...` (`delete`, `empty` need `--yes`); `templates list\|save\|delete\|new\|apply\|undo\|rename\|restore` |
| History | `profile history <game> <profile>`; `profile revert <game> <profile> <eventId>`; `history <game> --all`; `history diff <game> <profile> <a> <b>`; `history revert <game> <profile> <eventId> --item <mod>`; `history usage <game>`; `history trim <game> <profile> --keep N` |
| Play | `play <game> <profile> --check` (pre-Play summary; exits 3 on any issue); `play <game> <profile> --test`; `launch <game> <profile> [--install <id>] [--preset NAME] [--wait] [--force]`; `launch <game> --vanilla`; `status <game>`; `stop <game> [--install <id>]`; `runs`; `logs`; `logs share`; `logs search <query> [--profile <name>]`; `logs fixes`; `sweep <game>` |
| Mods | `mods <game> <profile>`; `mods enable\|disable\|pin\|unpin\|remove ... <mod id>...`; `mod ... <mod id>`; `mods tag\|untag\|category\|note\|skip-version\|channel\|split\|combine\|win\|files\|config\|preset\|menu\|compat\|report\|by-author`; `mods group <game> <profile> list\|create\|rename\|delete\|add\|remove\|on\|off …`; `update <game> <profile> <mod id>...\|--all`; `install <game> <profile> <archive> [--allow-unscanned]` (installs although the antivirus flagged it, recorded in history); `source untrack <game> --source nexus --all\|--unused` (`--yes` skips the prompt); `source tracked <game> <profile> --source nexus --missing`; `browse <game> <text> [--source <id>\|all]` |
| Saved groups | `groups <game>`; `groups apply <game> <saved group> <profile>`; `groups create <game> <name> <profile> [mod id...]`; `groups delete <game> <saved group>`; `groups rename <game> <saved group> <name>`; `groups add <game> <saved group> <profile> <mod id>...`; `groups remove <game> <saved group> <mod id>...` |
| Tabs | `conflicts`; `conflicts map`; `who`; `problems [--format text]`; `problems dismissed`, `problems dismiss <index>`, `problems restore <token\|index>` (`--profile`, `--game stardew`); `problems check-updates`; `updates`; `updates apply --everywhere`; `saves <game> <profile>`; `saves check`; `perf reports <game> <profile>` |
| Sharing | `share`; `export <game> <profile> <file.mortar>`; `open <link\|file>` (share links and `.mortar` files); `lan peers\|inbox\|send\|accept\|decline\|pair\|paired\|unpair` |
| Settings, tools | `settings get\|set [--game id]`; `settings export\|import <file>`; `settings reset [key] [--game id]`; `launchers`, `launchers add\|remove <id> <folder>`; `tools <game>`, `tools run\|add\|update\|remove`; `antivirus status` (scanner in use: `amsi`, `clamd`, `command`, `off`); `links register`; `links enable\|disable --source nexus\|thunderstore\|curseforge`; `app update check\|install`; `support diagnostics save` |
| Downloads, backups | `queue`, `queue add`, `queue retry\|skip [id]`, `queue pause\|resume\|clear`, `queue pause-item\|resume-item\|top ID`, `queue retry-failed`; `backups list`, `backups create <save>` (pins a Manual backup, by folder or app-shown name), `backups keep\|unkeep`, `backups restore <name> [save...]`, `backups usage`, `backups trim --keep N` (newest N per save; pinned stay) |
| Library, data | `library extra <game>`; `library hidden <game> <profile>`; `library old-files <game> <profile> [--keep\|--delete KEY]`; `library strays <game> [<profile> --move FOLDER...] [--dismiss FOLDER]`; `store report\|remove\|check\|repair`; `bisect start\|status\|stop`; `cache size\|clear`; `data location`; `data usage --by-mod`; `data move\|cleanup [--preview]`; `archive preview <path>`; `archive downloads <game>` |
| Environment | `doctor`; `quit [--force]`; `version`; `completion bash\|zsh\|fish` |

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
scripts/selftest.sh regress     # about 40 s warm, needs the real Stardew install and a display
```

In a throwaway `/var/tmp/mortar-regress-XXXXXX` sandbox on a free port it builds under the shared flock, copies the real data, sets the launch method to direct, hashes the game folder and launches the first profile through the CLI (`MORTAR_REGRESS_PROFILE` picks another). It waits for SMAPI's "Loaded N mods" and "Loaded M content packs" (`MORTAR_REGRESS_TIMEOUT`, default 180 s) and requires N + M plus the mods SMAPI names as skipped to equal the profile's enabled count. It then stops the game by its exe-verified pid, waits for Mortar to go idle, re-hashes and requires an empty diff. A pass deletes its directory; a failure keeps it (`game-before.txt`, `game-after.txt`, `server.log`) and exits non-zero.

### Lethal Company regression run (opt-in, not in the gate)

```sh
scripts/selftest.sh regress --game lethal-company     # about 50 s, needs Lethal Company and Proton - Experimental in the real Steam library
```

Same shape, in `/var/tmp/mortar-regress-lc-XXXXXX`:

- **Setup:** copies the game, Steam's `steamclient.so` files and `.steam` links into the sandbox home (Proton needs the library, not a running Steam; Steam is never started). Every Proton process runs under `bwrap --unshare-net`: a copy registered under its Wine pid in a running Steam's loopback IPC leaves Steam on "Waiting for Lethal Company to shut down" for good. It creates the Proton prefix once, installs BepInEx 5 (`MORTAR_REGRESS_BEPINEX`) plus LethalConfig, ShipLoot and MoreCompany from Thunderstore, and launches the profile through a Proton wrapper set as its launch prefix (`MORTAR_REGRESS_PROTON` names the Proton folder; `MORTAR_REGRESS_MORTAR_PROTON=1` runs the copy as a folder Steam did not install, through Mortar's own Proton runtime).
- **Checks:** BepInEx's "Chainloader startup complete" (`MORTAR_REGRESS_TIMEOUT`, default 240 s), the three plugins' "Loading" lines and, for the bridge, "Loading [Mortar BepInEx Bridge" and "Listening on 127.0.0.1". It stops the game with `mortar stop`, reaps what is left of the sandbox prefix by pid (each verified by its `WINEPREFIX` or `STEAM_COMPAT_DATA_PATH`), waits for idle, re-hashes and requires an empty diff.
- **Cache:** the Thunderstore index and downloads live in `/var/tmp/mortar-regress-cache` (`MORTAR_REGRESS_CACHE`); `MORTAR_REGRESS_OFFLINE=1` proves a rerun needs no network, with a dead proxy.
- **Bridge:** profiles get the catalog's published [mortar-bepinex-bridge](https://github.com/Rethunk-Tech/mortar-bepinex-bridge) release, which an offline run cannot download and so skips the bridge check. `MORTAR_REGRESS_LOCAL_BRIDGE=1` builds the checkout beside this repo instead (`MORTAR_REGRESS_BRIDGE_REPO=<checkout>` names another and implies it), handed over through `MORTAR_LOCAL_BRIDGES` (a folder holding `<game>.zip`, standing in for the release in `loadersvc.ensureBridge`). The summary says `published` or `local`.
- **Launches** run on the sandbox's hidden display under the session's launch cap ([docs/testing.md](docs/testing.md#sandbox-game-launches)).
- **r2modman import:** `MORTAR_REGRESS_R2_CODE=<key>` adds a second profile imported with `mortar profile import --game lethal-company`. Its downloads need the network (no archive cache). BepInEx must count as many plugins as it logs "Loading [" lines, and the purge must again leave the game folder identical. `MORTAR_REGRESS_R2_DISABLE` lists package ids (`Namespace-Name`) to switch off first. `MORTAR_REGRESS_R2_EXPORT=1` publishes an r2modman code to thunderstore.io (refused together with the matrix, which uses the session's remaining launches).
- **Trap:** a mod that writes into the game folder at runtime fails the hash (Mortar purges only what it placed). RugbugRedfern-Skinwalkers creates an empty `Dissonance_Diagnostics` folder, so `019c9378-d2e6-4fec-5a2f-e4b22f8cf1d1` passes with `MORTAR_REGRESS_R2_DISABLE=RugbugRedfern-Skinwalkers`.
- A pass deletes its directory; a failure keeps it and prints the path (`LogOutput-<step>.log`, `game-before.txt`, `game-after-<step>.txt`, `server.log`, `wineboot.log`, `r2-*.json`).

## Release

```sh
wails3 task linux:create:appimage     # bin/mortar-linux-x86_64.AppImage and the portable bin/mortar-linux-amd64
wails3 task linux:nfpm                # .deb, .rpm, Arch package, bin/mortar-aur-linux-amd64
wails3 task linux:flatpak             # bin/mortar-linux-x86_64.flatpak (needs flatpak-builder)
wails3 task linux:flatpak:arm64       # bin/mortar-linux-aarch64.flatpak, on x86_64
wails3 task linux:build:arm64         # bin/mortar-aur-linux-arm64 and its .deb, .rpm, Arch package
wails3 build GOOS=windows             # bin/mortar.exe (ARCH=arm64 for Windows on ARM; wails3 task windows:package ARCH=arm64 also writes bin/mortar-arm64-installer.exe)
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

- **AppImage:** linuxdeploy and the type2 runtime are pinned with SHA-256 in `build/linux/appimage/pins.txt`, kept in `$XDG_CACHE_HOME/mortar-build` (`MORTAR_BUILD_CACHE` overrides; CI caches `~/.cache/mortar-build`) and checked on every build.
- **`linux:build:arm64`** cross-compiles on x86_64 with no emulator registered. It needs `zig`, `docker` (fetches the arm64 Ubuntu packages it links against, extracted under `tmp/`), `nfpm` and `qemu-aarch64` (checks that every shared library resolves). The arm64 AppImage is built only in CI.
- **`linux:flatpak:arm64`** (`build/linux/flatpak/build-aarch64.sh`) runs that cross build with `linux:flatpak ARCH=arm64`. `flatpak-builder` runs the manifest's `build-commands` through the aarch64 SDK's `/bin/sh`, so the script registers `qemu-aarch64-static` in a user namespace's own `binfmt_misc` (Linux 6.7 or newer; no root). It needs the build:arm64 tools plus `flatpak`, and uses the host's `qemu-aarch64-static`, `flatpak-builder` and `appstreamcli-compose`, unpacking missing ones from Fedora's packages into `tmp/rpm/` (`dnf download`, no install). The aarch64 GNOME runtime and SDK go into the user Flatpak installation. CI builds the release bundle natively on the arm64 leg.
- **`scripts/package-matrix.sh [tag]`** downloads a release's Linux assets and, in throwaway containers (podman, else docker), installs and smoke-runs the `.deb` (Debian, Ubuntu), `.rpm` (Fedora), Arch package and AppImage (extract-and-run), then the Flatpak bundle on the host in a temporary `FLATPAK_USER_DIR`. Each case runs `mortar version`, checks the binary, `.desktop`, metainfo, icon and MIME files land, uninstalls and checks nothing is left; it prints a pass/fail table and exits non-zero on failure.
- **`scripts/windows-installer-smoke.ps1 -Installer <exe>`** does the same for the Windows installer (silent install, `mortar version`, Start Menu shortcut, autostart Run key, silent uninstall, leftovers); it needs a Windows machine or the test VM and an x64 installer.
- **`release:manifest`** refuses a `VERSION` other than `build/config.yml`'s `info.version`, needs `bin/mortar-windows-amd64.exe` (copy `bin/mortar.exe` to it; `release.yml` does), adds `mortar-windows-arm64.exe` and the arm64 AppImage when present, writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. The app reads the manifest from the latest release, or the latest pre-release when Settings › Updates includes betas. Where the key lives: [docs/architecture.md](docs/architecture.md#release).

### Cutting a release in CI

`.github/workflows/release.yml` runs on a `v*` tag:

1. The gate on the amd64 leg (skipped when `ci.yml` already passed on that commit).
2. On x86_64: the AppImage, nfpm packages, Flatpak bundle (SDK cached between runs), and the Windows exes and per-user NSIS installers for amd64 and arm64 (`bin/mortar-amd64-installer.exe`, `bin/mortar-arm64-installer.exe`, both cross-built; the arm64 installer is arm64-native). On an `ubuntu-24.04-arm` runner: the AppImage and nfpm packages again.
3. The signed `manifest.json`, the rendered AUR `PKGBUILD` and `SRCINFO` (the release asset for `.SRCINFO`), the Flathub manifest and `SUBMISSION.md`, and the GitHub release with all of them.

- **Notes** are user-facing: `build/release/notes.sh` keeps `feat`, `fix` and `perf` commits since the previous `v*` tag outside internal scopes, in sentence case, grouped New and Fixed by area (games, browsing, profiles and sharing, launching, problems, downloads, saves, interface), at most 15 lines each, with a changelog link.
- **Immutable releases:** the release is created as a draft, every draft asset is downloaded and checked against the staged files and the manifest's digests (`build/release/verify.sh`), and only then published; a rerun finishes the existing draft. After publishing, `build/release/verify.sh --public` fetches the manifest, its assets, the pinned component assets and the newest components manifest without a token: a warning while the repo is private, a failure once public.
- **Dry run:** a manual dispatch (Actions › Release › Run workflow) runs both legs, the signed manifest, package-manager sources, the notes (in the run summary) and `build/release/verify.sh --staged` for `build/config.yml`'s version, with no GitHub release and no attestation. A tag run reads the caches a dispatch on `main` saved but cannot share its own, so the dry run warms them for the tag.
- **Secret `MORTAR_UPDATE_KEY`** holds the private key file's PEM contents, not its path:

  ```sh
  gh secret set MORTAR_UPDATE_KEY --repo Rethunk-Tech/mortar < ~/.config/mortar-release/updater.key
  ```

- **Optional org secret** `MORTAR_REPO_GPG_KEY` (`gh secret set MORTAR_REPO_GPG_KEY --org Rethunk-Tech --repos Rethunk-Tech/mortar`), the armored private key of "Mortar packages <opensource@rethunk.tech>" (fingerprint `D8B1C4C205C5FB3633CB8B9900B75959B637DCB5`); the output is skipped with a run notice while unset. After publishing, the `package-repo` job runs `scripts/package-repo.sh` on the release's `.deb`, `.rpm` and Arch packages, uploads the signed tree as the `package-repo` artifact and force-pushes it, under `packages/` beside the tag's `site/`, as the single commit of the orphan `packages` branch. App Platform deploys both components from that branch on push (site at <https://mortar.rethunk.tech>, repository at /packages), so the site changes only with a release. `package-repo.yml` (Actions › Package repository) rebuilds both for an existing tag.
  - The tree holds `deb/` (apt-ftparchive `Packages`, `Release`, `InRelease`, `Release.gpg`), `rpm/<arch>/` (createrepo_c metadata with a signed `repomd.xml.asc`; packages are signed copies of the release assets via `rpmsign` in a throwaway Fedora container, so `mortar.repo` sets both `repo_gpgcheck=1` and `gpgcheck=1`), `arch/<arch>/` (`repo-add` database, package and database `.sig`), `flatpak/` (the bundle in an OSTree repo with signed commits and summary, plus `tech.rethunk.Mortar.flatpakref` and `mortar.flatpakrepo`), the client snippets `mortar.list`, `mortar.repo`, `pacman.conf`, and the public key as `mortar-archive-keyring.asc` and `.gpg`.
  - The script refuses a key whose fingerprint differs from the committed `build/linux/repo/mortar-archive-keyring.asc` and holds only the latest release. Locally (needs podman or docker, and gpg): `MORTAR_REPO_KEY=<private key file> MORTAR_REPO_URL=<base url> scripts/package-repo.sh <asset dir> <out dir>`.

Per release:

1. Set `info.version` in `build/config.yml` (the only place), then `go run ./cmd/version` copies it into the Windows resources, the Linux metainfo and the AUR `PKGBUILD` (`main.go` reads `build/config.yml` itself). Run `build/release/notes.sh --metainfo build/linux/tech.rethunk.Mortar.metainfo.xml v1.2.3 HEAD` to write the release's user-facing notes into that version's `<release><description>` (software centres and Flathub show it). Gate (it fails on any differing copy, and `appstreamcli validate --strict --no-net` checks the metainfo), commit and push `main`.
2. `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal that version with a leading `v`, or the release stops before building.

### Flathub

CI attaches a single-file `.flatpak` for sideloading, built by `build/linux/flatpak/tech.rethunk.Mortar.yml` from the CI binary; users get the Flatpak from Mortar's own repo at mortar.rethunk.tech/packages until Flathub approves it. The Flathub listing is a pending, separate submission that builds the tagged source.

- `build/linux/flathub/render.sh OUT_DIR` writes it. `tech.rethunk.Mortar.yml` is the sideload manifest's header and finish-args (the one list of permissions; `build/linux/flathub/render.test.ts` keeps the Flathub template free of its own) above `build/linux/flathub/tech.rethunk.Mortar.yml`'s modules, which build the wails3 CLI, the TypeScript bindings, the frontend (`vite build`; catalogs, credits and notices are committed) and Mortar with `-X main.packaged=flatpak`, inside the GNOME SDK with the Go and Node 24 SDK extensions and no network.
- `go-sources.json` lists every proxy.golang.org file a clean amd64 and arm64 build reads, served by `GOPROXY=file://`. `node-sources.json` comes from [flatpak-node-generator](https://github.com/flatpak/flatpak-builder-tools/tree/master/node) over `yarn.lock` (bun's lock in yarn v1 format; the generator reads no bun lock).
- Rendering needs network, Go, bun, jq and `uvx`. `--local` builds this checkout's HEAD instead of the tag, for `flatpak run org.flatpak.Builder` tests; the builder sees only home and `/var/tmp`, so test from a clone there.
- Each release attaches the four files with a `flathub-` prefix and `flathub-SUBMISSION.md` (the PR text). To submit: meet the [Flathub app requirements](https://docs.flathub.org/docs/for-app-authors/requirements) (AppStream metainfo `tech.rethunk.Mortar.metainfo.xml`, 128×128 and 256×256 icons, screenshots, AGPL-3.0 license text; builds from source with no network, every dependency a manifest source with a public URL), fork [flathub/flathub](https://github.com/flathub/flathub), open a PR adding `tech.rethunk.Mortar`, then maintain the app repo Flathub creates.
- finish-args grant: network, Wayland/X11 with ipc, DRI, native and Flatpak Steam libraries, read-only Heroic (`xdg-config/heroic`) and Lutris (`xdg-data/lutris`) configs, read-write Flatpak Lutris and Heroic app data, `~/GOG Games`, `~/Games/Heroic`, `xdg-download`, `/run/media` and `/mnt` (second-drive Steam libraries), and `org.freedesktop.Flatpak` (`flatpak-spawn --host` starts Steam and the game outside the sandbox). Launch at login and profile shortcuts use the Background and DynamicLauncher portals. The game-library paths and `flatpak-spawn` are the linter exceptions `SUBMISSION.md` requests.

### winget

`build/windows/winget/` holds the last submitted manifests for `RethunkTech.Mortar`. After a release publishes, `build/windows/winget/bump.sh 1.2.3 <winget-pkgs checkout>` verifies the release's `SHA256SUMS` signature against the repo key and writes `manifests/r/RethunkTech/Mortar/1.2.3/` with both installers' hashes. Run `winget validate`, send the update to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) as a PR, and copy the new manifests back over `build/windows/winget/` once it merges. Scoop and the Homebrew tap update themselves from the release ([scoop-bucket](https://github.com/Rethunk-Tech/scoop-bucket), [homebrew-tap](https://github.com/Rethunk-Tech/homebrew-tap)).

### AUR mortar-bin

`build/linux/aur/PKGBUILD` installs `mortar-aur-linux-amd64` or `mortar-aur-linux-arm64` (the packaged build, updater off) from the GitHub release, plus the tagged desktop entry and icon. Publishing:

1. `git clone ssh://aur@aur.archlinux.org/mortar-bin.git`
2. Copy the release's `PKGBUILD` and `SRCINFO` (saved as `.SRCINFO`) in; both carry real sums. The committed `PKGBUILD` keeps `SKIP` sums and its `pkgver` follows `build/config.yml` through `go run ./cmd/version`. If the source archive was not publicly downloadable when the release rendered them, its sum is `SKIP`: run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`.
3. `git add PKGBUILD .SRCINFO && git commit -m "mortar-bin $pkgver" && git push`

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

- The workflow writes the secret to a temporary file, runs the generator with `GOTMPDIR=/var/tmp TMPDIR=/var/tmp`, and creates a `components-v2-<serial>` release (not marked latest) holding `components.json` and `components.json.sig`, each published once (releases are immutable). Until `Rethunk-Tech/mortar-smapi-bridge` is public the generator fails with a 404 looking it up, since the workflow's token reads only this repo.

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
| `stores` | the game's id per store driver key: `steam.appId`, `gog.productId`, `gog.folder`, `lutris.slug`, `lutris.keyword`, `bottles.folder`, `ea.folder`, `ea.registry` (the HKLM key whose "Install Dir" the installer writes, read on Windows to find a game on another drive; Vortex reads The Sims 4 the same way), `epic.appNames` (one per product sold), `xbox.package` |
| `templates` | starter profiles for "Starter templates" in the New profile dialog: `id`, `name`, `description`, `packages` (`source`, `ref` (Nexus mod id, GitHub `owner/repo` or Thunderstore `Namespace-Name`), `name`) |
| `paths` | folders and files outside the install, by role, one template per platform; each starts with a token: `{appData}`, `{localAppData}`, `{localLow}`, `{documents}`, `{xdgConfig}`, `{xdgData}`, `{home}` or `{install}`. Roles: `saves`, `errorLogs`, `startupPreferences`; `mods` (a `folder` game's mod folder, filled for a launch, then emptied with the player's files returned); `options` (one settings file; each profile holds its own copy, seeded on first launch and swapped in per launch, `requiredSettings` written into that copy or only reported per the game's settings mode); `tray` (shared by every profile, filled by a target with `role: "tray"`); `userData` (the base `caches` paths resolve under). Any other role a `requiredSettings` or `caches` entry names must be listed too |
| `sources` | each mod site: `key` (the site's name for the game: Nexus domain, Thunderstore community, Modrinth facet such as `categories:fabric`, CurseForge's Mods class id, itch.io search keyword) and `gameId` where the API wants a number. CurseForge may list `classes` (numeric ids a search covers; must contain `key`; merged in the player's sort order). A link and handoff source (`patreon`, `itch`) is just `{"id": "..."}` with no `key`: browse does not search it, a pasted address resolves to it and Mortar opens the page. Catalog order is browse order |
| `loaders` | ids the game can run (`smapi`, `bepinex5`, `melonloader`, `returnofmodding`, `gdweave`, `lovely`, `northstar`, `rivet`, `umm`, `shimloader`, `folder` for a game with none; the test's `knownLoaders` names them), each with optional `companion` (a catalog component of kind `bridge`) and, for BepInEx, `package` (a community's own pack, e.g. `denikson-BepInExPack_Valheim`; default `BepInEx-BepInExPack`) and `major` (6 for IL2CPP on `BepInEx-BepInExPack_IL2CPP`, as Cat Mail Co.; 5 when absent) and `proxy` (the pack's Doorstop proxy when it is not `winhttp.dll`: `version.dll` for In Silence; the doctor's Proton check and a launch before the profile holds a pack read it). BepInEx also takes `routes`, from the game's r2modman `installRules` beyond the standard ones: `dir` (a profile-root folder), `extensions` (files ending in one go there wherever they sit) and `track` (`subdir` keeps a package's files below a folder named for it, `subdir-no-flatten` also keeps their path from the package's top) and, for MelonLoader, `default` (the route that takes a file no folder or extension claims); a package folder named like `dir`'s last segment goes there too, so keep r2modman's rule order. A MelonLoader entry with no `routes` takes 0.7's recursive layout, and its `package` is `LavaGang-MelonLoader` so a mod's dependency on it is the loader's own. A `returnofmodding`, `gdweave`, `lovely`, `northstar`, `rivet`, `umm` or `shimloader` entry names its Thunderstore `package` (`Hell2Modding-Hell2Modding`, `NotNet-GDWeave`, `Thunderstore-lovely`, `northstar-Northstar`, `ReDoIngMods-Rivet`, `UMM-UMM`, `Thunderstore-unreal_shimloader`), which the loader installs from the game's community; Northstar also takes `routes` (`R2Northstar/mods`); Shimloader takes `routes` (r2modman's `shimloader/mod`, `pak`, `cfg`, `overlay` rules) and `placeDir` (the slash path below the install of the folder its files are placed in, `<data folder>/Binaries/Win64`, when that is not `markerDir`). A bridge installs only from the game's own `kind: bridge` component (Lethal Company's and Valheim's own builds, and the core build for REPO, PEAK, Risk of Rain 2 and Content Warning), taken from the bridge's GitHub release, never Thunderstore, so a game without one gets no startup timing, in-game performance, stream overlay, intro skip or console commands even when its loader names a `companion` |
| `deploy` | `redirect` when the loader points the game at the profile's folder (SMAPI's `--mods-path`), or `profile` when the profile holds the loader and its mods (BepInEx; only the Doorstop proxy is placed into the install for a launch) |
| `targets` | where mod files go: a profile `root` (`{profileMods}`, `{profile}` or a folder below it) and optional `maxDepth`. `keepWhole` (`folder` games) lists extensions that make an archive one entry; an archive with none splits per file. A target with `role` instead of `root` places files at install into that path role's folder, shared by every profile; `extensions` routes file types there |
| `importIds` | Vortex id (the extension's `GAME_ID`) and MO2 name (the plugin's `GameName`); omit when neither supports it |
| `metadata` | provider ids (`smapi-updates`, `smapi-compat`, `stardew-dataset`) |
| `saveFiles`, `saveCompanions`, `saveCopies`, `saveCopyFolders`, `saveLinked` | for saves kept as files: name patterns in the saves folder or one below (`!` first excludes), same-stem companion extensions, and patterns for the game's own copies beside a save (`{name}`, `{stem}`, `{ext}`) that backups list and restore. `saveCopyFolders` are folders beside the saves folder that also hold those copies; `saveLinked` are slash paths beside the saves folder of files that belong to one save (`{uuid}` is the world's id). Without `saveFiles` a save is a folder holding a file of its own name |
| `folderNames` | other names a `paths` folder goes by on some installs (The Sims 4's user folder follows the game's language); the one that exists and changed last wins |
| `requiredSettings` | `{path, section, key, value, message}` in the game's own `key = value` files; `path` is a path role, `section` optional. Problems reports another value; Mortar writes only into a profile's own copy of an `options` file |
| `newProfileSeparateSaves` | a new profile starts with its own saves folder |
| `caches` | `{role, path}` rebuildable files the game recreates; deleted once on the first launch after a profile's mod set changes |
| `knownBroken` | curated mods or Thunderstore packages that do not work: `id`, optional `versions` (space-separated constraints), `reason`, optional `replacement` |
| `titleScene` | Unity scene of the main menu, where a BepInEx startup measurement ends; without it the first scene ends it |
| `graphics` | player choice of graphics API (`explanation`, `choices` with `id`/`label`/`args`, `recommended`, `reason`, `source`) |
| `r2modmanFolder` | r2modman's name for the game (the Thunderstore schema's `internalFolderName`) |

The Sims 4 (`sims4` in `components.source.json`) is the worked `folder` game: `loaders` `[{"id": "folder"}]`, `deploy` `profile`, `marker` `TS4_x64.exe` with `markerDir` `Game/Bin`, `paths` for `mods`, `saves`, `options`, `tray` and `userData` under `{documents}/Electronic Arts/The Sims 4`, a `mods` target at `{profile}/Mods` with `maxDepth` for `package` and `ts4script` and `keepWhole` `["ts4script"]` (a script mod stays whole, a package-only archive splits per file), a `tray` target with `role` `tray`, `requiredSettings` for the two `Options.ini` switches, `caches` for the thumbnail and avatar caches, `newProfileSeparateSaves`, and `sources` CurseForge (with `classes`), Nexus, GitHub, `itch` and `patreon`.

### A catalog-only game, end to end

A game on a loader, store and sources Mortar already has needs no Go: `game.Find` answers for every catalog entry, and only a game the entry cannot describe is a type in `internal/game` listed in `coded` (Stardew Valley, for the process names SMAPI's install leaves). Lethal Company is the worked example (Steam, BepInEx 5, Thunderstore, Nexus, CurseForge, GitHub).

1. **Entry.** Add it to `games` with `enabled: false` while checking. Lethal Company's names `stores.steam.appId`, `loaders: [{id: "bepinex5", companion: "bridge"}]`, `sources` for `thunderstore` (`key` is the community slug), `nexus` (domain `key` and numeric `gameId`) and `github`, `r2modmanFolder`, `paths.saves`, `deploy: "profile"` and the `profile` target.
2. **Components.** A loader whose release Mortar installs (SMAPI) or a companion (the bridge) is a `components` entry with a GitHub source, an asset pattern and a version policy. BepInEx ships as a Thunderstore package and needs none.
3. **Manifest.** Regenerate the embedded manifest ([Publishing components](#publishing-components)).
4. **Check.** `go test ./internal/components ./internal/source/all` runs `GameInfo.Validate` and `TestEveryCatalogReferenceResolves`.
5. **Try it.** `TestACatalogOnlyGameNeedsNoCode` (`internal/game`: a test-only Content Warning entry is selectable, found on Steam and given BepInEx) and `TestACatalogOnlyGameBrowsesAndAdds` (`internal/browse`) are the pattern: put the entry in a test manifest selected with `components.Use` (never the shipped catalog). In the sandbox, `MORTAR_ENABLE_GAMES=<id>` (the self-test sets `lethal-company`; `MORTAR_SELFTEST_ENABLE` overrides) switches on a game the bundled manifest lists as disabled, so the entry must be in `components.source.json` and the regenerated bundled manifest first. `scripts/selftest.sh` copies Stardew Valley, Lethal Company and Valheim (when installed) into its Steam library and `seed` fills profiles for the first two; a new game needs its app id and folder added to the script.
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
| Deployer | `internal/deploy` | `Deployer` (`Plan`, `Apply`, `Purge`, `Recover`); `deploy.Register`; the catalog's `deploy` names the id |
| Framework | `internal/framework` | `Framework` (`ID`, `Matches`, `Analyze`); `framework.Register`. Content Patcher is the one driver |
| Loader | `internal/loader` | `Loader` (`ID`, `Formats`, `Status`, `Install`, `Contribute`) in `internal/loader/<id>/`; `loader.Register`; import in `internal/loader/all/all.go`. Capabilities by type assertion: `Vanilla`, `Owner`, `WithLogs`, `WithConfig`, `WithLaunchSettings` (shown in Edit profile), `WithPlayerLog`, `WithCompanion`, `Console`, `Releases`, `ProcessNames`, `SteamExe`. A loader that redirects the game to the profile (SMAPI) declares no install-side files; one that needs files beside the executable (BepInEx's Doorstop `winhttp.dll`, `doorstop_config.ini`) declares them in its plan and the deployer places them |

### Self-test

```sh
go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime ./internal/game ./internal/browse
scripts/selftest.sh start
```

`scripts/selftest.sh` serves the sandbox at `http://127.0.0.1:$PORT` (default 9455); `setup` creates the Steam library only. It names Stardew Valley and Lethal Company itself (`APP_ID`, `LC_APP_ID`, `LC_FOLDER`, `seed_lc`); a new game needs the same before the sandbox holds a copy.
