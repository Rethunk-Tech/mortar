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
| `launch <game> <profile> [--preset NAME] [--wait]`, `status`, `stop <game>`, `runs`, `logs`, `logs search <query> [--profile <name>]` | play and read past runs |
| `launchers`, `launchers add\|remove <id> <folder>` | what Settings › Launchers shows and changes |
| `tools <game>`, `tools run <game> <profile> <tool>` | configured external tools; start one |
| `bundles <game>`, `bundles apply <game> <bundle> <profile>` | list saved bundles; copy one into a profile |
| `nexus untrack <game> --all\|--unused` | untrack that game's Nexus mods (`--yes` skips the prompt) |
| `update game profile UniqueID...\|--all` | queue selected or all available mod updates |
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
mortar mods stardew bf8012eb5944d3ad --json | jq -r '.[] | select(.enabled | not) | .uniqueId'
source <(mortar completion bash)
```

## Gate

```sh
bun run gate    # runs the steps in package.json's gate script; stops at the first failure
```

`main.go` embeds `frontend/dist`, so a fresh clone needs `bun run bindings && bun run --cwd frontend build` once before the gate, as CI's setup does. The lefthook pre-push hook runs `gate` ([rethunk-gate-cli](https://github.com/Rethunk-Tech/rethunk-gate-cli)), which runs these steps plus actionlint. Where CI runs it: [AGENTS.md](AGENTS.md#verify).

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

`.github/workflows/release.yml` runs on a `v*` tag: the gate on the amd64 leg (skipped when `ci.yml` already passed on that commit), then the AppImage, nfpm packages, Flatpak bundle (SDK cached between runs), the Windows exes and per-user NSIS installers for amd64 and arm64 (`bin/mortar-amd64-installer.exe`, `bin/mortar-arm64-installer.exe`, both cross-built on x86_64; the arm64 installer is arm64-native) on x86_64, the AppImage and nfpm packages again on an `ubuntu-24.04-arm` runner, the signed `manifest.json`, the rendered AUR `PKGBUILD` and `SRCINFO` (the release asset for `.SRCINFO`) and Flathub manifest and `SUBMISSION.md`, and the GitHub release with all of those files. Its notes are user-facing: `build/release/notes.sh` keeps the `feat`, `fix` and `perf` commits since the previous `v*` tag outside internal scopes, in sentence case without scopes, grouped New and Fixed, at most 15 lines each, and links the full changelog. Releases are immutable once published, so the release is created as a draft, every draft asset is downloaded and checked against the staged files and the manifest's digests (`build/release/verify.sh`), and only then published; a rerun finishes the existing draft. After publishing, `verify.sh --public` fetches the manifest, its assets, the pinned component assets and the newest components manifest without a token: a failure is a warning while a repo is private and fails the run once it is public. A manual dispatch (Actions › Release › Run workflow) is the dry run: both legs, the signed manifest, package-manager sources, the notes (in the run summary) and `verify.sh --staged` on the staged files, for `build/config.yml`'s version, with no GitHub release.

CI reads the repository secret `MORTAR_UPDATE_KEY`, which holds the private key file's PEM contents, not its path:

```sh
gh secret set MORTAR_UPDATE_KEY --repo Rethunk-Tech/mortar < ~/.config/mortar-release/updater.key
```

An optional organisation secret adds a release output (`gh secret set NAME --org Rethunk-Tech --repos Rethunk-Tech/mortar`); it is skipped with a notice in the run while unset:

- `MORTAR_REPO_GPG_KEY` (the armored private key of "Mortar packages <security@rethunk.tech>", fingerprint `3283604606CAE2295D476F9883BC8751EE6F773D`): after publishing, the `package-repo` job runs `scripts/package-repo.sh` on the release's `.deb`, `.rpm` and Arch packages and uploads the signed repository tree as the `package-repo` artifact and force-pushes it as the single commit of the orphan `packages` branch, which the site's `packages` component serves at https://mortar.rethunk.tech/packages and redeploys on push. `package-repo.yml` (Actions › Package repository) rebuilds it for an existing tag. The tree holds `deb/` (apt-ftparchive `Packages`, `Release`, `InRelease`, `Release.gpg`), `rpm/<arch>/` (createrepo_c metadata with a signed `repomd.xml.asc`; the packages stay unsigned and byte-identical to the release assets, so `mortar.repo` sets `repo_gpgcheck=1`, `gpgcheck=0`), `arch/<arch>/` (`repo-add` database with package and database `.sig` files), the client snippets `mortar.list`, `mortar.repo` and `pacman.conf`, and the public key as `mortar-archive-keyring.asc` and `.gpg`. The script refuses a key whose fingerprint differs from the committed `build/linux/repo/mortar-archive-keyring.asc`. It holds only the latest release. Locally: `MORTAR_REPO_KEY=<private key file> MORTAR_REPO_URL=<base url> scripts/package-repo.sh <asset dir> <out dir>` (needs podman or docker and gpg).

Per release:

1. Set `info.version` in `build/config.yml`, the only place the version is set, then run `go run ./cmd/version` to copy it into the Windows resources, the Linux metainfo, and the AUR `PKGBUILD` (`main.go` reads it from `build/config.yml` itself). Then run `build/release/notes.sh --metainfo build/linux/tech.rethunk.Mortar.metainfo.xml v1.2.3 HEAD`, which writes the same user-facing notes the release will carry into that version's `<release><description>` (software centres and Flathub show it). Gate (it fails on any copy that differs, and `appstreamcli validate --strict --no-net` checks the metainfo), commit and push `main`.
2. `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal that version with a leading `v`, or the release stops before building.

### Flathub

CI only attaches a single-file `.flatpak` for people who sideload, built by `build/linux/flatpak/tech.rethunk.Mortar.yml` from the CI binary. Listing on Flathub is a separate submission that builds the tagged source. `build/linux/flathub/render.sh OUT_DIR` writes it: `tech.rethunk.Mortar.yml` is the sideload manifest's header and finish-args (the one list of permissions; `render.test.ts` keeps the Flathub template free of its own) above `build/linux/flathub/tech.rethunk.Mortar.yml`'s modules, which build the wails3 CLI, the TypeScript bindings, the frontend (`vite build`; the catalogs, credits and notices are committed) and Mortar with `-X main.packaged=flatpak` inside the GNOME SDK with the Go and Node 24 SDK extensions and no network. `go-sources.json` lists every proxy.golang.org file a clean build of Mortar and the wails3 CLI reads on amd64 and arm64, served to the build by `GOPROXY=file://`; `node-sources.json` comes from [flatpak-node-generator](https://github.com/flatpak/flatpak-builder-tools/tree/master/node) over `yarn.lock`, bun's lock written in yarn v1 format (the generator reads no bun lock), for yarn's offline mirror. Rendering needs network, Go, bun, jq and `uvx`; `--local` builds this checkout's HEAD instead of the tag, for `flatpak run org.flatpak.Builder` tests (the sandboxed builder sees only home and `/var/tmp`, so test from a clone there). Each release attaches the four files with a `flathub-` prefix and `flathub-SUBMISSION.md` with the PR text:

1. [Flathub app requirements](https://docs.flathub.org/docs/for-app-authors/requirements): AppStream metainfo (`tech.rethunk.Mortar.metainfo.xml`), 128×128 and 256×256 icons, screenshots, AGPL-3.0 license text.
2. Fork [flathub/flathub](https://github.com/flathub/flathub), open a PR that adds `tech.rethunk.Mortar`, then maintain the app repo Flathub creates.
3. Flathub builds from source only ([requirements](https://docs.flathub.org/docs/for-app-authors/requirements): no network during the build, every dependency a manifest source with a public URL), which is what the rendered manifest does.
4. finish-args grant network, Wayland/X11, DRI, native and Flatpak Steam libraries, read-only Heroic (`xdg-config/heroic`) and Lutris game configs (`xdg-data/lutris`), read-write Flatpak Lutris and Heroic app data, default `~/GOG Games` and `~/Games/Heroic` install trees, `xdg-data/mortar`, `xdg-download`, removable media (`/run/media`, `/mnt`) for second-drive Steam libraries, `xdg-config/autostart:create` and `xdg-data/applications:create` (launch at login and profile shortcuts on the host), the Secret Service (`org.freedesktop.secrets`, Nexus sign-in) and `org.freedesktop.Flatpak` (`flatpak-spawn --host`, which starts Steam and the game outside the sandbox), and the single-instance bus name.

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

The repository secret `MORTAR_UPDATE_KEY` contains the PEM contents, as for releases. Run the Components workflow manually from Actions, or dispatch it from a component release:

```sh
gh workflow run components.yml --repo Rethunk-Tech/mortar
gh api repos/Rethunk-Tech/mortar/dispatches -f event_type=components
```

The workflow writes the secret to a temporary file, runs the generator with `GOTMPDIR=/var/tmp TMPDIR=/var/tmp`, and creates a `components-<serial>` release (not marked latest) holding `components.json` and `components.json.sig`, each published once because releases are immutable. Until `Rethunk-Tech/mortar-smapi-bridge` is public the generator fails with a 404 looking it up, since the workflow's token reads only this repo.
