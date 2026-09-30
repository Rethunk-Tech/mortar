# Mortar runbook

How to run, build and gate Mortar. What it is and the rules it keeps: [AGENTS.md](AGENTS.md); how it works now: [docs/architecture.md](docs/architecture.md); decided work not yet built: [docs/design.md](docs/design.md).

## Prerequisites

- Go (the version in `go.mod`) and [bun](https://bun.sh)
- The Wails CLI, `wails3` (the beta named in `go.mod`)
- GTK4 and WebKitGTK 6.0 development packages: `sudo dnf install gtk4-devel webkitgtk6.0-devel gcc-c++ pkgconf-pkg-config`
- `golangci-lint` and `lefthook` on PATH for the gate and its git hooks

```sh
bun install
lefthook install
```

## Run and build

```sh
wails3 dev      # app with the Vite dev server
wails3 build    # production binary in bin/mortar
```

## Gate

```sh
bun run gate    # runs the steps in package.json's gate script; stops at the first failure
```

The pre-push hook runs the same command. CI repeats it, then builds, only on `v*` tags or by manual dispatch.

## Release

```sh
wails3 task linux:create:appimage     # bin/mortar-linux-x86_64.AppImage
wails3 build GOOS=windows             # bin/mortar.exe
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

`release:manifest` refuses a `VERSION` other than `main.go`'s `version`, copies `bin/mortar.exe` to `bin/mortar-windows-amd64.exe`, and writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. Attach both assets and `manifest.json` to the `v1.2.3` GitHub release; the app reads the manifest from the latest release. Where the key lives: [docs/architecture.md](docs/architecture.md#release).
