# AGENTS.md

Mortar is a desktop mod manager for several games (`mortar games` lists the catalog; Stardew Valley, Lethal Company and The Sims 4 each exercise a different loader). Docs: [architecture](docs/architecture.md) (how it works), [design](docs/design.md) (decided, unbuilt work), [gui-design](docs/gui-design.md) (read before any UI change), [security](docs/security.md) (threat model, guard and test per untrusted input), [testing](docs/testing.md), game facts ([Lethal Company](docs/lethal-company.md), [Valheim](docs/valheim.md), [The Sims 4](docs/sims4.md), [Vintage Story](docs/vintage-story.md)).

## Testing

- Test in the Wails dev server's browser view, and with the real game when a test needs it, always against a copied game folder (never symlinks into a real install: the SMAPI installer writes through them), stopped by its recorded PID.
- Browser self-tests use `scripts/selftest.sh` (`wails3 task selftest ARGS="start [--copy-data]"`, `restart` after changes): a server-mode build at <http://127.0.0.1:9455> with a sandboxed home, minimal Steam library and copied game. Its header lists every subcommand (`start`, `setup`, `restart`, `stop`, `destroy`, `reap`, `seed`, `fake-install`, `sims4-install`, `ea-fixture`, `regress`, `curseforge`, `harness-check`).
- Your own sandbox (`MORTAR_SELFTEST_DIR=/var/tmp/<name>`, 2 to 6 GB) ends with `selftest.sh destroy` when the task is done; `stop` leaves the folder.
- Give each hidden-display regress or matrix run its own `MORTAR_LAUNCH_SESSION` key (a session is capped at 3 launches: [docs/testing.md](docs/testing.md#sandbox-game-launches)). Launch only against a sandbox copy with a direct launch, hash the game folder before and after to prove purge, and never launch on the maintainer's screen or against the maintainer's real data.
- The maintainer owns no Steam Deck and no Valheim install; the faked gamepad and sandboxed fixtures cover them.
- The sandbox reads the live signed catalog. To try a bundled `internal/components/components.json` change first, delete the sandbox's `cache/components-v2-manifest.json` and `start` with `HTTPS_PROXY=http://127.0.0.1:9` (Mortar falls back to the bundled copy).
- The sandbox server runs in its own pid namespace, so the pid in its `control.json` is not a host pid: prove a check that ends or inspects a running Mortar by pid with two plain `mortar-server` runs on a throwaway `HOME`.
- zenity and kdialog reach the desktop even with `DISPLAY` and `WAYLAND_DISPLAY` unset: run a native-dialog proof on the sandbox's headless display.
- `scripts/install-local.sh [--windows]` builds committed HEAD from the working tree and installs it over the maintainer's copy. Edit no tracked file until it finishes, or the binary holds an uncommitted change under HEAD's stamp.
- Builds (installers, binaries, the installed copy) run in the main checkout, never a throwaway worktree, clone or `git archive` extract (no commit stamp; ~1 GB of `/var/tmp` per fresh-clone gate). Run `git status --short` first.
- Gate-green is not QA. Before an install over the maintainer's app, run the sandbox with `--copy-data` at the gated SHA and walk every screen and dialog per enabled game (Browse sources, cards, click and right-click; each Problems section and its empty state; every Settings section). Fix what is visibly wrong (missing art, duplicates, text where icons belong, misaligned rows, dead clicks, wrong defaults, findings that would delete in-use data) before the install.
- A reported wrong default or behaviour is fixed only when the code path the maintainer's build takes shows it: trace the value from the Go registry through module-init order and fallbacks to the React render, fix every link, add a test that fails on the old code, and confirm in the installed build or a sandbox with the maintainer's copied settings. A stored-setting edit is not a fix.
- Site and AppStream screenshots use the default window size (`window.go` `defaultWindowWidth`×`defaultWindowHeight`, scale 1).

## Decided

- The frontend ships as one bundle read from disk: no code splitting, `lazy()` routes or chunk-size work; component names are kept.
- Wails v3, pinned in `go.mod` (the Rethunk-AI fork), upgraded on purpose ([pin](docs/architecture.md#stack)). React, TypeScript and MUI on Vite; never Next.js.
- Solid window with a wallpaper backdrop under a tint, frameless with a themed title bar, MUI dark theme ([surfaces](docs/gui-design.md#surfaces-and-colour)). Translucent window: [docs/design.md](docs/design.md#later).
- Games, stores, runtimes, sources, metadata providers and hosts are registries joined by the signed catalog; no code pairs one with another, and a game normally needs no code ([how](docs/architecture.md#games-and-the-catalog), [adding one](HUMANS.md#adding-a-game)). Loaders are drivers in `internal/loader`; `internal/game`'s `Game` holds only identity and the install folder.
- Greenfield: no migration, compatibility or transition code; the maintainers' own data is migrated by hand. Pre-1.0 with no users, breaking changes are fine; releases are v0.3.x patches until a break needs more.
- A finished feature (passed its gate and sandbox regress) ships enabled: no launch flag, env var or "after the maintainer's test" gate. Publishing and store listings still need the maintainer's go. A signed catalog release (`components.yml`) may be published whenever the catalog changes, but first decode a catalog that adds a field value with the oldest supported tag's `components.Decode` (since 0.3.2 `Decode` skips an entry an older build cannot read), and give an entry that needs newer driver code `minVersion` (honoured from 0.3.6).
- Not yet: translations, a translucent or frosted window, macOS, hosted short share links. The site deploys only with a release.
- Site claims are generated, never hand-typed: Mortar's counts and version come from the code, other managers' counts and latest releases from `web/src/data/rival-facts.json`, which `bun run site:refresh` writes and each release's site publish reruns. The site is Astro in `web/`, built into `site/` (not committed) by `bun run site`. A new figure on a page gets a placeholder and a fetch, not a literal.
- The merged all-sources browse is the primary way to get mods. Lethal Company is modded from Thunderstore, Nexus, CurseForge and GitHub ([docs/lethal-company.md](docs/lethal-company.md)).
- Docs never state a setting's default value; defaults live in code (`settings.Defaults()`, checked by the registry test) and `mortar settings get` shows them.
- Anything tied to a game's loader, launcher, saves or content format (SMAPI options, launch method, backup/update-before-Play, run logs, Content Patcher display, default nxm profile, pre-Play check) is a per-game setting with per-profile override. Mortar-wide settings are app chrome only (theme, density, dates, notifications, data folder, LAN, Mortar's own updates).
- A new loader driver, source, store or game honours every existing setting that applies to it and adds a setting only for behaviour no existing one covers. Adding a driver includes walking `internal/settings/registry.go` and stating, per setting, whether the driver reads it or why not.
- AUR packages are prepared and checked locally (PKGBUILD, `.SRCINFO`, makepkg/namcap in an Arch container); each push to aur.archlinux.org needs the AUR account holder's permission, relayed by the maintainer.
- Flatpak: Mortar hosts its own repo until Flathub approves it.
- The CLI exists for agents and automation; end users are GUI-only, so GUI work ranks above CLI parity.
- Share links name their game ([form](docs/architecture.md#sharing)).
- Store or launcher behaviour the maintainers cannot test (EA/Origin, Epic, Xbox, a game they do not own) follows the source of r2modman (ebkr/r2modmanPlus), Gale (Kesomannen/gale) or another established manager, cited by file.
- LAN sharing accepts only local-network peers by default; "Accept connections from any network address" opens it to VPN and tailnet peers; the default stays.
- Mortar never re-hosts mod files; downloads come from each mod's own source. Exception: a profile sent between the user's own computers over the LAN, paired with a code, carries its mod files from every source.
- Nexus downloads start only from Nexus's own Mod Manager Download button (or a Premium API download the user asked for in Mortar). The browser extension reads and marks Nexus pages and relays those clicks; it never starts downloads or opens Nexus pages itself.
- The browser extension lives in Rethunk-Tech/mortar-browser-extension with its own version; the native-messaging protocol is versioned in the handshake (`internal/nativehost` `Protocol`, `MinProtocol`, `MaxProtocol`).

## Verify

All tests together (Go and frontend) stay under 10s warm: no duplicate coverage across layers; e2e only for flows no unit test can cover, with its own 2-minute budget ([docs/testing.md](docs/testing.md)). The memory (`bun run mem`) and render (`bun run renders`) budgets are in no gate: a UI change that adds a screen or interaction adds its render case, and a change to a busy screen is checked with `MORTAR_RENDER_ONLY`.

Bindings: `bun run bindings`; Lingui: `bun run --cwd frontend i18n:extract && bun run --cwd frontend i18n:compile`. Taskfile tasks run as `wails3 task <name>` (no standalone `task` binary). `bun run gate` is the offline gate ([HUMANS.md](HUMANS.md) § Gate). CI (`.github/workflows/ci.yml`) runs it on pull requests and pushes to `main`, skipping Dependabot PRs and changes touching only Markdown, `docs/` (not `docs/user-guide.md`, which a frontend test reads) or issue templates; gate locally and batch pushes. Packaging runs only on `v*` tags or manual dispatch (`release.yml`).
