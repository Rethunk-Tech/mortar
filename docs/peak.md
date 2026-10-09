# PEAK: research for the sixth catalog game

PEAK is a new, fast-growing co-op climbing game on the BepInEx 5 path (ranking: [design.md](design.md)). Every id below was read on 2026-10-09 from the named source.

- **Discovery:** Steam app `3527290` (Steam's store search API; r2modman's ecosystem schema agrees). The executable is `PEAK.exe` and the Steam folder `PEAK`; the game is Windows-only, so on Linux it runs through Proton. Hero art is Steam's `library_hero.jpg`, which answers 200.
- **Loader:** the community's own pack, `BepInEx-BepInExPack_PEAK` 5.4.75301 (Thunderstore API; the only BepInExPack listed in the `peak` community). The zip holds `BepInExPack_PEAK/winhttp.dll`, `doorstop_config.ini` and `.doorstop_version` 4.4.1, the shape the loader already installs.
- **Mods:** Thunderstore community `peak` (r2modman folder `PEAK`, data folder `PEAK_Data`). No Nexus or GitHub source is listed: Nexus answers 403 to automated reads, so its game id was not verified.
- **Saves:** PCGamingWiki lists no save location, so the entry has no `saves` role and the Saves tab stays empty.
- **Player.log:** `{localLow}/LandCrab/PEAK/Player.log`, built from the registry key PCGamingWiki gives (`HKCU\Software\LandCrab\PEAK`), which Unity names after the company and product; the file itself was not read from an install.
- **Graphics API:** r2modman's most-reacted game-specific issue is "PEAK - r2modman is forcing vulkan" ([#1842](https://github.com/ebkr/r2modmanPlus/issues/1842)): Steam's first launch option carries `-force-vulkan`, and the BepInExPack_PEAK readme says to add `-dx12` if PEAK crashes on startup. The catalog entry's `graphics` block offers DirectX 12 (`-dx12`, recommended) and Vulkan (the game's default); Play asks once, and again after an early exit under Vulkan.
- **Regress:** `scripts/selftest.sh regress --fake peak` (see [repo.md](repo.md) for what it does and does not show).
