# PEAK: research

Co-op climbing game on the BepInEx 5 path. Ids come from the named source.

- **Discovery:** Steam app `3527290` (Steam's store search API; r2modman's ecosystem schema agrees). Executable `PEAK.exe`, Steam folder `PEAK`. Windows-only, so Proton on Linux. Hero art is Steam's `library_hero.jpg`, which answers 200.
- **Loader:** the community's own pack `BepInEx-BepInExPack_PEAK` 5.4.75301 (Thunderstore API; the only BepInExPack in the `peak` community), holding `BepInExPack_PEAK/winhttp.dll`, `doorstop_config.ini` and `.doorstop_version` 4.4.1.
- **Mods:** Thunderstore community `peak` (r2modman folder `PEAK`, data folder `PEAK_Data`). No Nexus or GitHub source: Nexus answers 403 to automated reads, so its game id was not verified.
- **Saves:** campfire saves exist since the "Play It Your Way" patch, in `{localLow}/LandCrab/PEAK` (the catalog's `saves` role); PCGamingWiki's "no save data" is outdated. The exact save file name is unverified.
- **Player.log:** `{localLow}/LandCrab/PEAK/Player.log`, built from PCGamingWiki's registry key `HKCU\Software\LandCrab\PEAK` (Unity names the folder after company and product); not read from an install.
- **Graphics API:** r2modman's most-reacted game-specific issue is "PEAK - r2modman is forcing vulkan" ([#1842](https://github.com/ebkr/r2modmanPlus/issues/1842)): Steam's first launch option carries `-force-vulkan`, and the BepInExPack_PEAK readme says to add `-dx12` if PEAK crashes on startup. The catalog's `graphics` block offers DirectX 12 (`-dx12`, recommended) and Vulkan (the game's default).
- **Regress:** `scripts/selftest.sh regress --fake peak` (what it shows: [repo.md](repo.md)).
