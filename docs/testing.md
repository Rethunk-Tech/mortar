# Test suite map

Where the tests are, what each opt-in run proves and how to start it. The gate that runs the default suite: [HUMANS.md](../HUMANS.md#gate). Rules for writing tests: least tests for the most coverage, real dependencies over mocks.

## Budget

- The default suite (Go with `-race` plus `bun test`) runs in under 10 s warm. Only the `main` package links the Wails `application` package (GTK and WebKit through cgo; `TestOnlyPackageMainImportsWails`): services take a `winhost.Host` and `desktopnotify` a `Notifier`, both implemented in `wailshost.go`, so no other test binary pays that link.
- A slow path sits behind a flag below, not in the default run.
- Every `dotnet` invocation leaves an empty six-character dir in `TMPDIR`: a script that runs it gives it a per-run `TMPDIR`.
- Browser e2e stays under its 2-minute budget: `frontend/e2e/run-shards.ts` splits it across sandboxes, because the specs mutate one seeded server. Each shard costs a sandbox's memory and a headless display, so the count stays at the smallest that fits.

## Where tests live

| Layer | Location | Fixtures and helpers |
| --- | --- | --- |
| Go unit and service tests | `internal/<pkg>/*_test.go`, beside the code | `internal/testenv` opens stores for a service test, `testenv/testfs` points the data folder at a temp dir (`datadir` refuses the real one under `go test`), `testenv/packs` builds Content Patcher packs |
| Archives and installers | `internal/archive`, `internal/installer`, `internal/fomod` | Golden layouts in each package's `testdata/`; zips are built in the test. The format readers, their caps and their crafted `.7z` and `.rar` fixtures are tested in the go-archive module; `internal/archive` tests only Mortar's face (preview, extensions, name policy) |
| Share and sync | `internal/share`, `internal/sharesvc`, `internal/syncsvc`, `internal/lan` | A loopback LAN pair, golden share links, a 7z collection in `sharesvc/testdata` |
| Catalog and sources | `internal/components`, `internal/source/...` | `go test ./internal/source/all ./internal/components ./internal/gamestore ./internal/runtime` is the check for a catalog edit ([HUMANS.md](../HUMANS.md#adding-a-game)) |
| Updater | `internal/updatesvc` | `testdata/fixture`, two server-mode builds behind a signed manifest |
| Frontend unit | `frontend/src/**/*.test.ts`, run by `bun test` (`bunfig.toml` preloads happy-dom) | Logic in `*.test.ts`. The `*.test.tsx` files render through `@testing-library/react` or `renderToStaticMarkup`; keep DOM cases in those files |
| Browser e2e | `frontend/e2e/*.pw.ts` | Playwright on Chromium against its own sandbox (`frontend/e2e/sandbox.ts`); named `.pw.ts` so `bun test` skips it |

Threat-model guards and the test that proves each: [security.md](security.md).

## Default run

```sh
bun run gate       # everything CI runs except e2e
bun run test       # go test -race ./... and bun test, in parallel
go test -race ./internal/<pkg>
```

The default suite is `scripts/gate.sh` (its header lists the steps).

## Opt-in runs

| Run | Command | Proves |
| --- | --- | --- |
| Fuzz | `go test -fuzz=FuzzExtract -fuzztime=2m` in the go-archive checkout (archives), or `go test -fuzz=FuzzX -fuzztime=2m ./internal/<package>` (each untrusted input's target is in [security.md](security.md), in its package's `fuzz_test.go`) | Hostile input never panics or writes outside its destination, and a round-tripping format survives it. The default run replays each seed and every file in `testdata/fuzz` |
| Property | `MORTAR_SYNC_SEEDS=2000 go test ./internal/syncsvc -run Property` | Two machines editing at random never lose or invent a change (`property_test.go`) |
| Updater e2e | `wails3 task test:updater` | The self-updater swaps the binary through a signed manifest (`-tags updatetest`; skipped under `-short`) |
| Real SMAPI installer | `MORTAR_SMOKE=1 go test ./internal/loader/smapi -run Smoke` | The real installer works on a copy of a Stardew install (needs the network) |
| Real Content Patcher profile | `MORTAR_CP_REAL=<game>/<profile id> HOME=<sandbox copy> go test ./internal/framework/contentpatcher -run RealIncremental` | An incremental check after a pack edit, add or removal finds what a full check finds |
| Performance budgets | `bun run perf` (`gate --e2e` runs it; plain build, never `-race`) | A restarted Problems check, the Content Patcher check after one pack edit and the drift walk stay within their CPU budgets on a generated Stardew profile (`internal/problems/perf_test.go`, `internal/testenv/modfixture`). `MORTAR_PERF_REAL=stardew/<profile id>` with `XDG_DATA_HOME` at a `selftest.sh start --copy-data` sandbox runs them on real data. The Mods list row build has its budget under the same switch (`frontend/src/mods/useModGroups.test.ts`) |
| Memory budget | `bun run mem` (`scripts/mem-budget.sh`; in no gate: needs the maintainer's real data and runs for many minutes) | Peak PSS of the Go server and the browser per scenario; exits 1 over budget. Scenarios, data, pprof and the budget are in the header of `frontend/e2e/mem-budget.ts` |
| Render budget | `bun run renders` (`scripts/render-budget.sh`; in no gate) | Every screen and common interaction (`cases` in `frontend/e2e/render-budget.ts`; a new screen or interaction is a new case) on a copy of the maintainer's data: render counts, idle activity and layout breaks, in Chromium, and with `MORTAR_RENDER_WEBKIT=1` (`bun run renders:release`) in WebKitGTK too, which lays some CSS out differently. Thresholds, `MORTAR_RENDER_ONLY`, `MORTAR_RENDER_SHOTS` and the report (`index.html` beside the JSON) are described in the header of `render-budget.ts`. `bun run renders:diff OLD/chromium NEW/chromium` compares two runs' pictures; `bun run renders:webkit` runs the WebKitGTK pass alone against a running sandbox (`MORTAR_SELFTEST_DIR`) |
| Scan benchmark | `MORTAR_SCAN_BENCH=1 go test ./internal/framework/contentpatcher -run ScanBench` | Peak memory of a conflict scan on a large mod folder |
| Browser e2e | `bun run --cwd frontend e2e` | The main flows through the real UI against a seeded sandbox. |
| Stardew regress | `scripts/selftest.sh regress` | Mortar launches a real profile, SMAPI loads every enabled mod, and the game folder hashes the same after the purge |
| Lethal Company regress | `scripts/selftest.sh regress --game lethal-company` | The same under Proton with BepInEx, and the plugins load |
| BepInEx matrix | `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress --game lethal-company` | Each row of [bepinex-test-matrix.md](bepinex-test-matrix.md) passes. Needs the network and the .NET SDK; three launches, the whole session cap, so `MORTAR_REGRESS_R2_CODE` is refused with it |
| R2 step | `MORTAR_REGRESS_R2_CODE=<key> scripts/selftest.sh regress --game lethal-company` | An r2modman code imports into a new profile, downloads, launches, loads every plugin it counted, and the purge leaves the game folder unchanged |
| Launch harness | `scripts/selftest.sh harness-check` | With a dummy zenity window in place of a game: guarded launches land on the hidden display and reach no socket of the desktop session, the fourth launch is refused with exit 3, and stopping hits exactly the recorded pids |
| Windows VM | `/var/tmp/win11-vm/` (its `README.txt`) | Quick Windows repros and one-fix smokes; `scripts/windows-installer-smoke.ps1` for the installer. Long soaks are a human's |

The regress scripts need the real games and (for Lethal Company) Proton, so they never run in CI. There is no mutation-testing run; fuzz and property tests are the adversarial layer.

## Sandbox game launches

Every game a self-test sandbox starts, from its browser view or `regress`, goes through `scripts/launch-guard.sh`: a server build (`-tags server`) starts each direct launch through the program in `MORTAR_LAUNCH_WRAPPER`, which only `scripts/selftest.sh` sets.

- **Hidden display.** Headless mutter with a virtual monitor, its own Xwayland, D-Bus session and a private runtime dir (`<sandbox>/run`, exported as `MORTAR_HIDDEN_RUNTIME`), so no desktop Wayland socket, bus, PipeWire or Pulse socket is in reach. The guard refuses a launch whose environment names anything else.
- **No network.** Each launch runs under `bwrap --unshare-net`, so it cannot reach a Steam client on loopback (`127.0.0.1:57343`).
- **Cap.** A session gets at most 3 launches across all its sandboxes; the fourth exits 3 (`scripts/launch-guard.sh` names the session key and counter). `regress` refuses up front a run needing more launches than are left. The e2e sandbox has a session per run (`play.pw.ts` starts a stand-in, never a game).
- **Recorded pids.** Each game's pid and start time go to `<sandbox>/launches.pids`. `stop`, `destroy` and the end of `regress` stop exactly those, each found again by start time (the pid is the one inside the server's pid namespace), never by name.
- **Intro skip.** Guarded launches carry `MORTAR_SKIP_INTRO=1`, which the Mortar bridge reads to skip Lethal Company's intro; only the guard sets it.

The Lethal Company regress installs the bridge the catalog publishes, so a broken release asset fails it. Sandbox, environment variables (a local bridge build included) and failure artefacts: [HUMANS.md](../HUMANS.md#stardew-regression-run-opt-in-not-in-the-gate).
