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

The pre-push hook runs the same command. CI repeats it in the release workflow, which runs only on `v*` tags or by manual dispatch.

## Release

```sh
wails3 task linux:create:appimage     # bin/mortar-linux-x86_64.AppImage
wails3 build GOOS=windows             # bin/mortar.exe
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

`release:manifest` refuses a `VERSION` other than `main.go`'s `version`, copies `bin/mortar.exe` to `bin/mortar-windows-amd64.exe`, and writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. Attach both assets and `manifest.json` to the `v1.2.3` GitHub release; the app reads the manifest from the latest release. Where the key lives: [docs/architecture.md](docs/architecture.md#release).

### Cutting a release in CI

`.github/workflows/release.yml` does all of the above on a `v*` tag: it runs the gate, builds the AppImage, `bin/mortar.exe` and the per-user NSIS installer (`bin/mortar-amd64-installer.exe`), signs `manifest.json`, and publishes the GitHub release with those four files. A manual dispatch (Actions › Release › Run workflow) runs the same build and signing for `main.go`'s version and publishes nothing.

Once, add the repository secret `MORTAR_UPDATE_KEY` holding the full contents of the private key file (PEM, as generated), not its path:

```sh
gh secret set MORTAR_UPDATE_KEY --repo Rethunk-AI/mortar < ~/.config/mortar-release/updater.key
```

Then, per release:

1. Set `const version` in `main.go` to the new version, gate, commit and push `main`.
2. `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal `main.go`'s version with a leading `v`, or the manifest step fails and nothing is published.
