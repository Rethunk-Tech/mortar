# Valheim: research for the third game

Valheim is the BepInEx 5 game with the largest modding base after Lethal Company. Sources: the Thunderstore API (`/api/cyberstorm/community/`, `/api/experimental/package/`), the Nexus Mods v2 GraphQL API, r2modman's ecosystem schema (`ebkr/r2modmanPlus` `src/assets/data/ecosystem.json`), MO2's `modorganizer-basic_games` `games/game_valheim.py`, Vortex's `Nexus-Mods/game-valheim` `src/common.ts`, and the `denikson-BepInExPack_Valheim` 5.4.2351 zip. Figures are from 2026-10-06.

| Community | Thunderstore packages | Thunderstore downloads | Nexus mods | Loader |
| --- | ---: | ---: | ---: | --- |
| Lethal Company (supported) | 53,949 | 1,474,087,767 | 184 | BepInEx 5 |
| Valheim | 13,546 | 180,993,369 | 3,022 | BepInEx 5 (own pack) |
| Risk of Rain 2 | 7,662 | 501,729,605 | not on Nexus | BepInEx 5 (own pack) |
| R.E.P.O. | 7,216 | 379,524,318 | 144 | BepInEx 5 |
| H3VR | 6,444 | 71,323,215 | not found | BepInEx 5 |
| Content Warning | 1,350 | 28,803,713 | 12 | BepInEx 5 |

Valheim leads on mod count on both sites; Risk of Rain 2 leads on downloads.

- **Discovery:** Steam app `892970`; r2modman also lists the Xbox build and the dedicated server (`896660`). Steam on Linux installs the native build (`valheim.x86_64`); Mortar runs the Windows build (`valheim.exe`, the catalog marker) through Proton, so the player sets a Proton compatibility tool in the game's Steam properties first. The native build would need Doorstop's `libdoorstop_x64.so` through `LD_PRELOAD`, which the pack's `start_game_bepinex.sh` does and Mortar does not.
- **Loader:** the community's own pack, `denikson-BepInExPack_Valheim` (maintained by Azumatt, Vapok and Margmas; BepInEx 5.4.23.5 with patches, Doorstop 4 `winhttp.dll`, `.doorstop_version` 4.4.0). The generic `BepInEx-BepInExPack` is not listed in the community. The catalog names the pack in the loader's `package`. Its files sit under `BepInExPack_Valheim/` in the zip, which `InstallPack` finds by the folder holding the preloader. Its `BepInEx.cfg` hooks `UnityEngine.CoreModule` `GameObject..cctor`. Nexus mirrors the pack as mod 3605.
- **Mods:** Thunderstore community `valheim`, Nexus domain `valheim` (game id 3667), GitHub topic `valheim-mod` (138 repositories). Mods name the pack as a dependency, and Mortar skips that dependency because the loader install provides it. Jotunn (`ValheimModding-Jotunn`) is the shared library most content mods depend on.
- **Saves:** `{localLow}/IronGate/Valheim/` (inside `compatdata/892970/pfx` under Proton; `~/.config/unity3d/IronGate/Valheim/` for the native build). Characters are single files, `characters_local/<name>.fch`, beside a `.fch.old` backup. Worlds are pairs, `worlds_local/<name>.fwl` (metadata, seed) and `<name>.db` (terrain and buildings), with `.old` and timestamped `_backup_` copies. Steam Cloud saves live in Steam's `userdata/<id>/892970/remote/` instead. The catalog lists characters (`saveFiles: ["*.fch"]`). A world is two files, and a save is one file or one folder, so worlds are not listed.
- **Player.log:** `{localLow}/IronGate/Valheim/Player.log`.
- **Bridge:** `mortar-bepinex-bridge` targets netstandard2.1 against BepInEx.Core 5.4.21 and uses only `Application.version`, `Application.quitting` and `SceneManager`. It needs no change for Valheim's Mono runtime.
- **Logs:** the pack's `Chainloader startup complete` line ends with counts, which `Ready` still matches. Its plugin lines append the GUID after the bracketed name. Its duplicate-GUID skip reads differently from upstream's `Skipping [X] because a newer version exists (...)`, so that analyzer finding does not fire.
- **Unmeasured:** a real Proton launch of a modded profile. Mortar has had no Valheim install to test with.
