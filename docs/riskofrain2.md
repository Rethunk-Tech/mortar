# Risk of Rain 2: research for the fifth catalog game

Risk of Rain 2 is the busiest Thunderstore community of the BepInEx 5 games Mortar lists (ranking: [design.md](design.md)). Every id below was read on 2026-10-09 from the named source.

- **Discovery:** Steam app `632360` (Steam's store search API; r2modman's ecosystem schema lists it, an Epic id and the Steam app `1180760`, the dedicated server, which the catalog does not name). The executable is `Risk of Rain 2.exe` and the Steam folder `Risk of Rain 2`; the game is Windows-only, so on Linux it runs through Proton. Hero art is Steam's `library_hero.jpg`, which answers 200.
- **Loader:** the community's own pack, `bbepis-BepInExPack` 5.4.2122 (Thunderstore API; `RiskofThunder-BepInExPack` is deprecated). The generic `BepInEx-BepInExPack` is not what the community's mods depend on, so the catalog names the pack in the loader's `package`. The zip holds `BepInExPack/winhttp.dll`, `doorstop_config.ini` and `.doorstop_version` 4.0.0.
- **Mods:** Thunderstore community `riskofrain2` (r2modman folder `RiskOfRain2`). No Nexus or GitHub source is listed: Nexus does not carry the game's mods and its API answers 403 to automated reads.
- **Saves:** PCGamingWiki lists only Steam Cloud (`userdata/<id>/632360/remote/UserProfiles/`), outside any path template Mortar resolves, so the entry has no `saves` role and the Saves tab stays empty.
- **Player.log:** `{localLow}/Hopoo Games, LLC/Risk of Rain 2/Player.log`, built from the registry key PCGamingWiki gives (`HKCU\Software\Hopoo Games, LLC\Risk of Rain 2`), which Unity names after the company and product; the file itself was not read from an install.
- **Regress:** `scripts/selftest.sh regress --fake riskofrain2` (see [repo.md](repo.md) for what it does and does not show).
