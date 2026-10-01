# Mortar runbook

How to run, build and gate Mortar. What it is and the rules it keeps: [AGENTS.md](AGENTS.md); how it works: [docs/architecture.md](docs/architecture.md); decided work not yet built: [docs/design.md](docs/design.md).

## Prerequisites

- Go (the version in `go.mod`) and [bun](https://bun.sh)
- The Wails CLI, `wails3`, built inside the `Rethunk-AI/wails` fork's module at the version `go.mod` pins, since `go install` ignores `replace`: `go mod download github.com/wailsapp/wails/v3`, then `(cd "$(go list -m -f '{{.Replace.Dir}}' github.com/wailsapp/wails/v3)" && go build -o "$(go env GOPATH)/bin/wails3" ./cmd/wails3)`
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
```

The frontend build first runs `scripts/gen-credits.ts`, which rewrites `frontend/src/settings/generated/credits.json` (the licence list on Settings › About) from `frontend/package.json` and `go.mod`; commit the file when it changes.

## Gate

```sh
bun run gate    # runs the steps in package.json's gate script; stops at the first failure
```

The pre-push hook runs the same command. CI repeats it in the release workflow, which runs only on `v*` tags or by manual dispatch.

## Release

```sh
wails3 task linux:create:appimage     # bin/mortar-linux-x86_64.AppImage
wails3 task linux:nfpm                # .deb, .rpm, Arch package, bin/mortar-linux-amd64
wails3 task linux:flatpak             # bin/mortar-linux-x86_64.flatpak (needs flatpak-builder)
wails3 build GOOS=windows             # bin/mortar.exe
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

`release:manifest` refuses a `VERSION` other than `main.go`'s `version`, copies `bin/mortar.exe` to `bin/mortar-windows-amd64.exe`, and writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. Attach the AppImage, Windows exe, installer, `manifest.json`, packaged Linux files and Flatpak bundle to the `v1.2.3` GitHub release; the app reads the manifest from the latest release (AppImage and Windows only). Where the key lives: [docs/architecture.md](docs/architecture.md#release).

### Cutting a release in CI

`.github/workflows/release.yml` does all of the above on a `v*` tag: it runs the gate, builds the AppImage, nfpm packages, Flatpak bundle, `bin/mortar.exe` and the per-user NSIS installer (`bin/mortar-amd64-installer.exe`), signs `manifest.json`, and publishes the GitHub release with those files. A manual dispatch (Actions › Release › Run workflow) runs the same build and signing for `main.go`'s version and publishes nothing.

### Flathub (NOMAD)

CI only attaches a single-file `.flatpak` for people who sideload. Listing on Flathub is a separate submission using `build/linux/flatpak/tech.rethunk.Mortar.yml`:

1. [Flathub app requirements](https://docs.flathub.org/docs/for-app-authors/requirements): AppStream metainfo (`tech.rethunk.Mortar.metainfo.xml`), 128×128 and 256×256 icons, screenshots, AGPL-3.0 license text.
2. Fork [flathub/flathub](https://github.com/flathub/flathub), open a PR that adds `tech.rethunk.Mortar`, then maintain the app repo Flathub creates.
3. Build from source inside the GNOME SDK (or keep the file-source binary and accept Flathub review of that choice). The GitHub bundle is not what Flathub builds.
4. finish-args already grant network, Wayland/X11, DRI, `~/.local/share/Steam`, Flatpak Steam's data, and `~/.local/share/mortar`.

### AUR mortar-bin (NOMAD)

`build/linux/aur/PKGBUILD` and `.SRCINFO` install `mortar-linux-amd64` from the GitHub release plus the tagged desktop entry and icon. Publishing:

1. `git clone ssh://aur@aur.archlinux.org/mortar-bin.git`
2. Copy `PKGBUILD` and `.SRCINFO` in, set `pkgver` to the tag, run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`.
3. `git add PKGBUILD .SRCINFO && git commit -m "mortar-bin $pkgver" && git push`

Add the repository secret `MORTAR_UPDATE_KEY` holding the full contents of the private key file (PEM, as generated), not its path:

```sh
gh secret set MORTAR_UPDATE_KEY --repo Rethunk-AI/mortar < ~/.config/mortar-release/updater.key
```

Then, per release:

1. Set `const version` in `main.go` to the new version, gate, commit and push `main`.
2. `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal `main.go`'s version with a leading `v`, or the manifest step fails and nothing is published.

## Publishing components

`components.source.json` is the allowlisted input for the signed loader and bridge manifest. The generator resolves its GitHub releases, hashes each asset, refreshes the embedded fallback and verifies a signature when `MORTAR_UPDATE_KEY` names the private key file:

```sh
task components:refresh
```

The repository secret `MORTAR_UPDATE_KEY` contains the PEM contents, as for releases. Run the Components workflow manually from Actions, or dispatch it from a component release:

```sh
gh workflow run components.yml --repo Rethunk-AI/mortar
gh api repos/Rethunk-AI/mortar/dispatches -f event_type=components
```

The workflow writes the secret to a temporary file, runs the generator with `GOTMPDIR=/var/tmp TMPDIR=/var/tmp`, creates the fixed `components` release when needed, and uploads `components.json` and `components.json.sig` with `--clobber`.
