# TODO

Work that is decided and not yet built. `AGENTS.md` and `docs/` describe what exists;
this file is the only place that describes what does not. Delete an entry in the
change that lands it. Design text lives in [docs/design.md](docs/design.md).

## Mortar pins a Wails fork instead of an upstream release

`go.mod` still `replace`s Wails with `Rethunk-AI/wails` branch `mortar/v3.1.0-combined`
(tags `v3.1.0-mortar.N`).
Every Mortar build, the AppImage path and the GTK4 window rest on that fork, so
an upstream Wails v3 tag cannot be taken until it carries the same hooks. The
cost is a private pin that has to be rebased onto each Wails move, and a
release that cannot yet drop the replace.

The decided shape is to drop the replace and pin an upstream release once one
carries what the branch adds. Rejected: shipping Mortar against stock Wails
before those PRs land, or carrying a permanent fork.

### Read

- `go.mod` — the `Rethunk-AI/wails` replace and the pin this change drops.
- [docs/architecture.md](docs/architecture.md#stack) — the stack pin this
  release is allowed to move.
- [docs/design.md](docs/design.md#release) — the WEP/PR list that gates the drop.

### Traps

- **errorkind falls back through `Options.MarshalError`.** Upstream #6199; Mortar's
  `errorkind_test.go` relies on service methods using that fallback. A pin that
  lands without it fails those tests with a different wire error.
- **Single-instance lock released at shutdown start.** Upstream #6240. Holding
  the lock until process exit blocks the next instance during an update restart.
- **Apply-on-exit is fork-only.** No upstream WEP yet. Dropping the replace
  before that exists removes the apply-on-exit path Mortar already ships.

### Acceptance

- `go.mod` has no `Rethunk-AI/wails` replace and pins an upstream Wails module
  version.
- That version includes #6202 (OnUpdateApplied, WEP #6203), #6199, #6240, #6197,
  #6281, #6282, #6283, #6526, #6571 to #6578, #6582, WEPs #6243, #6244 and #6580,
  apply-on-exit and the server-mode shutdown hooks.
- `bun run gate` stays clean.

## The Linux game sandbox is unproven on real games

A malicious mod runs in the game process and can read `~/.ssh`, browser
profiles and Mortar's data folder. The bubblewrap path is built and opt-in
(maintainer decision of 2026-11-07); what has not been proven is that a real
game launch hides the home canaries, still writes the game's own folders, and
loads every enabled mod. Until that run exists, the setting is a claim, not a
measured control.

The decided shape is the existing opt-in sandbox (`sandboxLaunch`,
`sandboxExtraFolders`), proven on copied game folders. Rejected until asked:
seccomp, a network namespace with a loopback-only bridge, Windows AppContainer
and macOS.

### Read

- `internal/sandbox/sandbox.go:70` (`sandbox.Argv`) — bubblewrap argv and mounts.
- `internal/sandbox/sandbox.go:138` (`sandbox.Credentials`) — credential-looking
  environment variables stripped.
- `internal/sandbox/sandbox.go:155` (`sandbox.SteamRO`) — Steam parts Proton
  reads, bind-mounted read-only.
- `internal/sandbox/dbus.go:35` (`sandbox.ProxyArgv`) and
  `internal/sandbox/dbus.go:81` (`sandbox.StartProxy`) — filtered session bus.
- `internal/sandbox/dbus.go:120` (`sandbox.TaskCap`) — process cap.
- `internal/sandbox/folders.go:12` (`sandbox.ExtraFolders`),
  `folders.go:38` (`sandbox.Allowable`), `folders.go:60` (`sandbox.holdsSecrets`)
  — extra folders and the refusals for keys and sign-ins.
- `internal/game/start.go:296` — `game.sandboxed` applies it to a direct start
  and fails instead of starting unsandboxed (`sandbox.ErrNoBwrap` at
  `sandbox.go:18`).
- `internal/launchsvc/sandbox.go:18` (`launchsvc.sandboxSpec`) — profile, saves,
  cache and extra folders.
- `internal/launchsvc/service.go:1102` — the setting forces a direct start even
  where the store would relay.
- `internal/launchsvc/runs.go:63` (`Run.Sandbox`) — folders the run could write.
- `internal/launchsvc/sandbox.go:99` (`launchsvc.SandboxReport`) and
  `internal/sandbox/denial.go:20` (`sandbox.Denials`).
- `frontend/src/console/SandboxNote.tsx` and render case `console-sandboxed-run`
  (`frontend/e2e/stateCases.ts:340`).
- `internal/settings/registry.go:271` — `sandboxLaunch` and `sandboxExtraFolders`.
- `frontend/src/settings/GameSettingsPage.tsx:84` and
  `frontend/src/settings/prefCopyGame.ts:130`.
- `scripts/selftest.sh` `regress_sandbox_check` — `MORTAR_REGRESS_SANDBOX`
  reaches the Stardew and Lethal Company regress; neither has run sandboxed on
  a real game yet.
- `internal/sandbox/sandbox_test.go`
  `TestSandboxedProcessSeesNeitherSSHNorARealHomeCanary` — real bwrap, no game.
- Tests: `internal/sandbox/sandbox_test.go`,
  `internal/launchsvc/sandbox_spec_test.go`.

### Traps

- **Never prove this on a real install.** The SMAPI installer writes through
  symlinks; copied game folders only, stopped by the recorded PID.
- **Missing bwrap must fail the launch.** Falling back to unsandboxed is the
  defect `ErrNoBwrap` exists to catch, including when Mortar is the Flatpak.
- **Home must be hidden before extra binds.** The unit tests require that order;
  reversing it lets a canary under `$HOME` show up after an extra-folder bind.

### Acceptance

- A launch prefix that reads `$HOME/.ssh` and a canary file in the real home
  starts inside the sandbox and finds neither, while the game's own folders
  write.
- Stardew with SMAPI starts sandboxed, loads every enabled mod and saves into
  the profile or the Saves folder; the game folder hashes as the plain regress
  requires (`MORTAR_REGRESS_SANDBOX` covers Stardew, not only Lethal Company).
- A gamepad and audio work in a manual run (human-only).
- `bun run gate` stays clean. No seccomp, network namespace, Windows or macOS
  sandbox ships in this change.

## Several launch and import paths have never been run on real installs

Unit and sandbox coverage cannot see a real GOG install, Flatpak Steam, a
long-lived Vortex folder, OneDrive's data-folder timings, or Stardew launched
through Steam on Linux so SMAPI loads the profile's mods. Those paths stay
unmeasured until an outside tester runs them.

The decided shape is those runs, when a tester with that install is
available. Rejected: treating sandbox fixtures as proof of GOG, Flatpak
Steam, Vortex or OneDrive.

### Read

- [docs/design.md](docs/design.md#later) — the parked tester list.
- [docs/testing.md](docs/testing.md) — copied game folders, never a real
  install as the target of a write; Flatpak Steam is the exception this
  entry is for.
- [docs/games.md](docs/games.md#smapi) — Stardew's Steam launch method.

### Traps

- **Do not write through the tester's live install to "make it easier".**
  Copied folders still apply except where the run _is_ the real GOG / Flatpak
  Steam / Vortex / OneDrive path.
- **Store behaviour the maintainers cannot test follows r2modman or Gale,
  cited by file** — a tester run is evidence; it does not rewrite those
  citations without a cite of what changed.

### Acceptance

- A launch from a real GOG install and from Flatpak Steam is recorded.
- A Linux launch with Stardew's launch method set to Steam confirms SMAPI
  loads the profile's mods.
- Import from Vortex against a long-lived Vortex folder is recorded.
- Timings with the data folder under OneDrive are recorded.
- `bun run gate` stays clean. No production default changes without the
  maintainer's copied settings showing the value being replaced.

## Hosted share short codes are not provisioned

The hosted short-code service is in the tree, but it is not live: there is no
Spaces bucket, App Platform secrets, or `packages`-branch image that runs
`cmd/sharehost` beside the site. The privacy page must describe the hosted
share data before that ships.

Rejected: encrypting payloads in the app (NOMAD chose the shortest links over
a server that cannot read shares), falling back to an unencrypted transfer,
or a service that re-hosts mod files (LAN profile sends between the user's
own computers remain the exception that carries files).

### Read

- [docs/architecture.md](docs/architecture.md#sharing) — the current share
  link form (game in the name).
- [docs/design.md](docs/design.md#later) — hosted short codes and versioning.

### Traps

- **Mortar never re-hosts mod files.** The hosted service is codes and
  versioning, not a CDN for archives. LAN sends between the user's computers
  remain the file-carrying exception.
- **LAN still requires the same protocol on both computers.** A newer Mortar
  refuses an older peer, code-only sends included.
- **Hosted shares are readable by us.** The privacy page
  (`web/src/pages/privacy/index.astro`) must say what the service stores
  (profile name, game, mod list, version, last-read time, client IP for rate
  limiting) and for how long, before the service ships. Share would then
  need to say which link kind is private.
- **The site deploys from the `packages` branch** (`package-repo.yml`), not
  from `main`; the service component needs its own deploy path and the Spaces
  key as an App Platform secret, never in the repo.

### Remaining work

1. Teach `.github/workflows/package-repo.yml` to build `cmd/sharehost` with
   `Dockerfile`, copy the Astro `site/` output to `/site` and the packages
   tree to `/packages` in the image (the Dockerfile ships the binary only),
   and publish that image from the `packages` branch.
2. Create a DigitalOcean Spaces bucket for `shares/` objects; do not commit
   its name or keys.
3. Set App Platform secrets `SHARE_S3_ENDPOINT`, `SHARE_S3_BUCKET`,
   `SHARE_S3_KEY`, `SHARE_S3_SECRET` on the service (never in the repo or
   logs).
4. Swap the app spec to `build/sharehost/app.yaml` (one `basic-xxs` service
   in place of the two static sites) and apply it; do not apply from this
   change.

### Acceptance

- The privacy page describes the hosted share data before the service ships.
- Mod files are not re-hosted. LAN file-carrying sends still require matching
  protocol versions.
- `bun run gate` stays clean.

## Nexus listing tiles have no Add to Mortar button

Nexus downloads start only from Nexus's own Mod Manager Download button (or a
Premium API download the user asked for in Mortar). Listing tiles have no
Add to Mortar control, and adding one needs Nexus's approval first. The
browser extension already reads and marks Nexus pages and relays those
clicks; it never starts downloads or opens Nexus pages itself.

The decided shape is an Add to Mortar button on Nexus listing tiles, after
Nexus approval ([docs/nexus-application.md](docs/nexus-application.md)).
Rejected: the extension starting downloads or opening Nexus pages; shipping
the button before approval.

### Read

- [docs/nexus-application.md](docs/nexus-application.md) — the approval this
  button waits on.
- [docs/design.md](docs/design.md#later) — the tile button.
- Rethunk-Tech/mortar-browser-extension — the extension that relays Nexus
  clicks and must not start downloads.

### Traps

- **The extension never starts downloads or opens Nexus pages.** Relaying the
  official button is the only download start besides a Premium API download
  the user asked for in Mortar.
- **Nexus approval first.** A tile button without it is a terms break, not a
  feature.

### Acceptance

- Nexus listing tiles show Add to Mortar after Nexus has approved it.
- Downloads still start only from Nexus's Mod Manager Download button or a
  Premium API download the user asked for in Mortar.
- `bun run gate` stays clean. The extension still does not start downloads or
  open Nexus pages.

## Windows builds are unsigned

Windows installers and binaries ship without Authenticode. SmartScreen and
enterprise policy treat them as untrusted, which is the whole cost of not
signing.

The decided shape is Windows code signing with Azure Trusted Signing.
Rejected: a one-off purchased EV cert outside Azure Trusted Signing, or
shipping a partial sign that still trips SmartScreen.

### Read

- [docs/design.md](docs/design.md#later) — Azure Trusted Signing.
- [docs/release.md](docs/release.md) — packaging and the Windows build this
  sign attaches to.

### Traps

- **Signing is a maintainer/account decision.** This entry does not create
  the Azure Trusted Signing account; it attaches signing once that exists.
- **Do not check a signing secret into the repo or the public workflow logs.**

### Acceptance

- Windows installers and binaries produced by the release path are signed
  with Azure Trusted Signing.
- `bun run gate` stays clean. No signing material is committed.

## GitHub topic-search fallback lives in Mortar instead of go-github

`internal/source/github/fallback.go` is Mortar-specific topic-search fallback.
It cannot move until a second consumer needs it; extracting it earlier would
be a Mortar-only library split.

The decided shape is to move that fallback into go-github once a second
consumer needs it. Rejected: splitting it into go-github (or a `mortar-*`
module) for one consumer; the 200-line rule still applies, and a client for
one external service may stand alone under 200 only when it is usage-agnostic.

### Read

- `internal/source/github/fallback.go` — the fallback this move picks up.
- [AGENTS.md](AGENTS.md) — library-split rules (`go-<name>`, 200-line
  threshold, `replace ... => ../<lib>` until Mortar's gate is green).

### Traps

- **No library repo until switching Mortar to it deletes more than 200 lines
  from Mortar** (or it is a usage-agnostic client). A split only Mortar could
  use is `mortar-<name>`, which this fallback should not become.
- **Build the library change and Mortar's use together** with `replace`; tag
  the library only after Mortar's gate passes against it.

### Acceptance

- A second consumer uses the fallback from go-github, and Mortar's
  `internal/source/github/fallback.go` is gone in favour of that module.
- `bun run gate` stays clean. The replace is swapped for the tag in the
  commit that lands the feature.

## There is no macOS build

Mortar does not ship darwin/arm64 or amd64. The Linux-testable parts (Steam
roots, markers, loader branches) are under the earlier ready-to-build entry;
packaging, codesign, notarisation, Homebrew and a real Mac run are parked
until the maintainer owns or borrows a Mac. Lethal Company and most BepInEx
games ship no macOS build, so the first targets are native-macOS games and,
for Windows-only games, a CrossOver or Whisky prefix.

The decided shape is full macOS support as specified in
[docs/design.md](docs/design.md#later). Catalog `darwin` path
templates and the macOS asset picker already exist; go-safefs already has a
darwin `Clone` (`clonefile`). Rejected: shipping unsigned (Gatekeeper blocks
it), adding billable macOS CI runners, or enabling MelonLoader on macOS (no
upstream macOS build; the driver already refuses Linux).

### Read

- [docs/design.md](docs/design.md#later) — scope, Wails, discovery,
  loaders, packaging, testing and done-when.
- [docs/architecture.md](docs/architecture.md#stack) — the pinned Wails fork
  `wails3 build GOOS=darwin` runs on.
- go-gamelocator's Unix Steam locator (`//go:build !windows`) — today lists
  only Linux roots; add `~/Library/Application Support/Steam` (r2modman's
  darwin game-directory resolver) and `GOOS=darwin go vet ./...` in that
  module's crosscompile gate; read `steamapps/libraryfolders.vdf` as Linux
  does.
- `internal/game/install.go`, `internal/gamestore/standalone.go` — `marker`
  needs `darwinMarker` like `LinuxMarker` (`Contents/MacOS/<exe>` or
  `Contents/Resources/Data`).
- `internal/loader/bepinex5` `driver.go` (and thunderpack's) — places the
  Windows `winhttp.dll` pair; needs a darwin branch beside `contributeLinux`
  (`run_bepinex` as Steam launch option `"<profile>/run_bepinex" %command%`).
- `internal/loader/smapi/install.go` — hardcodes Linux names; the bundle is
  `Stardew Valley.app/Contents/MacOS/StardewValley`.
- `internal/nxm` — runtime `nxm://` registration that `Info.plist`
  `CFBundleURLTypes` replaces on darwin.
- `internal/loader/thunderpack/thunderpack_test.go` — the `Platform: "darwin"`
  unit-test pattern for pure-Go parts on Linux.
- r2modman darwin Steam runner (`steam_osx -applaunch <id>`) and `macos_proxy`
  launch wrapper — Gale has no macOS code.

### Traps

- **`wails3 build GOOS=darwin` is Mac-only** (cgo and the macOS SDK). Check
  the fork's WKWebView paths first: frameless title bar, `BackgroundTypeSolid`,
  drag regions, the tray (`tray_other.go`), `nxm://`, single-instance.
- **Steam strips dyld injection for hardened binaries.** Test a hardened game
  early; Doorstop's `libdoorstop.dylib` is the BepInEx path.
- **No macOS runners in workflows** (billable minutes). CI stays Linux and
  Windows; manual testing needs a lender, or a one-off VM or hosted runner
  the maintainer approves.
- **Do not ship unsigned.** Gatekeeper blocks an unnotarised app. Codesign
  (hardened runtime) and notarise (`notarytool`) need an Apple Developer ID
  (paid account, a maintainer decision).
- **MelonLoader stays off on macOS.**

### Acceptance

- `wails3 build GOOS=darwin` produces a signed, notarised `.dmg` that launches
  on macOS 14+ (arm64 and Intel) with the frameless window, tray and `nxm://`
  links.
- A Stardew Valley profile installs SMAPI, launches through Steam and loads a
  mod (`SMAPI-latest.txt`).
- Steam root and `.app` game folders are found without a manual folder choice.
- The Homebrew cask `Casks/mortar.rb` in Rethunk-Tech/homebrew-tap installs
  and updates it.
- Self-update through `wails3 updater manifest` with a `darwin` artifact in
  `Taskfile.yml`.
- `bun run gate` passes on Linux with the new darwin unit tests.
