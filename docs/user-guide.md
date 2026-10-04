# Mortar user guide

Mortar manages mods for Stardew Valley. It finds the game, installs SMAPI (the mod loader every Stardew mod needs), keeps each set of mods in its own profile and starts the game with the profile you pick. This guide is for players. For how Mortar works inside, see [architecture.md](architecture.md).

## Install

Download from [Releases](https://github.com/Rethunk-AI/mortar/releases/latest) or the [download page](https://mortar.rethunk.tech/download/).

### Windows

Use `mortar-amd64-installer.exe` on most PCs, or `mortar-arm64-installer.exe` on an ARM PC. The installer puts Mortar in `%LOCALAPPDATA%\Programs\Mortar` and adds Start Menu and Desktop shortcuts. `mortar-windows-amd64.exe` and `mortar-windows-arm64.exe` are the bare program: run them from any folder, with nothing installed.

The Windows builds are not code-signed, so SmartScreen shows "Windows protected your PC" the first time. Choose **More info**, then **Run anyway**.

### Linux

- **AppImage** (`mortar-linux-x86_64.AppImage` or `mortar-linux-aarch64.AppImage`): make it executable (`chmod +x`, or the file's Properties in your file manager) and run it. It needs nothing else installed.
- **Flatpak** (`mortar-linux-x86_64.flatpak`, x86-64 only): `flatpak install --user mortar-linux-x86_64.flatpak`. The Flatpak runs in a sandbox, which has these limits:
  - Mortar cannot see folders outside the ones the Flatpak is allowed to read (Steam, Heroic and Lutris folders and the usual game folders are allowed). A game installed somewhere else stays invisible until you run `flatpak override --user --filesystem=<folder> tech.rethunk.Mortar`.
  - **Add this profile to Steam** is refused, because Steam outside the sandbox cannot start Mortar inside it. The error shows the `flatpak run tech.rethunk.Mortar --play=<game>/<profile>` command to use instead.
  - With Flatpak Steam, Steam's own sandbox cannot read Mortar's profiles until Mortar grants it access. Mortar shows a **Grant access** button for this, in the game's settings.
  - Updates come through Flatpak, not from Mortar.
- **`.deb`, `.rpm` and Arch package**: install with your package manager (for example `sudo apt install ./mortar_*_amd64.deb`, `sudo dnf install ./mortar-*.rpm`, `sudo pacman -U mortar-*.pkg.tar.zst`). Updates come from the package manager too: Settings › Updates says "Updated by your package manager".

## First run

Mortar opens with a welcome screen and lists the launchers it found (Steam, GOG, Heroic, Lutris). Continue, then open Stardew Valley. Setup has three steps.

1. **Game folder.** Mortar shows where it found the game. If it did not find it, choose **Choose folder…** and pick the folder that holds the game. **Change folder…** and **Rescan** are there if the wrong copy was picked.
2. **Install SMAPI.** Mortar downloads SMAPI and puts it in the game folder. **Skip for now** leaves it for later. On Windows with Steam, Steam starts the game without SMAPI unless SMAPI is in the game's Steam launch options. Mortar shows the line to paste (right-click Stardew Valley in Steam, **Properties**, **Launch Options**), or choose **Set it in Steam for me** with Steam closed.
3. **First profile.** Choose **Start empty** (a profile with only SMAPI), **Import from the game's Mods folder** (copies the mods you already have; nothing in the game folder is moved or changed), or **From a shared link**.

## Sign in to Nexus Mods

Mortar downloads from Nexus with your account, so it needs a personal API key.

1. Sign in on nexusmods.com and open your account **Settings**, then **API Keys**. Copy the personal API key.
2. In Mortar open **Settings › Nexus account**, paste the key into **Personal API key** and choose **Sign in**. The key is kept in your system keyring.

Mortar then shows your name and whether the account is Premium or Free. Free accounts click **Mod Manager Download** on the Nexus page once for each file; Mortar opens each page in turn. Premium accounts download directly from inside Mortar. Mortar only downloads what Nexus allows and never re-hosts mod files.

To make that button work, turn on **Handle "Mod Manager Download" links** in **Settings › Downloads**.

## Browser extension

The extension marks Nexus Mods pages with what your profile already has and sends Mod Manager Download clicks to Mortar. It is optional. **Settings › Downloads** has a **Download the extension** button; the same file is `mortar-browser-extension.zip` on the release. Unzip it first.

- **Chrome, Edge or another Chromium browser:** open `chrome://extensions`, turn on Developer mode, choose **Load unpacked** and pick the unzipped folder.
- **Firefox:** open `about:debugging`, choose **This Firefox**, then **Load Temporary Add-on** and pick `manifest.json` in the unzipped folder. Firefox removes a temporary add-on when it restarts, so repeat this after each restart.

## Profiles

A profile is one set of mods. Switching profiles never touches the game folder's own `Mods` folder.

- **Create:** **New profile…** on the Profiles page.
- **Switch:** **Switch profile**, or pick one from the Profiles page. Play always starts the open profile.
- **Rename, duplicate, delete:** from the profile's menu. Deleted profiles wait in **Recently deleted** for 30 days, with **Restore** and **Delete permanently**.
- **Share as a link:** **Share profile**, then **Copy link**. A link names the mods and their Nexus or GitHub files, not the files. A profile too large for a link says so.
- **Share as a file:** the same dialog saves a `.mortar` file. It also carries the mods' settings and your notes, and suits large profiles.
- **Import:** **Import** on the Profiles page, then **From a link or file…**, **From the game's Mods folder…** or **From a backup…**. Mortar shows what the profile holds first, then installs or queues what is missing.

## Play

Press **Play** in Mortar. It applies the profile's mod list, then starts the game through SMAPI.

On Windows, Steam's own **Play** button is different: it starts SMAPI with the game's own `Mods` folder, not a profile. To play a profile, press Play in Mortar, or use a profile shortcut. A profile's menu has **Add a shortcut that plays this profile** (a Start Menu shortcut on Windows, a launcher entry on Linux) and **Add this profile to Steam**, which adds the profile to your Steam library as a non-Steam game. Close Steam before adding.

## Updates

- **Mortar:** the Mortar menu's **Check for updates**, or **Settings › Updates**, which can also include beta releases. An update downloads and applies when you close Mortar, or choose **Restart now**.
- **Mods:** updates show on each profile's mod list, and **Settings › Updates** has **Review mod updates**.
- **SMAPI:** Mortar tells you when a new SMAPI is out. The game's settings have its version and update.

## Save backups and restore

Mortar zips your saves before it changes them. **Back up saves before Play** (Settings, then the game's **Save backups**) backs up every time the game starts, and **Scheduled save backups** does it on a timer. Backups also happen before a profile update.

To restore, open the **Saves** tab, choose **Save backups…**, pick a backup and choose **Restore**. Mortar names the saves it will overwrite and backs up the current Saves folder first. Stop the game before restoring. **Open backups folder** shows the zips; **Backup location** in the game's settings moves where new ones go.

## Where your data lives

| | Windows | Linux |
| --- | --- | --- |
| Mortar's data (profiles, mods, downloads, backups, settings, logs) | `%LOCALAPPDATA%\Mortar` | `$XDG_DATA_HOME/mortar`, usually `~/.local/share/mortar` |
| Your saves | `%APPDATA%\StardewValley\Saves` | `~/.config/StardewValley/Saves` |

The Mortar menu's **Open data folder** opens the first row.

To move the data, stop the game and open **Settings › Storage**, then **Move…**. Mortar copies everything to a folder you pick, checks the copy, removes the old one and restarts. The new folder must be empty and have enough free space. A file named `portable` beside the Mortar program keeps all data in a `data` folder next to it (not in a Flatpak, and not where the folder is read-only).

## Uninstall

- **Windows:** use Settings › Apps (or the uninstaller in the install folder). It removes the program, shortcuts, the start-at-login entry, the Start Menu **Mortar** folder with your profile shortcuts, and the profiles you added to Steam. Your data folder stays, with all profiles, mods and settings. Delete `%LOCALAPPDATA%\Mortar` by hand to remove them. SMAPI and the game folder are not touched.
- **Linux:** first run `mortar --release-links && mortar uninstall-cleanup` (for the AppImage, `./Mortar.AppImage --release-links && ./Mortar.AppImage uninstall-cleanup`; for the Flatpak, `flatpak run tech.rethunk.Mortar` with the same two arguments). That gives nxm:// links back to the app that had them, removes the browser extension's connection, the start-at-login entry, your profile shortcuts, the profiles you added to Steam, and the menu entry, icon and file types an AppImage or the portable binary added. Then remove the package, the AppImage or the Flatpak. Removing the `.deb`, `.rpm` or Arch package with `sudo` runs those two steps for you, for your own account only; other accounts on the machine, or a removal from a software centre, need the commands. Delete the data folder by hand to remove profiles and mods.

Close Steam before uninstalling so the Steam cleanup sticks: Steam rewrites its shortcut list when it exits.

## Troubleshooting

- **Run checks:** **Settings › About › Diagnostics › Run checks…** inspects the data folder, games, Nexus links and the extension, and **Repair** fixes what it can.
- **Logs:** Mortar's own log is `mortar.log` in the data folder; `crash.log` is there too after a crash. Each profile's SMAPI log is shown on its log view, and every run is kept.
- **Report a bug:** the Mortar menu's **Report a bug** opens a prefilled GitHub issue. Leave **Attach diagnostics** ticked to save a zip (checks, logs, settings and recent runs, with secrets and your home folder removed) and add Mortar's checks to the issue; drag the zip into the issue before posting. **Save diagnostics…** in the Mortar menu and in Settings › About › Diagnostics writes the same zip.
- **Share a SMAPI log:** on the profile's log view choose **Share log…**. It uploads the log to smapi.io/log and gives you a link to send to whoever is helping.
- **A mod will not load:** the mod's page on Nexus lists what it needs. Mortar's problem checks warn about missing requirements and duplicates and never block Play.
