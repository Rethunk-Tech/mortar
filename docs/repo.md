# R.E.P.O.: research

Unity co-op game on the BepInEx 5 path, the nearest relative of Lethal Company. Ids come from the named source; the catalog (`internal/components/components.json`) holds the wiring.

- **Discovery:** Steam app `3241660` (Steam's `storesearch?term=R.E.P.O.`; r2modman's ecosystem schema `distributions` agrees). Executable `REPO.exe`, Steam folder `REPO` (the schema's `exeNames`, `steamFolderName`). Windows-only, so Proton on Linux. Hero art is Steam's `library_hero.jpg`, which answers 200.
- **Loader:** Thunderstore community `repo` lists the generic `BepInEx-BepInExPack` 5.4.2305 (so no catalog `package` key), holding `BepInExPack/winhttp.dll`, `doorstop_config.ini` and `.doorstop_version` 4.5.0.
- **Mods:** Thunderstore community `repo` (r2modman folder `REPO`, data folder `REPO_Data`). No Nexus or GitHub source: Nexus answers 403 to automated reads, so its game id was not verified.
- **Saves and log:** PCGamingWiki lists saves in `%USERPROFILE%\AppData\LocalLow\semiwork\Repo\saves\`, so `Player.log` is `{localLow}/semiwork/Repo/Player.log`. The layout inside `saves\` was not read from an install; with no `saveFiles` pattern a save is a folder holding a file of its own name.
- **Regress:** `scripts/selftest.sh regress --fake repo` runs Mortar against a stand-in install and a dummy window: discovery, the pack install, the Doorstop pair beside `REPO.exe` at launch, the hidden display, and a game folder that hashes as before after the purge. It does not show BepInEx loading inside the real game, which this machine cannot run.
