# BepInEx test matrix

What proves Mortar's BepInEx 5 support, with Lethal Company as the real game: each row names the automated test or regress step that covers it. The regress steps run with `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress --game lethal-company` (code: `scripts/regress-bepinex.sh`); each writes one PASS or FAIL row, with its evidence, to `matrix.tsv` in the sandbox, and any FAIL fails the regress. The run needs the copied game, Proton, the network (Thunderstore) and the .NET SDK, whose Roslyn builds the probe plugins in `scripts/bepinex-probe/` against the copied game's assemblies. Every Proton run stays in its own network namespace, away from a running Steam. Where the suite map lives: [testing.md](testing.md).

Probe packages are Thunderstore zips the regress builds: one plugin source (`Probe.cs`) compiled per variant, each logging its config values (a string and a float) and whether a data file sits beside it, plus a preloader patcher (`Patcher.cs`). Variants throw in `Awake`, quit the game after eight seconds, log a warning heartbeat and one error, or write `LCSaveFile1` through the game's own Easy Save 3.

## Launches

A session may start the game 3 times ([testing.md](testing.md#sandbox-game-launches)), so the matrix proves everything that needs no game process before or after its launches, and runs what does in two:

- **(a)** the base run's own launch, of its profile filled first with the modpack, the probe packages (Base, Beat, Nested, Own, Caps, Flat, Deep, Patcher, DupA, DupB) and two config edits, on the reinstalled latest pack (Doorstop 4). It proves that loader loading, the plugin count, the live Console, every edge case and the config edits loading, and ends with a Stop from Mortar.
- **(b)** a profile with the Throw, Save and Quit probes on the older pack pinned (5.4.2100, Doorstop 3). It proves the pinned loader loading, crash analysis, the save the update rows use, and a natural exit.

With `MORTAR_REGRESS_R2_CODE` the r2 step adds the third. A row marked *offline* runs with no game process: the profile's files, the command line Mortar would start (`launchsvc` `PreviewCommand`), Problems, the queue and the window's calls.

## Loader

| Item | Covered by |
| --- | --- |
| Fresh install: the pack's files, marker, `doorstop_config.ini` and `.doorstop_version` in the profile | offline `loader.fresh`; `TestInstallPackAndDoorstopFiles`, `TestLoaderContract` |
| Pin an older pack (5.4.2100, Doorstop 3) over a newer one: no stale `.doorstop_version` or core file, the player's `BepInEx.cfg` kept, and Doorstop 3's flag on the command line | offline `loader.pin`; `TestAnOlderPackOverANewerOneKeepsNoneOfItsFiles` |
| The pinned older pack starts: BepInEx 5.4.21 logs its version and Doorstop runs it | launch (b) `loader.pin-run` |
| Unpin and install the latest: the pack's files and Doorstop 4's flag | offline `loader.unpin` |
| Reinstall a version over itself: the pack's files and flags | offline `loader.reinstall` |
| The reinstalled latest pack starts: BepInEx 5.4.23 logs its version and Doorstop runs it | launch (a) `loader.reinstall-run` |
| Doorstop each time: the profile's files are the pack's, the game folder holds the same ini during the run, and the game's command line carries the flag spelling of that Doorstop with the profile's preloader in `Z:` form | offline `loader.pin`, `loader.unpin`, `loader.reinstall` (`mx_files_and_flags`); launch (a) `loader.reinstall-run` and (b) `loader.pin-run` (`mx_doorstop`); `TestLaunchArgs` |
| Uninstall: the game folder only ever holds the two proxy files during a run and hashes as before after every stop; a vanilla start switches off either Doorstop | launch (a) `stop.a`, launch (b) `launch.exit`; `TestLoaderContract`, `TestLaunchArgs`. Mortar has no action that removes the loader from a profile, and a vanilla start takes no launch prefix, so the regress cannot isolate it from Steam and does not launch one |
| Wine `winhttp` override in `user.reg` | `TestEnsureWinHTTPOverride`; launches (a) and (b) |

## Mods

| Item | Covered by |
| --- | --- |
| A 60-package modpack (the most downloaded, pinned) with its dependencies, queued as Browse queues | offline `modpack.install` |
| Load order: every enabled package listed once, after what it needs | offline `modpack.loadorder` |
| Plugin GUIDs: every plugin BepInEx loaded is one Mortar read from a DLL, and every plugin Mortar read was loaded or skipped | `mods.guids` on what launch (a) logged (`TestRealBepInExRun` in `internal/dotnet`, run with `-bepinex-profile <profile folder>`) |
| Problems shows only true problems: missing dependencies are really absent, deprecated packages are deprecated on Thunderstore, clashes have two copies, every load failure is in a log and names an installed package, nothing broken, duplicated, damaged or drifted | `problems.true` on what launch (a) logged; `TestThePlayerLogBelongsOnlyToTheProfileThatRanLast` |
| Duplicate GUIDs flagged, with each copy's version, and BepInEx loads one | offline `mods.dupguid`, launch (a) `mods.dupguid-runtime`; `TestEnabledPackagesGiveAFilePackageItsManifestVersion` |
| Deprecated packages suggest their replacement | offline `mods.deprecated`; `TestDeprecatedListsPackagesAndTheirNamedReplacement`, `TestDeprecatedPackagesMatchesInstalledOnesAndCarriesTheReplacement` |
| Config edited through the typed editor (`configsvc.Service.Set`, a string and a float) before a launch, which BepInEx reads and rewrites: the plugin logs both values and the file keeps them | launch (a) `mods.config`, `mods.config-float`, then `mods.config-file`; `internal/configsvc` tests |

## Launch

| Item | Covered by |
| --- | --- |
| BepInEx loads every plugin: its count equals its Loading lines | launch (a) `launch.count`, `mods.guids` |
| The Console shows levels and sources live: two reads of `launchsvc.Service.Lines` during a run, the second with more heartbeats, warnings and an error from the probe's source and lines from `BepInEx` | launch (a) `launch.console`; `internal/launch` parser tests |
| Stop ends the game politely: the game's process is gone before anything is reaped, Unity logged its clean-exit statistics, Mortar is idle and the game folder is purged | launch (a) `stop.a` |
| A natural exit returns Mortar to idle, records the run and leaves the game folder purged | launch (b) `launch.exit` |
| Run summary and crash analysis of a plugin that throws in `Awake`: Problems names the probe package and its exception; the run records the BepInEx version and its errors | launch (b) `launch.crash`; `TestSummarizeReadsTheBepInExVersion`, `TestAnalyze` |

## Updates

| Item | Covered by |
| --- | --- |
| Update all | offline `updates.all` |
| Update everywhere | offline `updates.everywhere` |
| Roll back with the update's undo | offline `updates.rollback` |
| A save backup before the update, holding the `LCSaveFile1` the game itself wrote, and restored byte for byte | launch (b) writes the save (`saves.fixture`); offline `updates.backup`, `updates.restore`; `internal/backup` tests |

## Sharing

| Item | Covered by |
| --- | --- |
| Share link imported into a new profile through the window's calls: the same packages | offline `share.link` |
| `.mortar` export and import | offline `share.mortar` |
| LAN send to a second paired sandbox, carrying local packages and configs | offline `share.lan`; `internal/lan` loopback tests |
| r2modman code export and import, BepInExPack listed (r2modman installs only what a code lists) | offline `share.r2code` with `MORTAR_REGRESS_R2_EXPORT=1` (publishes to thunderstore.io); `TestExportCodeCarriesTheConfigFolder`; import of a real code: `MORTAR_REGRESS_R2_CODE`, the third launch |
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
| Two names in one archive differing only in case are refused, naming the file: Windows keeps one and Wine finds either, so no copy is sure to be the one that loads | `TestCaseCollision`, `TestInstallArchive` |
| A plugin path past 260 characters as Wine sees it | launch (a) `edge.longpath` |
