# Mortar runbook

How to run, build and gate Mortar. Rules: [AGENTS.md](AGENTS.md); how it works: [docs/architecture.md](docs/architecture.md); unbuilt work: [docs/design.md](docs/design.md).

## Prerequisites

- Go (the version in `go.mod`) and [bun](https://bun.sh).
- `wails3`, built inside the `Rethunk-AI/wails` fork's module at the version `go.mod` pins (`go install` ignores `replace`): `go mod download github.com/wailsapp/wails/v3`, then `(v=$(go env GOVERSION); cd "$(go list -m -f '{{.Replace.Dir}}' github.com/wailsapp/wails/v3)" && GOTOOLCHAIN=$v go build -o "$(go env GOPATH)/bin/wails3" ./cmd/wails3)`. Mortar's Go version builds it (the fork's own `go.mod` asks for an older one).
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

- `selftest` (`scripts/selftest.sh`; its header lists every subcommand and variable) runs a `server`-tagged binary with its own HOME, copied (never linked) game folders and a minimal Steam library, so real data, game and Steam config stay untouched. `--copy-data` copies live profiles and settings in once. `stop` leaves the folder; `destroy` deletes it.
- `e2e` (Playwright, Chromium only; [docs/testing.md](docs/testing.md)) starts its own sandbox, opens every profile tab and Settings page and fails on a console error, a failed request or a main-thread task over 200 ms. Not in the gate or CI; run it after each wave of UI changes.
- The frontend build first runs `scripts/gen-credits.ts` (rewrites `frontend/src/settings/generated/credits.json` from `frontend/package.json` and `go.mod`), then extracts and compiles the Lingui catalogs in `frontend/src/locales/`. Commit those files when they change.

## Command line

With Mortar running, `mortar <command>` asks the open app and prints a table, or JSON with `--json`. `mortar help` lists every command; exit codes, the `--json` error shape and window-launch arguments are in `internal/cli/cli.go`. `<profile>` is a profile id or name; a mod id is `<format>:<local id>` (a bare id uses the game's own format, SMAPI's UniqueID for Stardew). A destructive verb is refused until you pass `--yes`.

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

It runs in a throwaway sandbox, launches the first profile, and requires an empty diff of the game folder afterwards. A pass deletes the sandbox; a failure keeps it and prints the path. The `MORTAR_REGRESS_*` variables are in the header of `scripts/selftest.sh`.

### Lethal Company regression run (opt-in, not in the gate)

```sh
scripts/selftest.sh regress --game lethal-company     # needs Lethal Company and Proton - Experimental in the real Steam library
```

Same shape. Steps and `MORTAR_REGRESS_*` variables: `scripts/selftest.sh` (`regress_lc`); the multi-launch matrix: `scripts/regress-bepinex.sh` ([docs/bepinex-test-matrix.md](docs/bepinex-test-matrix.md)). Traps:

- Steam is never started. Every Proton process runs under `bwrap --unshare-net`: a copy registered under its Wine pid in a running Steam's loopback IPC leaves Steam on "Waiting for Lethal Company to shut down" for good.
- Launches run on the sandbox's hidden display under the session's launch cap ([docs/testing.md](docs/testing.md#sandbox-game-launches)).
- A mod that writes into the game folder at runtime fails the hash, because Mortar purges only what it placed. RugbugRedfern-Skinwalkers creates an empty `Dissonance_Diagnostics` folder, so an r2modman code that holds it passes with `MORTAR_REGRESS_R2_DISABLE=RugbugRedfern-Skinwalkers`.
- The bridge check needs the network: an offline run cannot download the published bridge and skips it (`MORTAR_REGRESS_LOCAL_BRIDGE=1` builds the checkout beside this repo instead).

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

Translators work in Hosted Weblate, which opens pull requests against `main`. The Lingui catalogs are split for them: `frontend/src/locales/<locale>/{app,mods,settings,profiles,console}.po`, merged into one `messages.ts` per locale at compile (`frontend/lingui.config.ts`). Those are generated and untracked: every build, dev server and gate compiles them, and `bun run --cwd frontend catalogs` does it by hand.

Weblate project settings:

- One component per catalog. Version control **GitHub pull request**, repository `https://github.com/Rethunk-Tech/mortar.git`, branch `main`. File format **gettext PO file**, file mask `frontend/src/locales/*/<name>.po`, source language English. Leave the template for new translations empty.
- A sixth component holds the text Go shows through the OS (tray, notifications, native dialogs): file mask `internal/i18n/locales/*/native.po`, same settings.
- Translation license **MIT** on every component (translations ship inside the AGPL-3.0-only app, and MIT lets the strings be reused elsewhere).
- Component flag `icu-message-format`, so Weblate checks placeholders and plurals.
- Project setting **Translation propagation** on: a message used in several folders sits in each of their catalogs.
- Glossary: upload `frontend/src/locales/glossary.tbx` to the project glossary; it explains each recurring term once.
- **Adding new translation:** contact maintainers. Weblate must not run msgmerge; `lingui extract` keeps every locale's catalogs in step with the source.

Starting a language: add its code to `availableLocales` in `frontend/src/i18n/locales.ts`, copy `en/native.po` to `internal/i18n/locales/<locale>/native.po` with empty msgstr and its `Language` and `Plural-Forms` header, run `bun run --cwd frontend i18n:extract` and commit the new catalogs; Weblate picks them up. A regional locale (`es-AR`) needs only the messages where its region differs: empty ones fall back to its base language (`es`), then English.

## Adding a game

A game is catalog data, with code only where its behaviour differs. `TestEveryCatalogReferenceResolves` (`internal/source/all/catalog_test.go`) fails when an entry names a source, loader, metadata id, store field, path role or path token nothing answers to.

`wails3 task components:audit` lists Thunderstore communities with 20 or more packages that the catalog lacks, and catalog games Nexus hosts for their loader (or CurseForge hosts, with `MORTAR_CURSEFORGE_KEY` set) that have no source there, and fails when a catalog game's community or loader pack has gone from Thunderstore; run it before a release.

### The catalog entry

Add the game to `games` in `components.source.json`, then publish as under [Publishing components](#publishing-components). `GameInfo` (`internal/components/components.go`) documents every field, and `GameInfo.Validate` rejects an incomplete entry. Traps that are not one field's:

- A key `GameInfo` does not know is dropped at publish, and an older Mortar ignores a field it lacks, so set `minVersion` when the entry needs driver code or fields an older build lacks.
- Known-broken lists are not part of the entry: they are published from Rethunk-Tech/mortar-known-broken (`internal/components/knownbroken.go`).
- The Sims 4 (`sims4`) is the worked `folder` game: loader `folder`, `deploy` `profile`, a `mods` target with `keepWhole` `["ts4script"]` and a `tray` target with `role` `tray`.

### A catalog-only game, end to end

A game on a loader, store and sources Mortar already has needs no Go: `game.Find` answers for every catalog entry, and only a game the entry cannot describe is a type in `internal/game` listed in `coded`. `TestACatalogOnlyGameNeedsNoCode` (`internal/game`) and `TestACatalogOnlyGameBrowsesAndAdds` (`internal/browse`) are the pattern: the entry goes in a test manifest selected with `components.Use`, never the shipped catalog. Lethal Company is the worked catalog entry.

1. **Entry.** Add it to `games` with `enabled: false` while checking.
2. **Components.** A loader whose release Mortar installs (SMAPI) or a companion (the bridge) is a `components` entry with a GitHub source, an asset pattern and a version policy. BepInEx ships as a Thunderstore package and needs none.
3. **Manifest.** Regenerate the embedded manifest ([Publishing components](#publishing-components)).
4. **Check.** `go test ./internal/components ./internal/source/all`.
5. **Try it.** In the sandbox, `MORTAR_ENABLE_GAMES=<id>` switches on a game the bundled manifest lists as disabled, so the entry must be in `components.source.json` and the regenerated bundled manifest first. `scripts/selftest.sh` copies Stardew Valley, Lethal Company and Valheim (when installed) into its Steam library; a new game needs its app id and folder added to the script.
6. **Enable.** Flip `enabled` to true and publish the manifest.

### Drivers

A new store, site, runtime, provider, host, installer, deployer, framework or loader is one Go package or file plus its registration; a game using only existing ones needs none. Each driver implements the interface named here, which documents its optional capabilities (type assertions):

- Store: where its files live and how to read them go in a go-gamelocator package (one per store); `Store` in `internal/gamestore/gamestore.go` gets a driver over it, added to `All()`, and `components.GameStores` the matching field.
- Source: `Source` in `internal/source/<id>/`; `source.Register`; import in `internal/source/all/all.go`. A searcher sets `Item.Adult` from the site's flag, since browse hides adult hits unless opted in. Tests serve the site from a fake HTTP server.
- Runtime: `Runtime` in `internal/runtime`, in `drivers` before `native`, which claims whatever nothing else does.
- Metadata provider: `internal/metadata`; add the id's case in `For`.
- Host: `Direct` (any HTTPS URL) and `GitHub` in `internal/host`, built where a download starts.
- Pack format: `Format` in `internal/pack`; pass it to `pack.Read` where the importer lists formats. It writes nothing.
- Installer: `Installer` in `internal/installer`; `installer.Register`; id in `detectionOrder` before `plain`. Every `Layout` path is validated to stay inside its target.
- Framework: `Framework` in `internal/framework`; `framework.Register`.
- Loader: `Loader` in `internal/loader/<id>/`; `loader.Register`; import in `internal/loader/all/all.go`. A loader that redirects the game to the profile declares no install-side files; one that needs files beside the executable declares them in its plan and the deployer places them.

### Self-test

```sh
go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime ./internal/game ./internal/browse
scripts/selftest.sh start
```

`scripts/selftest.sh setup` creates the sandbox's Steam library only.
