# Test suite map

Where the tests are, what each opt-in run proves and how to start it. The gate that runs the default suite: [HUMANS.md](../HUMANS.md#gate). Rules for writing tests: least tests for the most coverage, real dependencies over mocks.

## Budget

The whole default suite, Go with `-race` and `bun test`, runs in under 30 s cold and 10 s warm. A slow path sits behind a flag below instead of in the default run. `scripts/gate.sh` runs the suite in parallel with the lint steps, so the gate takes as long as its slowest step.

## Where tests live

| Layer | Location | Fixtures and helpers |
| --- | --- | --- |
| Go unit and service tests | `internal/<pkg>/*_test.go`, beside the code | `internal/testenv` opens stores for a service test, `testenv/testfs` points the data folder at a temp dir (`datadir` refuses the real one under `go test`), `testenv/packs` builds Content Patcher packs |
| Archives and installers | `internal/archive`, `internal/installer`, `internal/fomod` | Crafted `.7z` and `.rar` files and golden layouts in each package's `testdata/`; zips are built in the test |
| Share and sync | `internal/share`, `internal/sharesvc`, `internal/syncsvc`, `internal/lan` | A loopback LAN pair, golden share links, a 7z collection in `sharesvc/testdata` |
| Catalog and sources | `internal/components`, `internal/source/...` | `go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime` is the check for a catalog edit ([HUMANS.md](../HUMANS.md#adding-a-game)) |
| Updater | `internal/updatesvc` | `testdata/fixture`, two server-mode builds behind a signed manifest |
| Frontend unit | `frontend/src/**/*.test.ts`, run by `bun test` | Pure logic only; components are not rendered under jsdom |
| Browser e2e | `frontend/e2e/*.pw.ts` | Playwright on Chromium against its own sandbox (`e2e/sandbox.ts`); named `.pw.ts` so `bun test` skips it |

Threat-model guards and the test that proves each: [security.md](security.md).

## Default run

```sh
bun run gate       # everything CI runs except e2e
bun run test       # go test -race ./... && bun test
go test -race ./internal/<pkg>
```

## Opt-in runs

| Run | Command | Proves |
| --- | --- | --- |
| Fuzz | `go test -fuzz=FuzzExtract -fuzztime=2m ./internal/archive`; every untrusted input's target is named in [security.md](security.md), each in its package's `fuzz_test.go` | Hostile input never panics, never writes outside its destination, and where a format round-trips, survives it. The default run replays each seed and every file in `testdata/fuzz` as a regression test |
| Property | `MORTAR_SYNC_SEEDS=2000 go test ./internal/syncsvc -run Property` | Two machines editing at random never lose or invent a change (`syncsvc/property_test.go`) |
| Updater e2e | `wails3 task test:updater` | The self-updater swaps the binary through a signed manifest (`-tags updatetest`; skipped under `-short`) |
| Real SMAPI installer | `MORTAR_SMOKE=1 go test ./internal/loader/smapi -run Smoke` | The real installer works on a copy of a Stardew install (needs the network) |
| Real Content Patcher profile | `MORTAR_CP_REAL=<game>/<profile id> HOME=<sandbox copy> go test ./internal/framework/contentpatcher -run RealIncremental` | An incremental check after a pack is edited, added or removed finds what a full check finds, on a real profile |
| Performance budgets | `bun run perf` (`gate --e2e` runs it too; plain build, never `-race`) | On a generated 811-mod Stardew profile (`internal/testenv/modfixture`, calibrated against the maintainer's data), a restarted Problems check over warm caches, the Content Patcher check after one pack edit and the drift walk each stay within twice the CPU time they took when the budgets were set (`internal/problems/perf_test.go`). `MORTAR_PERF_REAL=stardew/<profile id>` with `XDG_DATA_HOME` at a sandbox copy (`scripts/selftest.sh start --copy-data`) runs the same budgets on real data, to recalibrate the fixture. The Mods list's row build for the same size has its budget in the default run (`frontend/src/mods/useModGroups.test.ts`) |
| Scan benchmark | `MORTAR_SCAN_BENCH=1 go test ./internal/framework/contentpatcher -run ScanBench` | Peak memory of a conflict scan on a large mod folder |
| Browser e2e | `bun run --cwd frontend e2e` | The main flows through the real UI against a seeded sandbox |
| Stardew regress | `scripts/selftest.sh regress` | Mortar launches a real profile, SMAPI loads every enabled mod, and the game folder hashes the same after the purge |
| Lethal Company regress | `scripts/selftest.sh regress --game lethal-company` | The same under Proton with BepInEx, and the plugins load |
| BepInEx matrix | `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress --game lethal-company` | Each row of [bepinex-test-matrix.md](bepinex-test-matrix.md) passes: loader pins and Doorstop, a pinned modpack, probe plugins, updates, sharing and package-layout edge cases. It needs the network and the .NET SDK, and starts the game 8 times besides the base run, so the launch cap below refuses it before it begins; its steps cannot share a launch, since each tests what a fresh start does |
| R2 step | `MORTAR_REGRESS_R2_CODE=<key> scripts/selftest.sh regress --game lethal-company` | An r2modman code imports into a new profile, downloads, launches, loads every plugin it counted, and the purge leaves the game folder unchanged |
| Launch harness | `scripts/selftest.sh harness-check` | With a dummy GTK window (zenity) in place of a game: launches through the guard land on the hidden display and reach no socket of the desktop session (its X server, Wayland socket, bus or audio), the fourth launch is refused with exit 3, and stopping hits exactly the recorded pids. Starts no game |
| Windows VM | `/var/tmp/win11-vm/` (`README.txt`, `start.sh`) | Quick Windows repros and one-fix smokes; `scripts/windows-installer-smoke.ps1` for the installer. Long soaks are a human's |

The regress scripts need the real games and (for Lethal Company) Proton, so they never run in CI.

## Sandbox game launches

Every game a self-test sandbox starts, from its browser view or from `regress`, goes through `scripts/launch-guard.sh`: a server build (`-tags server`) starts each direct launch through the program in `MORTAR_LAUNCH_WRAPPER`, which only `scripts/selftest.sh` sets. The guard:

- **Hidden display.** The sandbox runs a headless mutter with a virtual monitor, its own Xwayland, D-Bus session and a private runtime dir (`<sandbox>/run`, so no desktop Wayland socket, bus, PipeWire or Pulse socket is in reach). The server and everything it starts get that `DISPLAY`, `WAYLAND_DISPLAY`, `XAUTHORITY` and bus, with the desktop's unset; the guard refuses a launch whose environment names anything else. Proton and Wine draw through the Xwayland.
- **No network.** Each launch runs under `bwrap --unshare-net`, so it cannot reach a Steam client on loopback (`127.0.0.1:57343`); the Lethal Company Proton wrapper keeps its own as well.
- **Cap.** A session gets at most 3 launches across all its sandboxes. The session is `MORTAR_LAUNCH_SESSION`, else the agent session (`CLAUDE_CODE_SESSION_ID`), else the login session; its counter is `/var/tmp/mortar-launch-cap/<session>`, beside the sandboxes, since a regress sandbox lasts one run. The fourth launch exits 3 with the reason, and `regress` refuses up front a run that needs more launches than are left. The e2e sandbox has a session of its own per run, since `play.pw.ts` starts a stand-in, never a game.
- **Recorded pids.** Each game's pid and start time go to `<sandbox>/launches.pids`. `stop`, `destroy` and the end of `regress` stop exactly those processes, each found again by its start time (the pid is the one inside the server's pid namespace), never by name.
- **Intro skip.** Every guarded launch has `MORTAR_SKIP_INTRO=1`, which the Mortar bridge reads to skip Lethal Company's intro. Only the guard sets it, so a desktop Mortar's launches never carry it.

The Lethal Company regress builds the bridge from the `mortar-bepinex-bridge` checkout beside this repo (`MORTAR_REGRESS_BRIDGE_REPO` names another) and hands it to the server through `MORTAR_LOCAL_BRIDGES`, so a run tests the bridge as it is before its release; without a checkout it uses the published bridge. Their sandbox, environment variables and failure artefacts are described in [HUMANS.md](../HUMANS.md#stardew-regression-run-opt-in-not-in-the-gate). There is no mutation-testing run; the fuzz and property tests are the adversarial layer.
