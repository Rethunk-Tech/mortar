# Mortar user guide

Mortar manages mods for the PC games its catalog enables, starting with Stardew Valley. It finds the game, installs its mod loader (SMAPI for Stardew Valley), keeps each set of mods in its own profile and starts the game with the profile you pick. This guide is for players. For how Mortar works inside, see [architecture.md](architecture.md).

## Install

Download from [Releases](https://github.com/Rethunk-Tech/mortar/releases/latest).

### Windows

Use `mortar-amd64-installer.exe` on most PCs, or `mortar-arm64-installer.exe` on an ARM PC. The installer puts Mortar in `%LOCALAPPDATA%\Programs\Mortar` and adds Start Menu and Desktop shortcuts. `mortar-windows-amd64.exe` and `mortar-windows-arm64.exe` are the bare program: run them from any folder, with nothing installed.

The Windows builds are not code-signed, so SmartScreen shows "Windows protected your PC" the first time. Choose **More info**, then **Run anyway**.

With [Scoop](https://scoop.sh): `scoop bucket add mortar https://github.com/Rethunk-Tech/scoop-bucket`, then `scoop install mortar/mortar`. Scoop then handles updates and Mortar's own updater stays off.

### Linux

- **AppImage** (`mortar-linux-x86_64.AppImage` or `mortar-linux-aarch64.AppImage`): make it executable (`chmod +x`, or the file's Properties in your file manager) and run it. It needs Ubuntu 24.04, Debian 13, Fedora 39 or a newer Linux (glibc 2.38 or newer) and nothing else installed. On an older system it says so and does not start; use the Flatpak there, which brings its own libraries.
- **Portable program** (`mortar-linux-amd64` or `mortar-linux-arm64`): the bare program, to run from any folder. It uses the system's GTK 4 and WebKitGTK 6.0 (`libwebkitgtk-6.0-4` on Debian and Ubuntu, `webkitgtk6.0` on Fedora, `webkitgtk-6.0` on Arch), so it needs the same Linux versions as the AppImage. Make it executable and run it. Like the AppImage, it updates itself in place, and on first run it adds Mortar to your applications menu and takes nxm:// links.
- **Flatpak**: `flatpak install --user https://mortar.rethunk.tech/packages/flatpak/tech.rethunk.Mortar.flatpakref` adds Mortar's signed Flatpak repository, so `flatpak update` keeps it current; each release also carries `mortar-linux-x86_64.flatpak` and `mortar-linux-aarch64.flatpak` to install once, without updates. The Flatpak runs in a sandbox, which has these limits:
  - Mortar cannot see folders outside the ones the Flatpak is allowed to read (Steam, Heroic and Lutris folders and the usual game folders are allowed). A game installed somewhere else stays invisible until you run `flatpak override --user --filesystem=<folder> tech.rethunk.Mortar`.
  - **Add this profile to Steam** is refused, because Steam outside the sandbox cannot start Mortar inside it. The error shows the `flatpak run tech.rethunk.Mortar --play=<game>/<profile> --steam-session` command to use instead.
  - With Flatpak Steam, Steam's own sandbox cannot read Mortar's profiles until Mortar grants it access. Mortar shows a **Grant access** button for this, in the game's settings.
  - Updates come through Flatpak, not from Mortar.
- **Homebrew on Linux**: `brew install rethunk-tech/tap/mortar` installs the AppImage, and `brew upgrade` updates it.
- **`.deb`, `.rpm` and Arch package files**: install with your package manager (for example `sudo apt install ./mortar_*_amd64.deb`, `sudo dnf install ./mortar-*.rpm`, `sudo pacman -U mortar-*.pkg.tar.zst`). A package installed this way does not update itself; add the package repository below to get updates.

### Linux package repository

On Debian, Ubuntu, Fedora and Arch, install Mortar from its package repository and it updates with the rest of your system; Settings › Updates says "Updated by your package manager". The repository is signed by the key "Mortar packages <opensource@rethunk.tech>", fingerprint `D8B1 C4C2 05C5 FB36 33CB 8B99 00B7 5959 B637 DCB5`.

Debian and Ubuntu (amd64, arm64):

```sh
sudo apt update && sudo apt install curl
sudo curl -fsSLo /usr/share/keyrings/mortar-archive-keyring.gpg https://mortar.rethunk.tech/packages/mortar-archive-keyring.gpg
sudo curl -fsSLo /etc/apt/sources.list.d/mortar.list https://mortar.rethunk.tech/packages/mortar.list
sudo apt update && sudo apt install mortar
```

Fedora (x86_64, aarch64); dnf asks once to import the key, so check its fingerprint against the one above:

```sh
sudo curl -fsSLo /etc/yum.repos.d/mortar.repo https://mortar.rethunk.tech/packages/mortar.repo
sudo dnf install mortar
```

Arch (x86_64, aarch64):

```sh
curl -fsSL https://mortar.rethunk.tech/packages/mortar-archive-keyring.asc | sudo pacman-key --add -
sudo pacman-key --lsign-key D8B1C4C205C5FB3633CB8B9900B75959B637DCB5
curl -fsSL https://mortar.rethunk.tech/packages/pacman.conf | sudo tee -a /etc/pacman.conf
sudo pacman -Sy mortar
```

## First run

Mortar opens with a welcome screen and lists the launchers it found (Steam, GOG, Heroic, Lutris, Bottles). Continue, then open Stardew Valley. Setup has three steps.

1. **Game folder.** Mortar shows where it found the game. If it did not find it, choose **Choose folder…** and pick the folder that holds the game. **Change folder…** and **Rescan** are there if the wrong copy was picked.
2. **Install SMAPI.** Mortar downloads SMAPI and puts it in the game folder. **Skip for now** leaves it for later. On Windows with Steam, Steam starts the game without SMAPI unless SMAPI is in the game's Steam launch options. Mortar shows the line to paste (right-click Stardew Valley in Steam, **Properties**, **Launch Options**), or choose **Set it in Steam for me** with Steam closed.
3. **First profile.** Choose **Start empty** (a profile with only SMAPI), **Import from the game's Mods folder** (copies the mods you already have; nothing in the game folder is moved or changed), or **From a shared link**.

## Sign in to Nexus Mods

Mortar downloads from Nexus with your account, so it needs a personal API key.

1. Sign in on nexusmods.com and open your account **Settings**, then **API Keys**. Copy the personal API key.
2. In Mortar open **Settings › Accounts**, find the Nexus Mods section, paste the key into **Personal API key** and choose **Sign in**. The key is kept in your system keyring.

Mortar then shows your name and whether the account is Premium or Free. Free accounts click **Mod Manager Download** on the Nexus page once for each file; Mortar opens each page in turn. Premium accounts download directly from inside Mortar. Mortar only downloads what Nexus allows and never re-hosts mod files.

To make that button work, enable **Handle "Mod Manager Download" links** in **Settings › Downloads**.

## Browser extension

The extension marks Nexus Mods pages with what your profile already has and sends Mod Manager Download clicks to Mortar. It is optional. It has its own releases at <https://github.com/Rethunk-Tech/mortar-browser-extension/releases/latest>; **Settings › Downloads** has a **Download the extension** button for `mortar-browser-extension.zip` there. Unzip it for Chrome and Edge. When Mortar and the extension are too far apart in version to talk, the extension says which one to update, and Settings › Downloads says **The browser extension is too old for this Mortar** (or too new).

- **Chrome, Edge or another Chromium browser:** open `chrome://extensions`, turn on Developer mode, choose **Load unpacked** and pick the unzipped folder.
- **Firefox:** download `mortar-browser-extension.xpi` from the extension's [latest release](https://github.com/Rethunk-Tech/mortar-browser-extension/releases/latest) (**Settings › Downloads** links it too) and open it in Firefox, or drop it on `about:addons`. It is signed by Mozilla, so it stays installed. A release without the `.xpi` needs the zip instead: open `about:debugging`, choose **This Firefox**, then **Load Temporary Add-on** and pick `manifest.json` in the unzipped folder; Firefox removes a temporary add-on when it restarts.

## Find and add mods

The **Browse** tab searches every site the game's mods come from at once: for Stardew Valley, Nexus Mods and GitHub. **All sources** is the default, so where an author published a mod does not matter; results from each site are interleaved, each site keeping its own ranking. A mod that several sites list shows as one card, and a badge on the card picks the site to install from. The game's settings (the **Mods** page) have a **Preferred source** row that sets which site a card installs from first. The chips next to **All sources** narrow the search to one site. If a site does not answer, the result count says which one and the rest still show.

1. Open the **Browse** tab. With nothing typed, each site lists its top mods, so the page is never empty.
2. Type a name to search. Results update as you type.
3. Choose **Add** (or **Download** on Nexus for a Premium account) on a card. The card shows the install's progress, then **In this profile**.

- **Nexus Mods:** Premium accounts download straight into the profile. Free accounts open the mod's files page and use **Mod Manager Download**, which the browser extension or the nxm link hands to Mortar.
- **GitHub:** **Add** puts the latest release in the profile. Mortar uses your GitHub CLI login (`gh auth login`) when it is present, which raises GitHub's rate limits; without it, it works anonymously.
- **Grid or list:** the toggle at the left of the search row switches between cards and rows.

### Narrow and sort the results

Under the search row:

- **Include categories** keeps only mods in any of the categories you pick. **Exclude categories** drops mods in any of them. A site with no categories (GitHub, or Nexus while you are signed out) turns both off, and a search that includes categories leaves such a site out.
- **Sort** orders each site's results: best match, most downloaded, most endorsed, recently updated or name. A site that cannot sort a given way keeps its own order.
- **Show** has one row for each kind of mod you may not want in the way: **Mods in this profile**, **Obsolete** and **Broken**. Each is **Off** (shown as usual), **Gray out** (dimmed, still there) or **Hide** (left out, and counted as "N hidden" beside the result count). **Broken** needs the game to have a compatibility list, which Stardew Valley has.

Mods Mortar already holds in the profile show **In this profile**, whichever site they came from, including a mod you installed from an archive when its manifest names the same Nexus or GitHub page. A loader's helper mod that Mortar installs for you shows **Installed by Mortar**.

## Modrinth and itch.io

Modrinth and itch.io are mod sites Mortar can search and install from for a game that lists them. They show as chips beside **All sources** on that game's Browse tab and take part in the merged search like any other site. A game whose catalog entry names one gets its chip; Stardew Valley's does not.

- **Modrinth** needs no account. **Add** installs the project's newest version together with the projects it requires, and Mortar checks each downloaded file against the SHA-512 Modrinth publishes.
- **itch.io** needs your own API key, which keeps itch.io's search working for you:
  1. On itch.io open your account **Settings**, then **API keys**, and create a key.
  2. In Mortar open **Settings › Accounts**, find the itch.io section, paste the key into **API key** and choose **Test**.
  3. Mortar shows your itch.io name once the key works. The key is kept in your system keyring. **Remove key** forgets it.

  Until a key is stored, the itch.io chip is greyed out and its tooltip says it needs an API key. itch.io's search has no sort order and no categories, so **Sort** and the category pickers leave it alone. **Add** downloads the game's first upload that is not a demo or a soundtrack.

## Edit a mod's settings

Mortar edits a mod's config file as a form, so you never open a text file.

1. In the **Mods** tab, select a mod. The details panel opens beside the list; drag its left edge to resize it.
2. Under **Config**, choose **Edit config**. (The **More details** dialog has the same button under **Settings**.)
3. The editor takes over the tab. The mod's config files are listed on the left (`config.json` for a SMAPI mod, each BepInEx `.cfg` for a Lethal Company plugin). Pick one.
4. Change a value. On and off settings are switches, numbers are fields (sliders when the mod gives a small range), choices are drop-downs, colours open a colour picker and lists are chips you add to and remove from. A setting's info button shows the mod's own description of it.
5. Each change saves as you make it. A setting that is not the mod's default shows a **Reset to default** button; **Reset all** puts the whole file back after you confirm. **Search entries** filters the long ones.
6. **Presets** saves the file's values under a name (**Save current as…**) and applies a saved set to any profile's copy. The **X** closes the editor.

Mortar does not save a change while the game runs the profile.

A mod that has a Generic Mod Config Menu also lists **In-game menu**, with the settings that menu shows. A change there says **Applied when the game next starts**: Mortar keeps it and the game applies it when it next starts. Undo in the change history takes such a change back, like any other settings edit.

## Lethal Company and Thunderstore

Lethal Company's mods are Thunderstore packages that run on BepInEx. Mortar installs BepInEx, searches Thunderstore, and keeps each package in the profile, the way it does for Stardew Valley's mods.

- **Browse:** the Browse tab searches Thunderstore's Lethal Company community along with Nexus Mods and GitHub. **Add** puts the package in the profile together with the packages it depends on; a dependency the profile still lacks is flagged on the Load order tab and in the **Before you play** list.
- **Thunderstore links:** the **Install with Mod Manager** button on thunderstore.io opens a `ror2mm://` link. Mortar does not take these links over from another manager until you ask: run `mortar links enable --source thunderstore` (and `mortar links disable --source thunderstore` to hand them back). A package link names no game, so it installs into the open game when that game has a Thunderstore community.
- **Play:** BepInEx reads only the game's own folder, so for each launch Mortar places the profile's BepInEx files beside the game and takes them back when the game exits. Nothing of the profile stays in the game folder, and a crash is finished off at the next start.
- **Settings the game writes:** BepInEx reads and writes its plugins and settings (`BepInEx/config`) in the profile, so the next launch and a shared profile keep them. The game folder gets only `winhttp.dll` and `doorstop_config.ini` for the launch, and they are removed when the game exits.
- **Log:** the profile's BepInEx log is `BepInEx/LogOutput.log` in the profile folder; diagnostics include its last lines.
- **BepInEx launch options:** open the profile's menu, choose **Edit profile**, and find the **BepInEx** group under the launch options. BepInEx's log always streams into Mortar's **Console** tab; **Show the console window** (off by default) also opens BepInEx's own console window, and **Log level** picks what BepInEx writes to its console and log: **Warnings and errors**, **Standard**, **Debug** or **Everything**. Each change saves at once into the profile's `BepInEx/config/BepInEx.cfg` and applies the next time the game starts.

## Pair your computers

Pairing makes two of your own computers trust each other, so a profile sent between them carries its mod files from every source (Nexus, GitHub and Thunderstore) instead of being downloaded again. It needs no account. A computer you have not paired still receives the profile and downloads each mod from its source.

1. On both computers enable **Share profiles on the local network** in **Settings › General › Sharing**.
2. On the first computer choose **Pair a computer**. It shows a code like `ABCD-EFGH`; the code works once and for five minutes.
3. On the second choose **Enter code**, pick the first computer from the nearby list (or type its `host:port` when none is listed) and type the code. Five wrong codes lock the entry from that computer for ten minutes.
4. Both computers now list each other under **Paired computers**; **Unpair** forgets one. From the command line, `mortar lan pair` shows a code and `mortar lan pair --code <code>` enters one.

To send, use **Share…**, choose **Nearby** and pick the computer. The receiver is asked to accept; **Auto-accept from paired computers** in the same settings skips the question.

## Switch games

Click the game's name in the title bar. A menu lists each playable game, the one you opened last first, with a tick on the current one. Choose a game and Mortar opens it on the profile you used last. **All games…** opens Game select, which shows every game, including the ones Mortar did not find.

## Game installs

When a game is installed more than once (Steam and GOG, a Steam beta branch, a copy in another folder), each is a separate install. A profile uses the selected install unless you pin it to one in the game's settings, and two profiles pinned to different installs can run at the same time. The install row shows its store and, on Linux, whether it runs natively or through Proton.

### Bottles

On Linux, Mortar finds a Windows build of the game inside a [Bottles](https://usebottles.com) bottle, from the native or the Flatpak Bottles, when the game sits in the bottle's Steam or GOG folders. The install's store reads **Bottles**.

1. Open the game from **Game select**. Setup lists Bottles among the launchers it read, and a game found in a bottle needs no folder to be chosen. If the bottle keeps the game somewhere unusual, choose its folder with **Choose folder…**; Mortar still runs it in that bottle.
2. For the loader step, Mortar runs SMAPI's own Windows installer inside the bottle, so the bottle holds the Windows SMAPI and not the Linux one.
3. Press **Play**. Mortar starts the game through the bottle with the profile you picked.

The Mortar Flatpak is allowed to read and write the Bottles data folders, native and Flatpak, so installing SMAPI into a bottle works from it.

## Profiles

A profile is one set of mods. Switching profiles never touches the game folder's own `Mods` folder.

- **Create:** **New profile…** on the Profiles page, or **New profile…** in the profile switcher in the title bar. Type a name, then pick **Start from**:
  - **Empty profile** holds only the loader.
  - **Copy of \<the open profile\>** duplicates the profile that is open now, with its mods, their settings and what the mods wrote, and opens the copy.
  - A template (one you saved with **Save as template…**) starts with the template's mods and its launch settings. Mods Mortar does not hold download in the background, and a toast says how many. **Manage templates…** renames or deletes templates.
- **Switch:** **Switch profile**, or pick one from the Profiles page. Play always starts the open profile.
- **Rename, duplicate, delete:** from the profile's menu. Deleted profiles wait in **Recently deleted** for 30 days, with **Restore** and **Delete permanently**.
- **Share as a link:** **Share profile**, then **Copy link**. A link names the mods and their Nexus or GitHub files, not the files. A profile too large for a link says so.
- **Share as a file:** the same dialog saves a `.mortar` file. It also carries the mods' settings and your notes, and suits large profiles.
- **Import:** **Import** on the Profiles page, then **From a link or file…**, **From the game's Mods folder…**, **From a backup…** or **Import…**. Mortar shows what the profile holds first, then installs or queues what is missing.
- **Import from another manager:** choose **Import…** to see every profile Mortar found on this computer for the game, from r2modman, Gale, Vortex, Mod Organizer 2 and Stardrop, each with its manager and mod count. Pick one and Mortar previews it before anything downloads, then makes a new profile. For a Thunderstore game, **Use a code or file…** reads an r2modman or Gale profile code or `.r2z` file instead.
- **Export as a Thunderstore modpack:** for a Thunderstore game, the profile menu's **Export as Thunderstore modpack…** saves a modpack zip of the profile's switched-on Thunderstore mods and its config folder. Mods from other sites, and switched-off ones, are left out and named in the zip's README.
- **Multiplayer mod list:** for Stardew Valley, the profile menu's **Multiplayer mod list…** copies the host's switched-on mods, with their versions and download sources, so friends can match them. A friend pastes the list under **The host's list** and chooses **Check**: mods they lack show **Install**, mods on another version show **Update** (the host's exact file), and **Fix all** queues every one. A mod with no download source, such as one added from a local archive, is named to install by hand. SMAPI does not require a match itself, but many mods refuse a player on another version, so match before joining. The same from a terminal: `mortar profile farm export|check|fix`.
- **Keep a profile's saves separate:** by default every profile plays on the game's one Saves folder. To give a profile its own, open the profile's menu, choose **Edit profile**, enable **Keep this profile's saves separate** and confirm **Enable**. Leave **Start with a copy of my current saves** ticked to start from your saves, or untick it to start empty. From then on, each time you press Play Mortar sets your shared Saves folder aside, shows the game the profile's own folder (`saves` inside the profile) and puts the shared folder back when the game closes. Nothing is deleted or moved for good, and if Mortar quits while the game runs, the next start puts the shared folder back. Switching the option off leaves the profile's folder where it is. Only games with a save folder have the switch.

## Play

Press **Play** in Mortar. It applies the profile's mod list, then starts the game through SMAPI.

When a run crashes, a card appears at the top of the profile. Mortar reads the loader's log (and, for a Lethal Company run, Unity's player log, where a crash prints `Crash!!!` or `Fatal error`) and names the likely cause:

- **Mortar thinks \<mod\> caused the crash** with the log's line. **Disable and retry** switches that mod off and starts the game again.
- **The last run crashed.** with the log's line, when the log names no mod you have.
- **Bisect from here** halves the mods and relaunches until one mod is left, and **Dismiss** hides the card for that run.

On Windows, Steam's own **Play** button is different: it starts SMAPI with the game's own `Mods` folder, not a profile. To play a profile, press Play in Mortar, or use a profile shortcut. A profile's menu has **Add a shortcut that plays this profile** (a Start Menu shortcut on Windows, a launcher entry on Linux) and **Add this profile to Steam**, which adds the profile to your Steam library as a non-Steam game. Close Steam before adding.

## Problems

The **Problems** tab checks the open profile for missing requirements, conflicts, broken or outdated mods, damaged files, duplicates and more. It warns and never blocks Play.

- A strip across the top of the tab has one segment for each kind of problem the profile has, each with its count. Choose a segment to see its rows; the others are hidden, so only one list is on screen at a time. Left and Right arrow keys move between segments.
- Mortar opens on the first segment that holds something broken, and remembers your choice for each profile while Mortar runs. A kind with nothing in it has no segment.
- The chosen segment's own action sits at the right end of the strip: **Add all** installs every missing requirement Mortar can find, and **Dismiss all** hides harmless overlaps.
- Segments include **Failed to load** (a plugin the loader's log shows failed), **Plugins shipped twice** (two enabled packages carrying the same plugin, with **Keep newer**) and **Deprecated packages** (Thunderstore marks them, with **Replace with** where it names a replacement).
- A mod's name in a row filters the **Mods** tab to that mod.

### What each problem means

Each row has a **Why?** that explains it, with a link to its section here.

#### Missing requirement

A mod you enabled needs another mod that is not in the profile, or is disabled. The first mod may fail to load or leave out features. **Add** installs the one it needs when Mortar can find it.

#### Broken mod

SMAPI's list marks the mod as broken, obsolete or abandoned for your game version. It may crash, do nothing, or stop working after the next game update. Update it, or replace it with the mod the row names.

#### Duplicate mod

Two enabled copies of the same mod are in the profile. The game loads only one, and which one is not something you control. Remove the copy you do not want.

#### Conflicting files

Two mods change the same file or asset. Only one change wins, so the other mod may look or behave wrongly. The row says which mod wins; disable the one you prefer to lose, or dismiss the row when the overlap is harmless. Open **Why?** and choose **Show conflicts** to see every mod that changes the asset, in load order, with the kind of change each makes and which one wins.

#### Errors in the last run

A mod wrote errors to the game's log the last time you played. It may be broken, out of date, or missing something it needs. Open the log to read what it said.

The Console's **Problems in this run** strip reads SMAPI's log for the cause and offers one fix per mod:

- **Install** a requirement that is not installed.
- **Update** a mod SMAPI calls no longer compatible, one whose requirement is too old or was skipped itself, or one whose Harmony patches failed.
- **Update SMAPI** for a mod that needs a newer SMAPI.
- **Disable** a mod that crashed on entry, has a broken manifest or DLL, is obsolete, needs a newer game, or a content pack whose patches failed.
- **Remove duplicate** for a mod installed twice, and **Remove** for a mod SMAPI flags as malicious.

Each recorded run in the Console's run list also says how many of its log's warnings and errors Mortar could not classify.

#### Failed to load

The loader's log shows that a plugin did not start. Its features will be missing in the game. The row quotes the log line; the usual causes are a wrong game version or a missing requirement. The button follows the cause:

- **Find** searches for a plugin it needs that is not installed.
- **Update** opens mod updates for a plugin built for other versions, one that failed to load, or one whose Harmony patches failed.
- **Reinstall loader** for a plugin that needs a newer BepInEx, or when the loader itself failed to start.
- **Disable** for an older copy skipped in favour of a newer one, a plugin incompatible with another installed one, one whose requirement failed, or one that keeps reporting errors.

#### Setting suggestion

A mod works better, or only works, with a setting changed. Nothing is wrong yet. The row says which setting, and the button changes it.

#### Damaged files

Files of a stored mod went missing, changed or appeared since Mortar stored it, often from an antivirus, a disk fault or a manual edit. The mod may misbehave. Reinstall it to restore the stored files.

#### Plugins shipped twice

Two enabled packages carry the same plugin. The loader starts only one of them, so you may be running the older one. **Keep newer** leaves the newest enabled.

#### Deprecated packages

The author marked the package as deprecated on Thunderstore. It will not get fixes and may break with the next game update. Replace it with the package the row names, when it names one.

#### Changed outside Mortar

Something changed a profile's mods folder without Mortar: a file added, removed or edited by hand or by another tool. Mortar flags it so the profile matches what you expect. Keep the change or put the file back.

## Notifications and change history

The bell in the title bar opens **Notifications** (Ctrl+Shift+N does the same). It holds two lists:

- **New** has this session's toasts you have not read, with their buttons (such as **Undo**) while they still apply.
- **Earlier** has the open profile's own change history, newest first: mods added, removed, updated, rolled back and switched, each with when it happened. **Show diff** lists what changed and **Undo** puts the profile back to how it was before that change.

**All changes…** opens the full history, and **Clear** empties the toasts. History lives with the profile, so it is still there after a restart.

## Sync your profiles between computers

A sync folder shares each profile's mod list, sources, settings and mod config files between your computers through a folder that something else carries, such as Syncthing, Dropbox or a NAS. Mod files are never copied; each computer downloads them from their sources. Sync is off until you choose a folder.

1. On each computer open **Settings › Mods and profiles**, find **Sync folder** and choose **Choose…**. Pick the folder your sync tool keeps in step.
2. Work as usual. A few seconds after a profile stops changing, Mortar writes it into the folder.
3. On another computer, a toast says **\<profile\> was changed on \<computer\>**. Choose **Review** to open **Changes from your other machines**. **Show diff** lists the mods the change adds and removes, **Apply** makes your profile match, **Keep mine** keeps yours and writes it over the other revision, and **Later** closes the list.
4. A profile that exists only on the other computer arrives as a new profile when you choose **Apply**.
5. When both computers changed the same profile, the toast and the list say so and Mortar never merges them: pick **Use theirs** or **Keep mine**.

**Disable** beside the folder stops syncing and leaves the folder's files alone.

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
