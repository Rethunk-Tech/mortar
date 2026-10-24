# Mortar user guide

Mortar manages mods for the PC games its catalog enables. It finds the game, installs its mod loader (SMAPI for Stardew Valley), keeps each set of mods in its own profile and starts the game with the profile you pick. Internals: [architecture.md](architecture.md).

## Install

Download from [mortar.rethunk.tech/download](https://mortar.rethunk.tech/download/), or every file from [Releases](https://github.com/Rethunk-Tech/mortar/releases/latest).

### Windows

- **Installer:** `mortar-amd64-installer.exe` (`mortar-arm64-installer.exe` on an ARM PC). It installs for your account only, into `%LOCALAPPDATA%\Programs\Mortar`, with Start Menu and Desktop shortcuts and no administrator prompt.
- **Bare program:** `mortar-windows-amd64.exe` or `mortar-windows-arm64.exe`, run from any folder.
- **Scoop:** `scoop bucket add mortar https://github.com/Rethunk-Tech/scoop-bucket`, then `scoop install mortar/mortar`. Scoop handles updates; Mortar's own updater stays off.

The window needs Microsoft's WebView2 Runtime; the installer adds it when missing and the bare program asks first. The builds are not code-signed, so SmartScreen warns the first time: choose **More info**, then **Run anyway**.

### Linux

- **AppImage** (`mortar-linux-x86_64.AppImage`, `mortar-linux-aarch64.AppImage`): `chmod +x` and run it. It needs glibc 2.35 or newer (Ubuntu 22.04, Linux Mint 21, Debian 12, Fedora 36); on an older system use the Flatpak.
- **Portable program** (`mortar-linux-amd64`, `mortar-linux-arm64`): the same requirements plus GTK 3 and WebKitGTK 4.1 (`libwebkit2gtk-4.1-0` on Debian and Ubuntu, `webkit2gtk4.1` on Fedora, `webkit2gtk-4.1` on Arch). It updates itself and on first run adds a menu entry and takes nxm:// links.
- **Flatpak:** `flatpak install --user https://mortar.rethunk.tech/packages/flatpak/tech.rethunk.Mortar.flatpakref` adds Mortar's signed repository, so `flatpak update` keeps it current. Each release also has `mortar-linux-x86_64.flatpak` and `mortar-linux-aarch64.flatpak` for a one-time install without updates. Sandbox limits:
  - Folders outside the Flatpak's allowed set (Steam, Heroic, Lutris and the usual game folders) are invisible. Grant one with `flatpak override --user --filesystem=<folder> tech.rethunk.Mortar`.
  - **Add this profile to Steam** is refused; the error shows the `flatpak run tech.rethunk.Mortar --play=<game>/<profile> --steam-session` command to use instead.
  - Flatpak Steam cannot read profiles until you choose **Grant access** in the game's settings.
- **Homebrew:** `brew install rethunk-tech/tap/mortar` installs the AppImage; `brew upgrade` updates it.
- **`.deb`, `.rpm`, Arch package:** `sudo apt install ./mortar_*_amd64.deb`, `sudo dnf install ./mortar-*.rpm` or `sudo pacman -U mortar-*.pkg.tar.zst`. These do not update themselves; add the repository below.

### Linux package repository

Installed from the repository, Mortar updates with the rest of your system (Settings › Updates says "Updated by your package manager"). The signing key is "Mortar packages <opensource@rethunk.tech>", fingerprint `D8B1 C4C2 05C5 FB36 33CB 8B99 00B7 5959 B637 DCB5`.

Debian and Ubuntu (amd64, arm64):

```sh
sudo apt update && sudo apt install curl
sudo curl -fsSLo /usr/share/keyrings/mortar-archive-keyring.gpg https://mortar.rethunk.tech/packages/mortar-archive-keyring.gpg
sudo curl -fsSLo /etc/apt/sources.list.d/mortar.list https://mortar.rethunk.tech/packages/mortar.list
sudo apt update && sudo apt install mortar
```

Fedora (x86_64, aarch64); dnf asks once to import the key, so compare its fingerprint with the one above:

```sh
sudo curl -fsSLo /etc/yum.repos.d/mortar.repo https://mortar.rethunk.tech/packages/mortar.repo
sudo dnf install mortar
```

Arch (x86_64, aarch64):

```sh
curl -fsSL https://mortar.rethunk.tech/packages/mortar-archive-keyring.asc | sudo pacman-key --add -
sudo pacman-key --lsign-key D8B1C4C205C5FB3633CB8B9900B75959B637DCB5
curl -fsSL https://mortar.rethunk.tech/packages/pacman.conf | sudo tee -a /etc/pacman.conf
sudo pacman -Syu mortar
```

## First run

Continue past the welcome screen (it lists the launchers found: Steam, GOG, Heroic, Lutris, Bottles) and open Stardew Valley. Setup has three steps.
1. **Game folder.** Mortar shows where it found the game. A copy in Desktop, Downloads or Games (on Windows also a drive's root) is offered under **Found a copy outside your launchers**; otherwise use **Choose folder…**. **Change folder…** and **Rescan** fix a wrong pick. On Linux, a Windows build outside Steam runs through a Proton version your Steam already has.
2. **Install SMAPI.** Mortar installs it by itself; if that fails, choose **Retry** or **Skip for now**. On Windows with Steam, the game starts without SMAPI unless SMAPI is in its Launch Options (right-click the game in Steam, **Properties**). Mortar shows the line to paste, or choose **Set it in Steam for me** with Steam closed.
3. **First profile.** **Start empty** (only SMAPI), **Import from the game's Mods folder** (copies your mods, changes nothing in the game folder), **From another mod manager** or **From a shared link**.

## Sign in to Nexus Mods

In **Settings › Accounts**, choose **Sign in to Nexus Mods** and approve Mortar in your browser. If that section shows **Personal API key** instead, copy your key from nexusmods.com (**Settings**, then **API Keys**), paste it in and choose **Sign in**. The key is kept in your system keyring.

Premium accounts download inside Mortar. Free accounts click **Mod Manager Download** on the Nexus page once per file (Mortar opens each page in turn); enable **Handle "Mod Manager Download" links** in **Settings › Downloads** so the button reaches Mortar.

### Downloads

The downloads panel opens with Ctrl+J.

- A download over 2 GB waits for **Continue anyway**. Before any download starts, Mortar checks that the file, its unpacked copy and a little spare room fit on the drive; if not, it fails at once and says how much is needed and free. Free some space, then **Retry**.
- **Pause** keeps what was fetched; **Resume** continues from there, also after a restart, and replaces Retry after a network error unless the file changed on the server.
- **Download now** moves a download to the front; if every slot is busy, the last running one steps aside and restarts when a slot frees. The arrows reorder waiting downloads.
- **Settings › Downloads › Download speed limit** caps all downloads together in MB/s (0 is no limit).
- **Pause downloads while the game runs** (a game's **Play** settings, also a profile override) holds downloads while a game you launched from Mortar is open. Downloads you paused yourself stay paused.

## Browser extension

The extension is optional. It marks Nexus pages with what your profile already has and sends Mod Manager Download clicks to Mortar. **Settings › Downloads** has a **Download the extension** button for `mortar-browser-extension.zip` from its [releases](https://github.com/Rethunk-Tech/mortar-browser-extension/releases/latest), and says **The browser extension is too old for this Mortar** (or too new) when versions drift.

- **Chrome, Edge, other Chromium:** install it from the [Chrome Web Store](https://chromewebstore.google.com/detail/hboeppdoecbglcmfojappgehkbecdfbi), which keeps it updated. To try the zip instead: unzip, open `chrome://extensions`, turn on Developer mode, choose **Load unpacked** and pick the folder.
- **Firefox:** open `mortar-browser-extension.xpi` from the latest release in Firefox, or drop it on `about:addons`; Mozilla signs it, so it stays. Without an `.xpi`, use the zip: `about:debugging`, **This Firefox**, **Load Temporary Add-on**, pick `manifest.json` (removed when Firefox restarts).

## Find and add mods

**Browse** searches every site the game's mods come from at once (for Stardew Valley, Nexus Mods, CurseForge and GitHub). A mod several sites list is one card; its badge picks the site to install from, and **Preferred source** (the game's settings, **Mods** page) sets the default. The chips beside **All sources** narrow to one site.

1. Open **Browse**. With nothing typed, each site lists its top mods.
2. Type a name; results update as you type.
3. Choose **Add** (**Download** on Nexus with Premium) on a card. It shows progress, then **Added**. On Nexus with a free account the button is **Get**, which opens the mod's files page.

- **GitHub:** **Add** installs the latest release; `gh auth login`, when present, raises GitHub's rate limits. Nobody reviews or scans what is published on GitHub or itch.io, and the first download from either says so once.
- **Card details:** **Show › Card details** picks what cards list; the Mods tab uses the same choice.
- **Downloads folder:** an archive saved to Downloads, Mortar's download folder or a folder under **Settings › Downloads › Also watch these folders** is offered for install.
- **Extra mods folders** (the game's settings): unpacked mods, or the output folder of a mod you are building. Each folder names the profile its mods install into (or the open one); they install when you add the folder and again whenever its files change, so the game loads your latest build. A change while the game runs waits for it to stop.
- **Hide from Browse:** a card's menu has **Hide from Browse** and **Hide everything by \<author\>**; the game's settings list **Hidden mods** with **Unhide**. Links and imports still install a hidden mod.
- **Modpacks, translations and cheats:** the **Exclude categories** picker starts with them; remove a chip to see them.
- **Open a mod's page:** Ctrl+click a card in Browse, or middle-click a mod in the Mods tab.
- **Versions:** **Choose a version…** on a card's details page installs a version other than the latest; an installed mod offers **Change version…**. A **Support the author** link shows when the mod's page gives a donation address.

### Narrow and sort the results

- **Include categories** and **Exclude categories** filter by category; sites without categories (GitHub, or Nexus while signed out) turn them off.
- **Sort** orders the results. Under **All sources**, most downloaded and recently updated rank every site as one list, leaving out (and naming) a site that cannot sort that way, such as GitHub for downloads. Best match takes turns between sites.
- **Show** has rows for **Mods in this profile**, **Obsolete** and **Broken**, each **Off**, **Gray out** or **Hide** (counted as "N hidden"); **Obsolete** and **Broken** can also **Flag** a mod with a warning mark. **Broken** needs a compatibility list, which Stardew Valley has.

Mods already in the profile show **Added**, whichever site they came from. A helper Mortar installs for a loader shows **Built in**.

## Modrinth and itch.io

A game whose catalog entry lists Modrinth or itch.io gets a chip for it beside **All sources**, and its results join the merged search. Stardew Valley lists neither.

- **Modrinth** needs no account. **Add** installs the newest version with its required projects.
- **itch.io** needs your own API key: create one in your itch.io account (**Settings › API keys**), paste it into **API key** in **Settings › Accounts** and choose **Test**. The key is kept in your system keyring; **Remove key** forgets it. Until then the chip is greyed out. itch.io has no sort order or categories, and **Add** takes the game's first upload that is not a demo or soundtrack.

## Edit a mod's settings

Mortar edits a mod's config as a form, and other files a BepInEx mod keeps (YAML, TOML, XML, INI, JSON, text) as text. A mod that has not run yet has no config.

1. In the **Mods** tab, select a mod, then choose **Edit config** under **Config** (or right-click it, or use its **⋯** menu).
2. Pick one of its config files (`config.json` for a SMAPI mod, each BepInEx `.cfg` for a Lethal Company plugin).
3. Change a value. Each change saves as you make it, but not while the game runs the profile. A setting that differs from the mod's default shows **Reset to default**; **Reset all** restores the file. An info button shows the mod's own description, and **Search entries** filters long files.
4. **Presets** saves the file's values under a name and applies them to any profile's copy.

The **Config** tab (Ctrl+6) lists every mod with settings and searches every config file in the profile. A change under **In-game menu** (Generic Mod Config Menu) says **Applied when the game next starts**. **Delete file** removes a config file no mod owns. **Centre config controls** (**Settings › Appearance**) sets each control nearer its name on a wide window.

## Your mod list

- **Favourites:** the star on a mod marks it; the profile menu's **Add to favourites** does the same for a profile.
- **Edit name, author and version…** (the mod's menu) changes how one profile shows a mod; **Use the mod's own** goes back.
- **Update channel:** a mod from GitHub, CurseForge or Modrinth can follow a different release channel; **Pin this version** stops its updates.
- **Disabling a mod:** when other enabled mods need it, Mortar offers to disable those too.
- **Report to author…** (the mod's menu) opens a prefilled report for a Thunderstore, CurseForge or Vintage Story mod.

## Lethal Company and Thunderstore

Lethal Company's mods are Thunderstore packages on BepInEx. Mortar installs BepInEx, searches Thunderstore alongside Nexus Mods, CurseForge and GitHub, and **Add** installs a package with its dependencies. A dependency still missing is flagged on the Load order tab and in **Before you play**.

- **Thunderstore links:** `ror2mm://` links (**Install with Mod Manager**) stay with your other manager until you turn on **Handle Thunderstore links** in Settings › Downloads (turn it off to hand them back). A link names no game, so it installs into the open game when that game has a Thunderstore community.
- **CurseForge links:** turn on **Handle CurseForge links** in Settings › Downloads, or tick **Open CurseForge Install links with Mortar** on the Windows installer's last page. A mod for a managed game asks which profile it goes into; other games' links go back to the CurseForge app.
- **Vintage Story Mod DB links:** turn on **Handle Vintage Story Mod DB links** in Settings › Downloads.
- **Play:** BepInEx reads only the game's folder, so on each launch Mortar places the profile's BepInEx files beside the game (`winhttp.dll`, `doorstop_config.ini`) and removes them when the game exits (a crash is cleaned up at the next start). Plugins and settings stay in the profile; the log is `BepInEx/LogOutput.log`, also shown in the **Console** tab.
- **Launch options:** profile menu › **Edit profile**, **BepInEx** group: **Show the console window** and **Log level**, applied at the next start.

## Pair your computers

Pairing lets a profile sent between two of your computers carry its mod files. An unpaired computer still receives the profile and downloads each mod from its source.

1. On both, enable **Share profiles on the local network** in **Settings › General › Sharing**.
2. On the first choose **Pair a computer**; it shows a code like `ABCD-EFGH`, valid once for five minutes.
3. On the second choose **Enter code**, pick the first from the nearby list (or type its `host:port`) and enter the code. Five wrong codes lock entry from that computer for ten minutes.
4. Both list each other under **Paired computers**; **Unpair** forgets one.

To send, use **Share…**, **Nearby**, and pick the computer. The receiver is asked to accept unless **Auto-accept from paired computers** is on.

On Windows, sharing is off while every connected network is **Public**; set it to **Private**, and use **Fix** in Settings › General if the firewall still blocks Mortar.

## Switch games

Click the game's name in the title bar; it opens on the profile you used last. **All games…** opens Game select, where right-clicking a game offers **Hide this game** (and **Unhide**) and clicking a section's heading folds it away. The mouse's back and forward buttons, and Alt with the left and right arrows, move through the pages you opened.

## Game installs

A game installed more than once (Steam and GOG, a beta branch, another folder) has a separate install for each. A profile uses the selected install unless you pin it in the game's settings; profiles pinned to different installs can run at once.

### Bottles

On Linux, Mortar finds a Windows build inside a [Bottles](https://usebottles.com) bottle (native or Flatpak Bottles) when the game sits in the bottle's Steam or GOG folders; the store reads **Bottles**. Open the game from **Game select** (use **Choose folder…** if it sits somewhere unusual in the bottle). Mortar installs the Windows SMAPI inside the bottle and **Play** starts the game through it.

## Profiles

A profile is one set of mods. Switching profiles never touches the game folder's own `Mods` folder.

- **Create:** **New profile…** (Profiles page or title-bar switcher), then **Start from**: **Empty profile**, **Copy of \<the open profile\>** (mods, their settings and what they wrote) or a template from **Save as template…** (**Manage templates…** renames or deletes them).
- **Nested groups:** name a group with a slash, such as **Looks/Hats**, to put it inside **Looks**. With the list grouped by **Group**, each level has a header that collapses its contents, and the switch on a header enables or disables everything inside. Renaming or deleting a group does the same to the groups inside it.
- **Why each mod is there:** a mod you add is **Explicit**; one added because another needs it is **Implicit**. Group the Mods list by **Install reason**, and right-click a mod to mark it either way.
- **Delete:** a deleted profile waits in **Recently deleted** (**Restore**, **Delete permanently**). Play always starts the open profile.
- **Freeze:** **Freeze this profile** holds its mods on their versions and offers no updates until **Unfreeze**.
- **A game that is not listed:** the game list's **Add a game…** takes a name, the game's program and how it is modded (BepInEx 5, BepInEx 6, MelonLoader, Shimloader or a Mods folder). It stays on this computer and has no mod site, so add its mods from archives. A Shimloader game (an Unreal Engine game such as Voices of the Void, whose program is the one in `Binaries/Win64`) also names its Thunderstore community, where its mods come from. **Remove** in the same dialog forgets the game and leaves its folder and profiles on disk.
- **Profile folder:** **Open profile folder** opens it in your file manager; **Move profile folder…** moves a profile to a folder of your choice, such as one on another drive, and back. On another drive its mods are copies instead of links into the shared store.
- **Share:** **Share profile**, then **Copy link** (names the mods and their Nexus or GitHub files, with pins, groups and install reasons) or save a `.mortar` file (also carries settings and notes; suits large profiles). **Share › Mod list** can also write a Thunderstore dependency string list. For a Thunderstore game, **Save .r2z…** writes an r2modman or Gale profile file without posting a public code, and **Export as Thunderstore modpack…** opens a form for the pack's details; the next export suggests the next patch version and writes a changelog of what changed since the last. Mods from other sources are named in the README.
- **Separate saves:** **Edit profile**, enable **Keep this profile's saves separate** and confirm. Leave **Start with a copy of my current saves** ticked to start from your saves, or untick it to start empty. On each Play Mortar swaps in the profile's own `saves` folder and restores the shared one when the game closes (or at the next start). Nothing is deleted. Only games with a save folder have it. In Cities: Skylines II only cities saved to **Local** are kept per profile; Steam Cloud and Xbox cloud cities are shared by every profile.

### Import

**Import…** (Profiles page, a profile's menu and the first-run steps) opens one dialog with four sources. Each ends in a preview of what the profile holds; Mortar then installs or queues what is missing. The preview offers a new profile, or adds to or replaces the open one. **Restore from a file…** reads a **Back up to a file…** backup.

- **A link or code:** a Mortar link, a Nexus collection, an r2modman code, a Gale sync code (the short form such as `E5RG8F` and the older 22-character form), or a Patreon or itch.io post (Mortar opens the page and offers the file you save from it).
- **A file:** a `.mortar` file, an r2modman or Gale `.r2z`, a Thunderstore modpack zip or a local profile folder.
- **Local mods:** mods the other manager added from disk (from a Gale profile on this computer, or a code or `.r2z` that carries their files) come along as Local mods, disabled if they were. The preview marks them before anything installs, because a code's files come from whoever made it.
- **The game's Mods folder:** copies the mods the game reads into a new profile and leaves the folder as is. Tick **Move them out of the game's Mods folder** to take the originals out; left there, the game keeps loading them even when a profile disables them, and Mortar refills that folder on each Play. For The Sims 4 it is `Mods` in the Documents folder.
- **Another mod manager:** lists profiles found for the game from r2modman, Gale, Vortex, Mod Organizer 2 and Stardrop, and can read every r2modman or Gale profile in a data folder you choose. Copied mods may have no page to update from: group the Mods list by **Source** and choose **Find sources**, or right-click a mod › **Set Nexus Mods page…**. Mods with an unknown file still update, but a share leaves them out.

### Multiplayer

- **Multiplayer mod list (Stardew Valley, Thunderstore games such as Lethal Company, REPO, PEAK, Content Warning and Valheim, and Vintage Story):** profile menu › **Multiplayer mod list…** copies the host's enabled mods with versions and sources. A friend pastes it under **The host's list** and chooses **Check**: missing mods show **Install**, other versions **Update** (to the host's version, even when yours is newer), and **Fix all** queues every one; a mod with no download source is named to install by hand. On Stardew, **Check** also reads **Share › Mod list** text and **Load the host's list from a file…**; those carry no ids, so mods match by name and may come out newer than the host's. Many mods refuse a player on another version, so match before joining.
- **Dedicated servers (Valheim and Risk of Rain 2):** each server is its own game in Game select, with profiles of its own. **Edit profile › Server** sets its name, port and password, and **Use mods from…** queues a client profile's mods for install, so the server and the clients share one list. Valheim refuses to start without a password of five or more characters.

## Play

**Play** applies the profile's mod list and starts the game through its loader. Direct Play (not a Steam or store URL launch) can start more than one instance, spaced by a delay, for local multiplayer: set both in the menu beside **Play**. A Heroic or Lutris copy starts through its own launcher, so the game runs in the prefix and runner the launcher keeps for it. In a BepInEx game whose bridge has a console, the **Console** tab takes commands. **Skip the intro** in Stardew Valley's settings skips the title intro. **Launch options…** in the menu beside **Play** opens the profile's own launch options: extra arguments for the game, a command prefix (for example `gamemoderun mangohud`) and environment variables; the prefix and the environment apply to direct launches only. Launch presets keep several sets, each a **Play with** choice in that menu.

On Windows, Steam's own **Play** starts SMAPI with the game's own `Mods` folder, not a profile. Press Play in Mortar, or use the profile menu's **Add a desktop shortcut** (Start Menu on Windows, launcher entry on Linux) or **Add this profile to Steam** (a non-Steam game; close Steam first).

### After a crash

A card on the profile's Home names the likely cause, read from the loader's log (for Lethal Company also Unity's player log) and, when those name no mod, the game's crash dump. When the game died in native code the card names the module and offset; for Unity's player and the Windows system libraries Mortar then downloads that module's public symbols from Unity's or Microsoft's symbol server (only then, and only when online) and names the function instead.

- **Most likely cause: \<mod\>** shows the log line and the **Stack frame** or dump line. **Disable it and play** disables that mod and relaunches.
- **Find the cause…** first checks that the game crashes with every mod on, then disables mods in halves and relaunches until one is left. A library stays on while a mod that needs it is on. **Dismiss** hides the card for that run.

## Problems

The **Problems** tab checks the open profile for missing requirements, conflicts, broken or outdated mods, damaged files, duplicates and more. It warns and never blocks Play.

- Each kind of problem is a section with a count (red when it holds something broken); an empty section is not shown. **Add all** in the missing-requirements section installs every requirement Mortar can find.
- Advice and history (Slow startup, Redundant, Cleanup, Replaced, Dismissed) are links under the list; choose one to open it below, and again to close it.
- A mod's name in a row filters the **Mods** tab to it. **Why?** on a row explains it and links to its section below. A row's fix is a button; its arrow opens the alternatives and **Dismiss**.

### What each problem means

#### Missing requirement

A mod you enabled needs another mod that is not in the profile or is disabled, so it may fail to load or lose features. **Add** installs the missing one when Mortar can find it.

#### Broken mod

SMAPI's list for your game version, Mortar's list of known-broken mods or the mod's own site marks it broken, obsolete, abandoned or gone (a hidden, removed or moderated Nexus page; an archived, inactive, abandoned or deleted Modrinth, CurseForge or GitHub project; a Thunderstore package taken off the site). The row names who said so. The mod may crash, do nothing, or break at the next game update. Update it, or replace it with the mod the row names.

#### Duplicate mod

Two enabled copies of one mod are in the profile. The game loads only one, and you do not control which. Remove the one you do not want.

#### Conflicting files

Two mods change the same file or asset and only one change wins, so the other may look or behave wrongly. The row says which wins; disable the one you prefer to lose, or dismiss the row if the overlap is harmless. **Why?** then **Show conflicts** lists every mod touching the asset in load order, with each change's kind and the winner.

#### Errors in the last run

A mod wrote errors to the game's log last time you played; it may be broken, out of date or missing something. Open the log to read them.

Right-click a Console line to **Copy this message**, or to **Hide this message**: every line from the same place in the game's code (or, for an engine warning, the same warning about the same object) leaves the Console, and in a BepInEx game the bridge stops the game writing it. The game's settings › **Console** › **Hidden messages** shows them again.

The Console's **Problems in this run** strip reads SMAPI's log and offers one fix per mod: **Add** a missing requirement, **Update** the mod, **Update SMAPI**, **Disable** a mod that crashed or is obsolete, and **Remove duplicate** or **Remove** (a mod installed twice, or one SMAPI flags as malicious).

#### Errors while running

A plugin that loaded and then logged errors or threw an exception is listed once, with its error count. The row names the method that threw and the other enabled mods that patch code on that path, since one of them may be the cause. The plugin ran, so these rows do not count as problems. **Disable** or **Update** it if the game misbehaves, or **Dismiss** the row for that run.

#### Failed to load

The loader's log shows a plugin did not start, so its features are missing. The row quotes the log line; usual causes are a wrong game version or a missing requirement. The button follows the cause: **Find** a plugin that is not installed, **Update** one built for other versions, **Reinstall loader** when a plugin needs a newer BepInEx or the loader failed to start, or **Disable** a plugin that is incompatible, skipped for a newer copy, or keeps reporting errors.

#### Setting suggestion

A mod works better, or only works, with a setting changed. Nothing is wrong yet. The row names the setting and the button changes it.

#### Damaged files

Files of a stored mod went missing, changed or appeared since Mortar stored it (an antivirus, a disk fault, a manual edit). **Repair** downloads the mod again from its source; a local mod is re-extracted from its archive if that is still in Downloads.

#### Paths too long

On Windows, a mod file's full path, where the game reads it, is 260 characters or more. Most games cannot open such a file, and turning on long paths in Windows does not change that. The row names the longest path. For a file the game reads from the profile, move Mortar's data folder somewhere shorter (**Open storage settings**, then **Move…**); a file placed in the game's own mods folder needs the game in a shorter folder. Linux, including games under Proton, is not affected.

#### Files two packages ship

Two enabled packages ship a file at the same path, and the one later in the package order replaces the other's copy. The row names the winner, the packages it beats and the files. **Make … win** moves a beaten package to just after the winner. These rows are information and do not count as problems.

#### Plugins shipped twice

Two enabled packages carry the same plugin. The loader starts only one, so you may be running the older. **Keep newer** leaves the newest enabled.

#### Deprecated packages

The author marked the package deprecated on Thunderstore. It gets no fixes and may break at the next game update. **Replace with** swaps in the package the row names, when it names one.

#### Changed outside Mortar

A profile's mods folder changed without Mortar (a file added, removed or edited by hand or another tool). Keep the change or put the file back.

## Notifications and change history

The bell in the title bar (Ctrl+Shift+N) opens **Notifications**. **New** holds this session's unread toasts. **Earlier** is the open profile's change history, newest first: **Show diff** lists what changed and **Undo** restores the profile to before that change. **All changes…** opens the full history, which lives with the profile and survives restarts.

## Sync your profiles between computers

A sync folder shares each profile's mod list, sources, settings and mod config files between your computers through a folder another tool carries (Syncthing, Dropbox, a NAS). Mod files are never copied; each computer downloads them from their sources.

1. On each computer open **Settings › Mods and profiles**, find **Sync folder**, choose **Choose…** and pick the folder your sync tool keeps in step. Each game syncs into its own subfolder; a game's **Profile sync** setting can give it a folder of its own or turn sync off for it.
2. Work as usual; a few seconds after a profile stops changing, Mortar writes it into the folder.
3. On another computer a toast says **\<profile\> was changed on \<computer\>**. **Review** opens **Changes from your other machines**: **Show diff**, **Apply** (make your profile match) or **Keep mine** (overwrite the other revision).
4. If both computers changed the same profile, Mortar never merges: pick **Use theirs** or **Keep mine**.
5. To keep one mod enabled or disabled, or one config file, different on this computer, right-click the mod and choose **Enable or disable on this computer only**, or press **Only on this computer** in the mod's config editor. That choice is never written to the sync folder, and **Apply** and **Keep mine** leave it alone. Press the same item again to let it follow the sync. A profile must have synced once first.

**Disable** beside the folder stops syncing and leaves its files alone.

## Updates

- **Mortar:** the Mortar menu's **Check mods and Mortar for updates**, or **Settings › Updates** (which can include betas). An update applies when you close Mortar, or choose **Restart now**. The Mortar menu's **What is new in \<version\>** shows what changed.
- **Mods:** updates show on the mod list; **Settings › Updates** has **Review mod updates**.
- **Mod loaders and SMAPI:** Mortar tells you when a new version is out; each game's settings hold the loader version and update it.

## Appearance

**Settings › Appearance** sets the theme, accent colour (**System** follows your desktop's), font, **Size** and background (an image, the desktop wallpaper or a solid colour). Animated backgrounds hold still while a game runs or **Reduce motion** is on.

## Language

Mortar follows your system language once a translation of it is mostly done, and English otherwise. **Settings › General › Language** picks one yourself; a language still being translated shows how much is done, and the rest stays in English. Volunteers translate Mortar on [Hosted Weblate](https://hosted.weblate.org/projects/mortar/), and **Help translate Mortar** beside the picker opens it.

## Save backups and restore

Mortar zips your saves before it changes them: before a profile update, on every game start with **Back up saves before Play**, and on a timer with **Scheduled save backups** (both under the game's **Save backups** in Settings). A game whose catalog entry has no save folder shows no Saves tab, no Saves card on Home and no **Save backups** page.

To restore, stop the game, open the **Saves** tab, choose **Save backups…**, pick a backup and choose **Restore**. Mortar backs up the current Saves folder first. **Open backups folder** shows the zips; **Backup location** in the game's settings moves new ones. For Valheim, The Sims 4 and Vintage Story the list also shows the game's own copies, marked **Kept by the game**.

## Screenshots

The **Screenshots** tab shows the game's own screenshots and, for a Steam game, Steam's. **Use as profile cover** sets the open profile's cover.

## Where your data lives

| | Windows | Linux |
| --- | --- | --- |
| Mortar's data (profiles, mods, downloads, backups, settings, logs) | `%LOCALAPPDATA%\Mortar` | `$XDG_DATA_HOME/mortar`, usually `~/.local/share/mortar` |
| Your saves | `%APPDATA%\StardewValley\Saves` | `~/.config/StardewValley/Saves` |

The Mortar menu's **Open data folder** opens the first row.

To move the data, stop the game, open **Settings › Storage** and choose **Move…**. Mortar copies everything to an empty folder you pick, checks it, removes the old one and restarts. A file named `portable` beside the program keeps all data in a `data` folder next to it (not in a Flatpak or a read-only folder).

## Uninstall

Close Steam first so the Steam cleanup sticks: Steam rewrites its shortcut list on exit.

- **Windows:** Settings › Apps, or the uninstaller in the install folder. It removes the program, shortcuts, the start-at-login entry and the profiles you added to Steam, and leaves your data folder, SMAPI and the game folder; delete `%LOCALAPPDATA%\Mortar` to remove profiles, mods and settings.
- **Linux:** first run `mortar --release-links && mortar uninstall-cleanup` (AppImage: `./mortar-linux-x86_64.AppImage --release-links && ./mortar-linux-x86_64.AppImage uninstall-cleanup`, or the `aarch64` file; Flatpak: `flatpak run tech.rethunk.Mortar --release-links && flatpak run tech.rethunk.Mortar uninstall-cleanup`). It gives nxm:// links back and removes the extension's connection, the start-at-login entry, profile shortcuts, Steam profiles, and what an AppImage or portable binary added to the menu. Then remove the package, AppImage or Flatpak. Removing a `.deb`, `.rpm` or Arch package with `sudo` runs those steps for your own account only. Delete the data folder to remove profiles and mods.

## Stream overlay

For OBS, Mortar's in-game bridge serves live values (Stardew Valley, Lethal Company, Valheim) to a browser source. In the game's settings open **Stream overlay**, turn it on and press Play (it is read at game start). The page has previews, a copy button for each OBS URL, the OBS steps, the **Port** and the **Token**. It listens on this computer only.

## Antivirus scan

Before a mod enters Mortar's store, Mortar scans it with Windows Security or, on Linux, ClamAV's clamd when running. A flagged file refuses the install and deletes the extracted files. The error (**Flagged by \<scanner\>: \<name\> in \<file\>**) offers **Install anyway**, which asks to confirm, installs that mod once without a scan and records it in the profile's history. A scan that cannot finish installs the mod and warns you.

**Settings › General › Antivirus** picks **Automatic**, **clamd socket** (with its path), **Custom command** (a program given the folder as `{path}`: exit 0 is clean, any other code is a detection named by the first line it prints) or **Off**. **Scanner in use** names what Mortar found, or "No antivirus found".

## Troubleshooting

- **Mortar does not start:** it shows why, with a button that opens the folder holding `mortar.log`. Starting Mortar again while a copy is stuck offers to end it.
- **Run checks:** **Settings › About › Diagnostics › Run checks…**; **Repair** fixes what it can.
- **Logs:** `mortar.log` in the data folder, plus `crash.log` after a crash. Each profile's SMAPI log is on its log view; **Share log…** there uploads it to smapi.io/log and gives you a link.
- **Slow start:** `mortar.log` times each step of the start (lines beginning `startup:`). For more, the Mortar menu's **Restart and record startup profile** records Mortar's first 30 seconds into the data folder's `startup-profile` folder, and the next diagnostics zip includes it.
- **Slow scrolling or loud fans (Linux):** **Settings › Appearance › Display › Graphics** set to **Smoother scrolling** keeps the graphics card drawing the window, which some NVIDIA setups otherwise hand to the processor; **Compatible** is for a window that stays blank or white. Either applies after a restart, and `mortar.log` names the one in use (its `msg=graphics` line). A window too blank to reach Settings: start Mortar once with `WEBKIT_DISABLE_DMABUF_RENDERER=1` in its environment, which overrides the setting, and pick **Compatible**.
- **Report a bug:** the Mortar menu's **Report a bug** opens a prefilled GitHub issue. Leave **Attach diagnostics** ticked to save a zip (checks, logs, settings and recent runs, with secrets and your home folder removed) and drag it into the issue. **Save diagnostics…** (Mortar menu, Settings › About › Diagnostics) writes the same zip. **Report a bug** can also copy the report for a new post in the bug-reports forum on Mortar's Discord.
- **Ask for help or an idea:** the Mortar menu has **Ask on Discord** and **Suggest a feature…**.
