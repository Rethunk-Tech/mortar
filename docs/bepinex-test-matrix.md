# BepInEx test matrix

What proves Mortar's BepInEx 5 support, with Lethal Company as the real game: each row names the automated test or regress step that covers it. Run it with `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress --game lethal-company` (code: `scripts/regress-bepinex.sh`). Each step writes one PASS or FAIL row with its evidence to `matrix.tsv` in the sandbox; any FAIL fails the regress. It needs the copied game, Proton, the network (Thunderstore) and the .NET SDK, whose Roslyn builds the probe plugins in `scripts/bepinex-probe/` against the copied game's assemblies. Suite map: [testing.md](testing.md).

Probe packages are Thunderstore zips the regress builds: one plugin source (`Probe.cs`) compiled per variant, each logging its config values (a string and a float) and whether a data file sits beside it, plus a preloader patcher (`Patcher.cs`). Variants throw in `Awake`, quit eight seconds after the main menu, log a warning heartbeat and one error, or write `LCSaveFile1` through the game's Easy Save 3.

## Launches

A session may start the game 3 times ([testing.md](testing.md#sandbox-game-launches)), so everything that needs no game process runs *offline* (profile files, the command line Mortar would start via `launchsvc` `PreviewCommand`, Problems, the queue, the window's calls), and the rest runs in three launches, using the whole cap (`MORTAR_REGRESS_R2_CODE` is refused with the matrix). (c) and (b) each differ from (a) in one variable, so the `bridge.reachable` rows say which, if either, costs Mortar its bridge:

- **(a)** the base run, on the reinstalled latest pack (Doorstop 4), its profile filled first with the modpack, the probe packages (Base, Beat, Nested, Own, Caps, Flat, Deep, Patcher, DupA, DupB) and two config edits. A measured launch (Measure next launch, asked before it starts), so it proves startup timing, in-game measuring, loader loading, the plugin count, the live Console, every edge case and the config edits; it ends with a Stop from Mortar.
- **(c)** the Throw and Quit probes on the latest pack (5.4.2305), run first while that pack is pinned: crash analysis, a natural exit, an unmeasured launch measuring nothing, and whether a plugin throwing in `Awake` alone takes the bridge down.
- **(b)** the Save and Quit probes on the older pack pinned (5.4.2100, Doorstop 3), no throwing plugin: the pinned loader loading, the save the update rows use, a natural exit, and whether the old pack alone costs the bridge.

Each launch waits up to 120 s for the main menu (the `Bridge plugin alive after scene MainMenu` line in `BepInEx/LogOutput.log`; the bridge is built from the `mortar-bepinex-bridge` checkout beside this repo). `bridge.reachable.a|c|b` read `BepInEx/config/mortar-bepinex-bridge.json` and send its port a `status` query from inside the game's network namespace (`nsenter`; the game's 127.0.0.1 is not the host's); a row passes on `ok` with scene `MainMenu`. `bridge.survived.*` are INFO rows counting True and False: with the default `HideManagerGameObject = false`, Lethal Company's first scene load destroys the loader's manager object and its plugin components, so False is expected and only an unreachable bridge fails.

## Measured launch and live state

Mortar's bridge client dials the host's 127.0.0.1, which under `--unshare-net` is not the game's, so during (a) and (c) the matrix runs a relay (`mx_bridge_relay`) that passes each command into the game's namespace with `nsenter`; Mortar's calls (`MeasureInGame`, the live poll) run unchanged. The window's events are read from `/wails/events` into `events-a.jsonl`.

| Item | Covered by |
| --- | --- |
| The bridge's preloader patcher times the measured launch (`Timing this launch's plugins until scene MainMenu`, no `Failed to run Initializer`) | launch (a) `startup.patcher` |
| The startup report: `startup/<utc>.json` written, `.measure-launch` and `.measure-next-launch` consumed, one row per Loading line, `entryMissed` 0 | launch (a) `startup.report`; bridge `StartupTimerTests`, `TestPrepareStartupArmsTheBepInExPatcherOnlyForAMeasuredLaunch` |
| Mortar's Startup view names every row by its package | launch (a) `startup.mortar`; `TestBepInExReportRowsBecomeTheirPackages` |
| Start measuring in game wraps plugin patches and updates (`Measuring N plugins: X methods timed, Y could not be wrapped`, Y at most 5 or 5%) | launch (a) `perf.start`; bridge `PerfTests` |
| A summary 10 s later: percentiles in order, rows named by package, saved and listed by `mortar perf reports` | launch (a) `perf.summary`; `TestPerfRowsNameAndAddUpEachPackagesPlugins` |
| Wrapping raises no new exceptions | launch (a) `perf.stable` |
| An unmeasured launch measures nothing: `{"measured":false}` and no startup report | launch (c) `perf.unmeasured` |
| The window's `launch:live` event reaches scene MainMenu | launch (a) `live.scene` |
| Loaded badges: only enabled packages; of two packages declaring one GUID exactly one reads loaded | launch (a) `live.badges` |
| The version the bridge reports is recorded as the version last played (the row shows the value) | after launch (a) `lastplayed.version` |

## Loader

| Item | Covered by |
| --- | --- |
| Fresh install: the pack's files, marker, `doorstop_config.ini` and `.doorstop_version` in the profile | offline `loader.fresh`; `TestInstallPackAndDoorstopFiles`, `TestLoaderContract` |
| Pin an older pack (5.4.2100, Doorstop 3) over a newer one: no stale `.doorstop_version` or core file, the player's `BepInEx.cfg` kept, and Doorstop 3's flag on the command line | offline `loader.pin`; `TestAnOlderPackOverANewerOneKeepsNoneOfItsFiles` |
| The pinned older pack starts: BepInEx 5.4.21 logs its version and Doorstop runs it | launch (b) `loader.pin-run` |
| Unpin and install the latest: the pack's files and Doorstop 4's flag | offline `loader.unpin` |
| Reinstall a version over itself: the pack's files and flags | offline `loader.reinstall` |
| The reinstalled latest pack starts: BepInEx 5.4.23 logs its version and Doorstop runs it | launch (a) `loader.reinstall-run` |
| Per Doorstop: the profile's files are the pack's, the game folder holds the profile's absolute-target relay ini during the run, and the command line carries that Doorstop's flag spelling with the preloader in `Z:` form | offline `loader.pin`, `loader.unpin`, `loader.reinstall` (`mx_files_and_flags`); launch (a) `loader.reinstall-run` and (b) `loader.pin-run` (`mx_doorstop`); `TestLaunchArgs` |
| Uninstall: the game folder holds only the two proxy files during a run and hashes as before after every stop; a vanilla start switches off either Doorstop | launch (a) `stop.a`, (c) `launch.exit`, (b) `loader.pin-exit`; `TestLoaderContract`, `TestLaunchArgs`. No vanilla launch is run: it takes no launch prefix, so the regress cannot isolate it from Steam |
| Wine `winhttp` override in `user.reg` | `TestEnsureWinHTTPOverride`; launches (a), (c) and (b) |

## Mods

| Item | Covered by |
| --- | --- |
| A 60-package modpack (the most downloaded, pinned) with its dependencies, queued as Browse queues | offline `modpack.install` |
| Load order: every enabled package listed once, after what it needs | offline `modpack.loadorder` |
| Plugin GUIDs: every plugin BepInEx loaded is one Mortar read from a DLL, and every plugin Mortar read was loaded or skipped | `mods.guids` on what launch (a) logged (`TestRealBepInExRun` in `internal/dotnet`, run with `-bepinex-profile <profile folder>`) |
| Problems shows only true problems (missing dependencies really absent, deprecated packages deprecated on Thunderstore, clashes with two copies, every load failure in a log naming an installed package, nothing broken, duplicated, damaged or drifted) | `problems.true` on what launch (a) logged; `TestThePlayerLogBelongsOnlyToTheProfileThatRanLast` |
| Duplicate GUIDs flagged, with each copy's version, and BepInEx loads one | offline `mods.dupguid`, launch (a) `mods.dupguid-runtime`; `TestEnabledPackagesGiveAFilePackageItsManifestVersion` |
| Deprecated packages suggest their replacement | offline `mods.deprecated`; `TestDeprecatedListsPackagesAndTheirNamedReplacement`, `TestDeprecatedPackagesMatchesInstalledOnesAndCarriesTheReplacement` |
| Config edited through `configsvc.Service.Set` (a string and a float) before a launch: the plugin logs both values and the file keeps them | launch (a) `mods.config`, `mods.config-float`, then `mods.config-file`; `internal/configsvc` tests |

## Launch

| Item | Covered by |
| --- | --- |
| BepInEx loads every plugin: its count equals its Loading lines | launch (a) `launch.count`, `mods.guids` |
| The Console shows levels and sources live (two reads of `launchsvc.Service.Lines` during a run, the second with more lines) | launch (a) `launch.console`; `internal/launch` parser tests |
| Stop ends the game politely: the process is gone before anything is reaped, Unity logged its clean-exit statistics, Mortar is idle, the game folder purged | launch (a) `stop.a` |
| A natural exit returns Mortar to idle, records the run and leaves the game folder purged | launch (c) `launch.exit`, launch (b) `loader.pin-exit` |
| The bridge answers a status query at the main menu | launches (a), (c) and (b) `bridge.reachable.a`, `.c`, `.b` (`bridge.survived.*` notes whether its component survived) |
| Crash analysis of a plugin throwing in `Awake`: Problems names the probe package and exception; the run records the BepInEx version and errors | launch (c) `launch.crash`; `TestSummarizeReadsTheBepInExVersion`, `TestAnalyze` |

## Updates

| Item | Covered by |
| --- | --- |
| Update all | offline `updates.all` |
| Update everywhere | offline `updates.everywhere` |
| Roll back with the update's undo | offline `updates.rollback` |
| A save backup before the update holds the game-written `LCSaveFile1` and restores byte for byte | launch (b) writes the save (`saves.fixture`); offline `updates.backup`, `updates.restore`; `internal/backup` tests |

## Sharing

| Item | Covered by |
| --- | --- |
| Share link imported into a new profile through the window's calls: the same packages | offline `share.link` |
| `.mortar` export and import | offline `share.mortar` |
| LAN send to a second paired sandbox, carrying local packages and configs | offline `share.lan`; `internal/lan` loopback tests |
| r2modman code export and import, BepInExPack listed | offline `share.r2code` with `MORTAR_REGRESS_R2_EXPORT=1` (publishes to thunderstore.io); `TestExportCodeCarriesTheConfigFolder`; real-code import is `MORTAR_REGRESS_R2_CODE`, refused with the matrix (no launch left) |
| Gale and r2modman formats | `internal/pack` tests (`gale_test.go`, `pack_test.go`) |

## Package layout edge cases

| Item | Covered by |
| --- | --- |
| Nested plugin folders (`plugins/Deep/Er/x.dll`) | launch (a) `edge.nested`; `TestRoute` |
| Files under folders no rule names are flattened beside the plugin, as r2modman does (MirageCore's `FSharp.Core/`) | launch (a) `edge.flatten`, `edge.flatten-probe`; `TestRoute`, `TestLayoutGoldens` |
| Patchers in `BepInEx/patchers` | launch (a) `edge.patcher`, `edge.patchers` |
| `config` placed flat, `plugins` per package | launch (a) `edge.config-placement`; `TestRoute` |
| A package with its own `BepInEx/` layout | launch (a) `edge.own-layout`; `TestRoute` |
| Case: `BEPINEX/Plugins` and `BEPINEX/Config` | launch (a) `edge.case`; `TestRoute`, `TestLayoutGoldens` |
| Two names in one archive differing only in case are refused, naming the file (no copy is sure to be the one Wine loads) | `TestCaseCollision`, `TestInstallArchive` |
| A plugin path past 260 characters as Wine sees it | launch (a) `edge.longpath` |
