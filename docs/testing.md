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
| Fuzz | `go test -fuzz=FuzzExtract -fuzztime=60s ./internal/archive` (also `FuzzCleanName`, `FuzzExtractZipEntries`, and `FuzzFOMODLayout`, `FuzzParse`, `FuzzParseR2Zip`, `FuzzModpackManifest` in their packages) | Hostile input never panics and never writes outside its destination. The default run replays each seed and every file in `testdata/fuzz` as a regression test |
| Property | `MORTAR_SYNC_SEEDS=2000 go test ./internal/syncsvc -run Property` | Two machines editing at random never lose or invent a change (`syncsvc/property_test.go`) |
| Updater e2e | `wails3 task test:updater` | The self-updater swaps the binary through a signed manifest (`-tags updatetest`; skipped under `-short`) |
| Real SMAPI installer | `MORTAR_SMOKE=1 go test ./internal/loader/smapi -run Smoke` | The real installer works on a copy of a Stardew install (needs the network) |
| Scan benchmark | `MORTAR_SCAN_BENCH=1 go test ./internal/framework/contentpatcher -run ScanBench` | Peak memory of a conflict scan on a large mod folder |
| Browser e2e | `bun run --cwd frontend e2e` | The main flows through the real UI against a seeded sandbox |
| Stardew regress | `scripts/selftest.sh regress` | Mortar launches a real profile, SMAPI loads every enabled mod, and the game folder hashes the same after the purge |
| Lethal Company regress | `scripts/selftest.sh regress --game lethal-company` | The same under Proton with BepInEx, and the plugins load |
| R2 step | `MORTAR_REGRESS_R2_CODE=<key> scripts/selftest.sh regress --game lethal-company` | An r2modman code imports into a new profile, downloads, launches, loads every plugin it counted, and the purge leaves the game folder unchanged |
| Windows VM | `/var/tmp/win11-vm/` (`README.txt`, `start.sh`) | Quick Windows repros and one-fix smokes; `scripts/windows-installer-smoke.ps1` for the installer. Long soaks are a human's |

The regress scripts need the real games, a display and (for Lethal Company) Proton, so they never run in CI. Their sandbox, environment variables and failure artefacts are described in [HUMANS.md](../HUMANS.md#stardew-regression-run-opt-in-not-in-the-gate). There is no mutation-testing run; the fuzz and property tests are the adversarial layer.
