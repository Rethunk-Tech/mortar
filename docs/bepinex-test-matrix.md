# BepInEx test matrix

What proves Mortar's BepInEx 5 support, with Lethal Company as the real game: each row names the automated test or regress step that covers it. Run it with `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress --game lethal-company` (code: `scripts/regress-bepinex.sh`, whose header explains the stages). Each step writes one PASS or FAIL row with its evidence to `matrix.tsv` in the sandbox; any FAIL fails the regress. It needs the copied game, Proton, the network (Thunderstore) and the .NET SDK, whose Roslyn builds the probe plugins in `scripts/bepinex-probe/` against the copied game's assemblies. Suite map: [testing.md](testing.md).

## Launches

A session may start the game 3 times ([testing.md](testing.md#sandbox-game-launches)), so everything that needs no game process runs offline and the rest runs in three launches, using the whole cap (`MORTAR_REGRESS_R2_CODE` is refused with the matrix). (c) and (b) each differ from (a) in one variable, so the `bridge.reachable` rows say which, if either, costs Mortar its bridge:

- **(a)** the base run on the reinstalled latest pack (Doorstop 4), a measured launch of a profile filled with the modpack, the probe packages and two config edits.
- **(c)** the Throw and Quit probes on the latest pack, run first while that pack is pinned: crash analysis, a natural exit, an unmeasured launch, and whether a plugin throwing in `Awake` alone takes the bridge down.
- **(b)** the Save and Quit probes on the older pack pinned (5.4.2100, Doorstop 3), no throwing plugin.

No vanilla launch is run: it takes no launch prefix, so the regress cannot isolate it from Steam.

`bridge.survived.*` are INFO rows: with the default `HideManagerGameObject = false`, Lethal Company's first scene load destroys the loader's manager object and its plugin components, so False is expected and only an unreachable bridge fails.

Rows are item -> test or step id (`launch (a)` and so on name the launch; `offline` needs none).

## Measured launch and live state

| Item | Covered by |
| --- | --- |
| Patcher times the measured launch | launch (a) `startup.patcher` |
| Startup report written and consumed | launch (a) `startup.report`; `TestPrepareStartupArmsTheBepInExPatcherOnlyForAMeasuredLaunch` |
| Startup view names rows by package | launch (a) `startup.mortar`; `TestBepInExReportRowsBecomeTheirPackages` |
| Start measuring in game | launch (a) `perf.start` |
| Summary 10 s later, saved and listed | launch (a) `perf.summary`; `TestPerfRowsNameAndAddUpEachPackagesPlugins` |
| Wrapping raises no new exceptions | launch (a) `perf.stable` |
| An unmeasured launch measures nothing | launch (c) `perf.unmeasured` |
| `launch:live` reaches scene MainMenu | launch (a) `live.scene` |
| Loaded badges | launch (a) `live.badges` |
| Version last played recorded | after launch (a) `lastplayed.version` |

## Loader

| Item | Covered by |
| --- | --- |
| Fresh install | offline `loader.fresh`; `TestInstallPackAndDoorstopFiles`, `TestLoaderContract` |
| Pin an older pack over a newer one | offline `loader.pin`; `TestAnOlderPackOverANewerOneKeepsNoneOfItsFiles` |
| Pinned pack starts | launch (b) `loader.pin-run` |
| Unpin and install the latest | offline `loader.unpin` |
| Reinstall a version over itself | offline `loader.reinstall` |
| Reinstalled latest pack starts | launch (a) `loader.reinstall-run` |
| Per-Doorstop files and flag spelling | `loader.pin`, `loader.unpin`, `loader.reinstall`, `loader.reinstall-run`, `loader.pin-run`; `TestLaunchArgs` |
| Uninstall leaves the game folder as before | `stop.a`, `launch.exit`, `loader.pin-exit`; `TestLoaderContract`, `TestLaunchArgs` |
| Wine `winhttp` override in `user.reg` | `TestEnsure` (`internal/wineoverride`); launches (a), (c) and (b) |

## Mods

| Item | Covered by |
| --- | --- |
| Modpack install | offline `modpack.install` |
| Load order | offline `modpack.loadorder` |
| Plugin GUIDs match what BepInEx loaded | `mods.guids` (`TestRealBepInExRun` in `internal/dotnet`, run with `-bepinex-profile <profile folder>`) |
| Problems shows only true problems | `problems.true`; `TestThePlayerLogBelongsOnlyToTheProfileThatRanLast` |
| Duplicate GUIDs flagged | offline `mods.dupguid`, launch (a) `mods.dupguid-runtime`; `TestEnabledPackagesGiveAFilePackageItsManifestVersion` |
| Deprecated packages suggest a replacement | offline `mods.deprecated`; `TestDeprecatedListsPackagesAndTheirNamedReplacement`, `TestDeprecatedPackagesMatchesInstalledOnesAndCarriesTheReplacement` |
| Config edits reach the plugin | launch (a) `mods.config`, `mods.config-float`, `mods.config-file`; `internal/configsvc` tests |

## Launch

| Item | Covered by |
| --- | --- |
| Plugin count equals Loading lines | launch (a) `launch.count`, `mods.guids` |
| Console shows levels and sources live | launch (a) `launch.console`; `internal/launch` parser tests |
| Stop ends the game politely | launch (a) `stop.a` |
| Natural exit returns Mortar to idle | launch (c) `launch.exit`, launch (b) `loader.pin-exit` |
| Bridge answers a status query at the main menu | `bridge.reachable.a`, `.c`, `.b` |
| Crash analysis of a throwing plugin | launch (c) `launch.crash`; `TestSummarizeReadsTheBepInExVersion`, `TestAnalyze` |

## Updates

| Item | Covered by |
| --- | --- |
| Update all | offline `updates.all` |
| Update everywhere | offline `updates.everywhere` |
| Roll back | offline `updates.rollback` |
| Save backup before the update restores byte for byte | launch (b) `saves.fixture`; offline `updates.backup`, `updates.restore`; `internal/backup` tests |

## Sharing

| Item | Covered by |
| --- | --- |
| Share link import | offline `share.link` |
| `.mortar` export and import | offline `share.mortar` |
| LAN send to a paired sandbox | offline `share.lan`; `internal/lan` loopback tests |
| r2modman code export and import | offline `share.r2code` with `MORTAR_REGRESS_R2_EXPORT=1` (publishes to thunderstore.io); `TestExportCodeCarriesTheConfigFolder` |
| Gale and r2modman formats | `internal/pack` tests |

## Package layout edge cases

| Item | Covered by |
| --- | --- |
| Nested plugin folders | launch (a) `edge.nested`; `TestRoute` |
| Folders no rule names are flattened | launch (a) `edge.flatten`, `edge.flatten-probe`; `TestRoute`, `TestLayoutGoldens` |
| Patchers | launch (a) `edge.patcher`, `edge.patchers` |
| `config` flat, `plugins` per package | launch (a) `edge.config-placement`; `TestRoute` |
| A package with its own `BepInEx/` layout | launch (a) `edge.own-layout`; `TestRoute` |
| Case: `BEPINEX/Plugins` | launch (a) `edge.case`; `TestRoute`, `TestLayoutGoldens` |
| Two names differing only in case are refused | `TestInstallArchive`; [go-archive `archive_test.go`](https://github.com/Rethunk-Tech/go-archive/blob/v0.1.0/archive_test.go) |
| Plugin path past 260 characters | launch (a) `edge.longpath` |
