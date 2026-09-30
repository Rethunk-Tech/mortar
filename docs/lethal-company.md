# Lethal Company: research for the second game

Not in the first release (NOMAD, 2026-09-29); this is the research for the second implementation, kept so it is not redone. Sources: Gale (`Kesomannen/gale`), r2modmanPlus (`ebkr/r2modmanPlus`), the Thunderstore server source (`thunderstore-io/Thunderstore`), and BepInEx 5.4.23.5's release zip.

- **Discovery:** Steam app `1966720`. On Linux it runs through Proton.
- **Mod index:** the chunked listing index at `https://thunderstore.io/c/lethal-company/api/v1/package-listing-index/` (25 gzipped chunks, 50,978 packages, 34.6 MB gzipped, measured 2026-09-29), fetched in parallel and cached. Never the full v1 package list, which is about 332 MB uncompressed.
- **Downloads:** `https://thunderstore.io/package/download/{namespace}/{name}/{version}/`, no auth. Dependencies are `Namespace-Name-Version` strings.
- **Loader:** the `BepInEx-BepInExPack` package. Doorstop's `winhttp.dll` and `doorstop_config.ini` must sit next to the game executable, so, as Gale does (`Kesomannen/gale` `src-tauri/src/profile/launch/mod.rs`, `copy_required_files`), launch copies the profile's top-level loader files and `doorstop_libs` into the game folder. The rest of the profile stays in Mortar's data directory.
- **Launch arguments** depend on the Doorstop version, read from the profile's `.doorstop_version` (default 3): `--doorstop-enable true --doorstop-target <preloader>` for 3.x, `--doorstop-enabled true --doorstop-target-assembly <preloader>` for 4.x.
- **Proton:** the Wine override `winhttp=n,b` is written into `compatdata/1966720/pfx/user.reg`, backed up first, as Gale and r2modman both do. Paths passed to Doorstop use the `Z:` form.
- **Package layout:** follows r2modman's rules. `plugins/`, `patchers/`, `monomod/`, `core/` and `config/` (with or without a `BepInEx/` prefix) map to `BepInEx/<dir>/<Namespace-Name>/`. `config/` keeps no per-mod subfolder. Loose DLLs go to `BepInEx/plugins/<Namespace-Name>/`.
- **r2modman profile codes:** import and export them, since that is how Lethal Company players already share. A code is `#r2modman\n` followed by base64 of an `.r2z` zip, which holds `export.r2x` (YAML: `profileName`, `mods[]` with `name`, `version {major,minor,patch}` and `enabled`) and the `config/` folder. Codes are stored with `POST https://thunderstore.io/api/experimental/legacyprofile/create/` and read with `GET .../legacyprofile/get/{key}/`. The endpoint is marked experimental; Gale uses it too.

**Measure:** on Windows, how BepInEx finds its root when the profile lives outside the game folder (r2modman passes `--r2profile` only on Linux).
