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
| `mods <game> <profile>`, `mods enable\|disable\|pin\|unpin\|remove ... <mod id>...`, `mod ... <mod id>` | list, change and inspect mods (mod id is the SMAPI UniqueID) |
| `install <game> <profile> <archive>` | install a local archive |
| `conflicts`, `problems [--format text]`, `updates`, `saves <game> <profile>` | what the Problems, Mods and Saves tabs show |
| `problems dismissed`, `problems dismiss <index>`, `problems restore <token\|index>` (`--profile`, `--game stardew`) | dismiss and restore Problems-tab warnings like the GUI |
| `share`, `export <game> <profile> [file]`, `open <link\|file>` | share links and `.mortar` files |
| `launch <game> <profile> [--wait]`, `status`, `stop <game>`, `runs`, `logs`, `logs search <query> [--profile <name>]` | play and read past runs |
| `launchers`, `launchers add\|remove <id> <folder>` | what Settings › Launchers shows and changes |
| `tools <game>`, `tools run <game> <profile> <tool>` | configured external tools; start one |
| `bundles <game>`, `bundles apply <game> <bundle> <profile>` | list saved bundles; copy one into a profile |
| `nexus untrack <game> --all\|--unused` | untrack that game's Nexus mods (`--yes` skips the prompt) |
| `update game profile UniqueID...\|--all` | queue selected or all available mod updates |
| `queue retry\|skip [id]`, `queue pause\|resume\|clear` | control queued downloads |
| `backups list`, `backups create <save>`, `backups restore <name> [save...]` | list, pin a Manual backup of one save, or restore |
| `queue`, `doctor`, `version`, `completion bash\|zsh\|fish` | the download queue, the environment, shell completion |

`--json` writes failures to stderr as `{"error":"...","code":...}`. Exit code 2 means usage or confirmation was required, 3 means Mortar was not running or `play --check` found issues, and 1 means another failure. `mortar doctor` falls back to read-only offline checks when Mortar is not running: it exits 1 when those checks find problems and 3 when they find none, never 0.

```sh
mortar conflicts stardew "Profile 2"
mortar mods stardew bf8012eb5944d3ad --json | jq -r '.[] | select(.enabled | not) | .uniqueId'
source <(mortar completion bash)
```

## Gate

```sh
bun run gate    # runs the steps in package.json's gate script; stops at the first failure
```

The pre-push hook runs the same command. Where CI runs it: [AGENTS.md](AGENTS.md#verify).

## Release

```sh
wails3 task linux:create:appimage     # bin/mortar-linux-x86_64.AppImage
wails3 task linux:nfpm                # .deb, .rpm, Arch package, bin/mortar-linux-amd64
wails3 task linux:flatpak             # bin/mortar-linux-x86_64.flatpak (needs flatpak-builder)
wails3 task linux:build:arm64         # bin/mortar-linux-arm64 and its .deb, .rpm, Arch package
wails3 build GOOS=windows             # bin/mortar.exe
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

`linux:build:arm64` cross-compiles on an x86_64 machine with no emulator registered: it needs `zig`, `docker` (to download the arm64 Ubuntu packages it links against, extracted under `tmp/`), `nfpm` and `qemu-aarch64`, which checks that every shared library resolves. The arm64 AppImage is built only in CI.

`release:manifest` refuses a `VERSION` other than `build/config.yml`'s `info.version`, needs `bin/mortar-windows-amd64.exe` to exist (copy `bin/mortar.exe` to it first; `release.yml` does that copy), and writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. The app reads the manifest from the latest release, or the latest pre-release when Settings › Updates includes beta releases; packaged Linux installs leave updating to the package manager. Where the key lives: [docs/architecture.md](docs/architecture.md#release).

### Cutting a release in CI

`.github/workflows/release.yml` runs on a `v*` tag: the gate, then the AppImage, nfpm packages, Flatpak bundle, `bin/mortar.exe` and the per-user NSIS installer (`bin/mortar-amd64-installer.exe`) on x86_64, the AppImage and nfpm packages again on an `ubuntu-24.04-arm` runner, the signed `manifest.json`, and the GitHub release with all of those files. A manual dispatch (Actions › Release › Run workflow) builds and signs x86_64 only, for `build/config.yml`'s version, and publishes nothing.

CI reads the repository secret `MORTAR_UPDATE_KEY`, which holds the private key file's PEM contents, not its path:

```sh
gh secret set MORTAR_UPDATE_KEY --repo Rethunk-AI/mortar < ~/.config/mortar-release/updater.key
```

Per release:

1. Set `info.version` in `build/config.yml`, the only place the version is set, then run `go run ./cmd/version` to copy it into the Windows resources, the Linux metainfo, the AUR `PKGBUILD` and the extension manifest (`main.go` reads it from `build/config.yml` itself). Gate (it fails on any copy that differs), commit and push `main`.
2. `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal that version with a leading `v`, or the release stops before building.

### Flathub

CI only attaches a single-file `.flatpak` for people who sideload. Listing on Flathub is a separate submission using `build/linux/flatpak/tech.rethunk.Mortar.yml`:

1. [Flathub app requirements](https://docs.flathub.org/docs/for-app-authors/requirements): AppStream metainfo (`tech.rethunk.Mortar.metainfo.xml`), 128×128 and 256×256 icons, screenshots, AGPL-3.0 license text.
2. Fork [flathub/flathub](https://github.com/flathub/flathub), open a PR that adds `tech.rethunk.Mortar`, then maintain the app repo Flathub creates.
3. Build from source inside the GNOME SDK (or keep the file-source binary and accept Flathub review of that choice). The GitHub bundle is not what Flathub builds.
4. finish-args grant network, Wayland/X11, DRI, native and Flatpak Steam libraries, read-only Heroic (`xdg-config/heroic`) and Lutris game configs (`xdg-data/lutris`), read-write Flatpak Lutris and Heroic app data, default `~/GOG Games` and `~/Games/Heroic` install trees, `xdg-data/mortar`, and the single-instance bus name.

### AUR mortar-bin

`build/linux/aur/PKGBUILD` installs `mortar-linux-amd64` or `mortar-linux-arm64` from the GitHub release, plus the tagged desktop entry and icon. Publishing:

1. `git clone ssh://aur@aur.archlinux.org/mortar-bin.git`
2. Copy `PKGBUILD` in (its `pkgver` follows `build/config.yml` through `go run ./cmd/version`), run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`.
3. `git add PKGBUILD .SRCINFO && git commit -m "mortar-bin $pkgver" && git push`

## Publishing components

`components.source.json` is the allowlisted input for the signed loader and bridge manifest. The generator resolves its GitHub releases, hashes each asset, refreshes the embedded fallback and verifies a signature when `MORTAR_UPDATE_KEY` names the private key file:

```sh
wails3 task components:refresh
```

The repository secret `MORTAR_UPDATE_KEY` contains the PEM contents, as for releases. Run the Components workflow manually from Actions, or dispatch it from a component release:

```sh
gh workflow run components.yml --repo Rethunk-AI/mortar
gh api repos/Rethunk-AI/mortar/dispatches -f event_type=components
```

The workflow writes the secret to a temporary file, runs the generator with `GOTMPDIR=/var/tmp TMPDIR=/var/tmp`, creates the fixed `components` release when needed, and uploads `components.json` and `components.json.sig` with `--clobber`.
