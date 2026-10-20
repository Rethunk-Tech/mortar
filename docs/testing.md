# Test suite map

Where the tests are, what each opt-in run proves and how to start it. The gate that runs the default suite: [HUMANS.md](../HUMANS.md#gate). Rules for writing tests: least tests for the most coverage, real dependencies over mocks.

## Budget

- The default suite (Go with `-race` plus `bun test`) runs in under 10 s warm. Only the `main` package links the Wails `application` package (GTK and WebKit through cgo): services take a `winhost.Host` and `desktopnotify` a `Notifier`, both implemented in `wailshost.go`, so no other test binary pays that link. Cold (empty Go and lint caches) is not enforced.
- A slow path sits behind a flag below, not in the default run.
- Every `dotnet` invocation leaves an empty six-character dir in `TMPDIR`: a script that runs it gives it a per-run `TMPDIR`.
- `scripts/gate.sh` builds `go build -race ./...` first, so the linux lint, govulncheck (`GOFLAGS=-race`) and the race tests read that one instrumented compile; the windows lint cannot share it and starts beside it. The cold wall time is a compile chain: the Wails `application` cgo step, then `modernc.org/sqlite/lib`, and `go test` starts no test binary until every one is linked.
- Browser e2e stays under its 2-minute budget: `bun run e2e` (`frontend/e2e/run-shards.ts`) runs two `playwright test --shard` processes, each with its own sandbox (folder, port, display), because the specs mutate one seeded server. The longest file, `a11y.pw.ts`, bounds any shard count, and each shard costs a sandbox's memory and a headless display, so the count stays at the smallest that fits the budget.

## Where tests live

| Layer | Location | Fixtures and helpers |
| --- | --- | --- |
| Go unit and service tests | `internal/<pkg>/*_test.go`, beside the code | `internal/testenv` opens stores for a service test, `testenv/testfs` points the data folder at a temp dir (`datadir` refuses the real one under `go test`), `testenv/packs` builds Content Patcher packs |
| Archives and installers | `internal/archive`, `internal/installer`, `internal/fomod` | Crafted `.7z` and `.rar` files and golden layouts in each package's `testdata/`; zips are built in the test |
| Share and sync | `internal/share`, `internal/sharesvc`, `internal/syncsvc`, `internal/lan` | A loopback LAN pair, golden share links, a 7z collection in `sharesvc/testdata` |
| Catalog and sources | `internal/components`, `internal/source/...` | `go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime` is the check for a catalog edit ([HUMANS.md](../HUMANS.md#adding-a-game)) |
| Updater | `internal/updatesvc` | `testdata/fixture`, two server-mode builds behind a signed manifest |
| Frontend unit | `frontend/src/**/*.test.ts`, run by `bun test` (`bunfig.toml` preloads happy-dom) | Logic in `*.test.ts`. The five `*.test.tsx` files render through `@testing-library/react` or `renderToStaticMarkup`; keep DOM cases in those files |
| Browser e2e | `frontend/e2e/*.pw.ts` | Playwright on Chromium against its own sandbox (`frontend/e2e/sandbox.ts`); named `.pw.ts` so `bun test` skips it |

Threat-model guards and the test that proves each: [security.md](security.md).

## Default run

```sh
bun run gate       # everything CI runs except e2e
bun run test       # go test -race ./... && bun test
go test -race ./internal/<pkg>
```

The default suite is `scripts/gate.sh`, which starts the Go race tests, lints and bun steps in one bash process so they share one `GOTMPDIR` and one bindings build.

## Opt-in runs

| Run | Command | Proves |
| --- | --- | --- |
| Fuzz | `go test -fuzz=FuzzExtract -fuzztime=2m ./internal/archive` (each untrusted input's target is in [security.md](security.md), in its package's `fuzz_test.go`) | Hostile input never panics or writes outside its destination, and a round-tripping format survives it. The default run replays each seed and every file in `testdata/fuzz` |
| Property | `MORTAR_SYNC_SEEDS=2000 go test ./internal/syncsvc -run Property` | Two machines editing at random never lose or invent a change (`property_test.go`) |
| Updater e2e | `wails3 task test:updater` | The self-updater swaps the binary through a signed manifest (`-tags updatetest`; skipped under `-short`) |
| Real SMAPI installer | `MORTAR_SMOKE=1 go test ./internal/loader/smapi -run Smoke` | The real installer works on a copy of a Stardew install (needs the network) |
| Real Content Patcher profile | `MORTAR_CP_REAL=<game>/<profile id> HOME=<sandbox copy> go test ./internal/framework/contentpatcher -run RealIncremental` | An incremental check after a pack edit, add or removal finds what a full check finds |
| Performance budgets | `bun run perf` (`gate --e2e` runs it; plain build, never `-race`) | On a generated 811-mod Stardew profile (`internal/testenv/modfixture`, calibrated on the maintainer's data): a restarted Problems check over warm caches, the Content Patcher check after one pack edit and the drift walk each stay within twice the CPU time at calibration (`internal/problems/perf_test.go`). `MORTAR_PERF_REAL=stardew/<profile id>` with `XDG_DATA_HOME` at a `selftest.sh start --copy-data` sandbox runs them on real data, to recalibrate. The Mods list row build has its budget under the same switch (`frontend/src/mods/useModGroups.test.ts`) |
| Memory budget | `bun run mem` (`scripts/mem-budget.sh`; in no gate: needs the real 811-mod data and runs for many minutes) | Peak PSS of the Go server's process tree and the browser's (headless Chromium standing in for the WebView), per scenario: merged browse, a generated 500 MB zip install, a full Problems check, a launch (`play --test`). Exits 1 when a total is over 1152 MiB. `MORTAR_MEM_DATA=<data folder>` measures real data, `MORTAR_MEM_ONLY=browse,problems` picks scenarios, `MORTAR_PPROF=127.0.0.1:6062` (server builds) serves pprof and saves per-scenario heap and allocs profiles under `MORTAR_MEM_OUT` for `go tool pprof -top -sample_index=inuse_space`; `MORTAR_MEM_PROFILE`, `MORTAR_MEM_BUDGET_MIB` override (`frontend/e2e/mem-budget.ts`) |
| Render budget | `bun run renders` (`scripts/render-budget.sh`; in no gate: minutes in all) | Every screen and common interaction (`cases` in `frontend/e2e/render-budget.ts`; a new screen or interaction is a new case) on a copy of the maintainer's data, headless Chromium, production bundle: React commits and re-renders (with causes), DOM mutations, layouts, long tasks, backend calls, then heap, DOM nodes and listeners, and the same per 10 s idle. Exits 1 when an action re-renders over 1,000 components or an idle screen commits, re-renders or makes over 2 backend calls in 10 s (a case may set its own). `MORTAR_RENDER_ONLY=mods,config` picks cases, `MORTAR_RENDER_URL=http://127.0.0.1:9455` drives a running `selftest.sh` server, `MORTAR_RENDER_SEEDED=1` the seeded sandbox, `MORTAR_RENDER_DATA`/`_PROFILE`/`_OUT` pick data, profile and where `render-budget.json` goes, `MORTAR_RENDER_TOP=40` lists more than 8 of each |
| Scan benchmark | `MORTAR_SCAN_BENCH=1 go test ./internal/framework/contentpatcher -run ScanBench` | Peak memory of a conflict scan on a large mod folder |
| Browser e2e | `bun run --cwd frontend e2e` | The main flows through the real UI against a seeded sandbox. On a failure, or a sandbox server gone at teardown, its `server.log`, `crash.log`, `server.exit` and a `server-state.txt` are copied to `frontend/test-results/sandbox-server/` before removal |
| Stardew regress | `scripts/selftest.sh regress` | Mortar launches a real profile, SMAPI loads every enabled mod, and the game folder hashes the same after the purge |
| Lethal Company regress | `scripts/selftest.sh regress --game lethal-company` | The same under Proton with BepInEx, and the plugins load |
| BepInEx matrix | `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress --game lethal-company` | Each row of [bepinex-test-matrix.md](bepinex-test-matrix.md) passes. Needs the network and the .NET SDK; one launch beyond the base run, so with an r2 code the run is 3 launches, the session cap |
| R2 step | `MORTAR_REGRESS_R2_CODE=<key> scripts/selftest.sh regress --game lethal-company` | An r2modman code imports into a new profile, downloads, launches, loads every plugin it counted, and the purge leaves the game folder unchanged |
| Launch harness | `scripts/selftest.sh harness-check` | With a dummy zenity window in place of a game: guarded launches land on the hidden display and reach no socket of the desktop session, the fourth launch is refused with exit 3, and stopping hits exactly the recorded pids |
| Windows VM | `/var/tmp/win11-vm/` (`README.txt`, `start.sh`) | Quick Windows repros and one-fix smokes; `scripts/windows-installer-smoke.ps1` for the installer. Long soaks are a human's |

The regress scripts need the real games and (for Lethal Company) Proton, so they never run in CI. There is no mutation-testing run; fuzz and property tests are the adversarial layer.

## Sandbox game launches

Every game a self-test sandbox starts, from its browser view or `regress`, goes through `scripts/launch-guard.sh`: a server build (`-tags server`) starts each direct launch through the program in `MORTAR_LAUNCH_WRAPPER`, which only `scripts/selftest.sh` sets.

- **Hidden display.** Headless mutter with a virtual monitor, its own Xwayland, D-Bus session and a private runtime dir (`<sandbox>/run`, exported as `MORTAR_HIDDEN_RUNTIME`), so no desktop Wayland socket, bus, PipeWire or Pulse socket is in reach. The guard refuses a launch whose environment names anything else.
- **No network.** Each launch runs under `bwrap --unshare-net`, so it cannot reach a Steam client on loopback (`127.0.0.1:57343`).
- **Cap.** A session gets at most 3 launches across all its sandboxes. The session is `MORTAR_LAUNCH_SESSION`, else `CLAUDE_CODE_SESSION_ID`, else the login session; the counter is `/var/tmp/mortar-launch-cap/<session>`. The fourth launch exits 3, and `regress` refuses up front a run needing more launches than are left. The e2e sandbox has a session per run (`play.pw.ts` starts a stand-in, never a game).
- **Recorded pids.** Each game's pid and start time go to `<sandbox>/launches.pids`. `stop`, `destroy` and the end of `regress` stop exactly those, each found again by start time (the pid is the one inside the server's pid namespace), never by name.
- **Intro skip.** Guarded launches carry `MORTAR_SKIP_INTRO=1`, which the Mortar bridge reads to skip Lethal Company's intro; only the guard sets it.

The Lethal Company regress installs the bridge the catalog publishes, so a broken release asset fails it. Sandbox, environment variables (a local bridge build included) and failure artefacts: [HUMANS.md](../HUMANS.md#stardew-regression-run-opt-in-not-in-the-gate).
