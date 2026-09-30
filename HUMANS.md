# Mortar runbook

How to run, build and gate Mortar. What it is and the rules it keeps: [AGENTS.md](AGENTS.md); decided work not yet built: [docs/design.md](docs/design.md).

## Prerequisites

- Go 1.25+ and [bun](https://bun.sh)
- The Wails CLI, `wails3` v3.0.0-beta.26
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
bun run gate    # biome, golangci-lint, tsc, go vet, go test; stops at the first failure
```

The pre-push hook runs the same command. CI repeats it, then builds, only on `v*` tags or by manual dispatch.
