# GUI design

How Mortar's screens are laid out and styled. Concrete (`LethalModding/Concrete`, archived) is one reference for the look, not a template to copy.

## Window

- **Logo (NOMAD, 2026-09-29):** the running-bond mark, seven rounded bricks in three courses, filled in the primary colour; used in the title bar at 20px, as the app icon on a dark rounded square, and in notifications.

- Frameless, with a solid window (see [architecture.md](architecture.md#stack)): 1280×720 by default, 768×432 minimum, as Concrete.
- The app draws its own title bar, 36px, marked `--wails-draggable: drag`. Double-click maximises.
  - **Left:** the logo in a darker square and the app name, which open the app menu.
  - **Location tabs**, underlined in the primary colour: "Game Select", or the current game. First run shows a "Setup" tab. While Mortar's Settings page is open the game tab gives way to a "Settings" tab, and the game settings page keeps the game tab highlighted.
  - **Right:** minimise, maximise and close.
  - **App menu**, a side drawer listing Settings, About Mortar, Open data folder, Source code (the AGPL-3.0 source offer, opening `https://github.com/Rethunk-AI/mortar`) and, after a divider, Quit. Report a bug opens a prefilled GitHub issue; Check for updates opens Settings › Updates and checks at once.
  - **Drawer foot:** the Nexus account, as the account name with a Premium or Free badge and Sign out, or "Not signed in" with Sign in, which opens Settings › Nexus Mods.
  - **Frame:** a 1px light border and rounded corners, since GNOME draws no shadow for a frameless window.
- `user-select: none` everywhere except text fields and the console.
- Menus (the mod menu, the app's other menus) use a dark paper, `rgba(28,28,34,0.99)`, with a 1px light border and 8px corners (`MuiMenu` in `theme/theme.ts`); items with an obvious icon carry a Lucide one.
- Icons come from one set, Lucide (MIT), at 1.5-2px stroke; no hand-drawn or mixed icons.
- Every focusable control shows a visible focus ring when reached by keyboard: a 2px outline in the primary colour, 2px outside the control.
- Button and chip labels never wrap (`white-space: nowrap`); a long message gets its own full-width row, truncating with an ellipsis rather than squeezing the buttons beside it.

## Surfaces and colour

- No gradients anywhere, not as backgrounds, scrims over art, fallbacks or placeholder art (NOMAD, 2026-09-29): legibility over art comes from a solid layer with alpha, and a missing image becomes a solid tone. The one exception: an image's edge may fade to transparent via an alpha mask; no coloured gradients (NOMAD, 2026-09-30).
- The window is solid, an opaque `rgb(25,25,30)` base. What shows behind the surfaces is the wallpaper backdrop under a static `rgba(25,25,30,0.8)` tint (NOMAD, 2026-09-30). Surfaces keep their alpha and composite over it: background `rgba(25,25,30,0.80)`, paper `rgba(50,50,60,0.80)`, the app drawer `rgba(40,40,48,0.92)`; dialog paper and the download queue sheet are solid (`rgb(40,40,48)` and near it) so the text they cover never shows through (NOMAD, 2026-09-30); dialog and drawer backdrops a static `rgba(0,0,0,0.30)`; dialogs and their backdrops open and close without a transition, the launch overlay `rgba(0,0,0,0.80)`.
- **Background** (NOMAD, 2026-09-30), Settings › Appearance, a three-way toggle, all opaque and applied at once. Stored as `background` and `backgroundImage` in `settings.json`.
  - **Image** (the default): the chosen image, else the system Fedora wallpaper, else the bundled copy. Shows a preview with **Choose image…** (PNG, JPEG or WebP) and **Reset to default**.
  - **Desktop:** the user's own desktop wallpaper, read at runtime (GNOME `org.gnome.desktop.background` `picture-uri-dark` under the dark colour scheme, else `picture-uri`; Windows `SystemParametersInfo` `SPI_GETDESKWALLPAPER`, unmeasured until milestone 6), falling back to Image's choice when it cannot be read.
  - **Solid:** no image, the plain base.
  - Image and Desktop draw the wallpaper full-window behind everything, the title bar included, `object-fit: cover`, fully opaque under the 0.8 tint, and it never fades or filters.
  - **Default wallpaper:** Fedora 44's `f44-01-night` (Fedora Design Team, CC-BY-SA-4.0, credited in About). Mortar uses the system copy, `/usr/share/backgrounds/f44/default/f44-01-night.jxl`, when it exists on Linux (WebKitGTK 2.54 decodes JPEG XL, measured), else a bundled 2560px JPEG.
- Text: primary `rgba(255,255,255,0.90)`, secondary `rgba(200,200,200,0.90)`. Links are secondary text with a dotted underline.
- Status colours from Concrete: info `#2B8BDA`, success `#0CDF64`, warning `#F3B416`, error `#C70A0A`. The primary colour is the user's choice (NOMAD, 2026-09-29), in Settings › Appearance: Sand `#D6B17A` (the default), Moss `#93B86A`, Copper `#D98C5F` or Sky `#79AEDC`, applied at once everywhere the primary appears. It is the same for every game; game colour appears only in hero art.
- Scrollbars are themed: 0.5em, primary-dark track, primary thumb, primary-light on hover.

## Type

- Open Sans (300, 400, 500, 700) from `@fontsource/open-sans`, `htmlFontSize` 18 with MUI's responsive font sizes, as Concrete.
- Every string goes through Lingui.

## Game Select

Mortar opens on the last game used; Game Select shows at start only until a game has been opened, and otherwise from the title bar's Game Select tab.

Full-width banner rows stacked down the window, one per game, about 300px tall at 1080p; rows share the window's height equally. Choosing a row opens its main screen; the title bar's game tab returns here.

- **Art:** the game's key art edge to edge under a light solid dim layer, with plain text straight on it kept legible by a text shadow and no panel behind. Art is Steam's `library_hero.jpg` read at runtime, never bundled.
- **Left:** the game name with "<loader> | <store>" under it ("SMAPI 4.5.2 | Steam"), and the profile count.
- **Right:** square badges for its mod sources, each the source's logo above its name (Nexus's own mark; GitHub and Thunderstore from Simple Icons).
- **The row that opens on click** has a primary-colour left edge. In v1 Stardew Valley is the only live row; Lethal Company shows dimmed as coming later.

## Main screen

A grid of a resizable sidebar and a detail pane, as Concrete's dashboard:

- **Sidebar** (150-300px, resized from an 8px handle on its right edge, 2px primary right border): a "Profiles" header that opens profile management, then one row per profile in the user's order, the open one selected. Each row shows a problem or update badge when it has any.
- **Bottom block** pinned under the sidebar: a **New profile** row at the foot of the list, icon buttons for Support (a menu: Get help, which opens the Console tab's upload, and Report a Mortar bug) and a Downloads icon that opens the download queue (with a count while it has items), then a full-width **Play** button in the primary colour. There is no Settings icon here (NOMAD, 2026-09-30): it displaced Play, and Settings stays in the app menu and on Ctrl+,.
- **Detail pane** for the open profile:
  - A hero, 190px tall in every view and tab (NOMAD, 2026-09-30; only the minimum-size layout folds it to one line), bleeding to the pane's edges.
    - **Cover:** the profile's cover image at full opacity under a light solid dim layer (`rgba(20,20,24,0.18)`), its alpha fading to transparent over the bottom 40% by an alpha mask so the backdrop shows through into the tab row (the hero has no background of its own).
    - **Cover source, in order:** an image the user picked, else the Nexus picture of the profile's most-endorsed mod (`endorsement_count` from the mod endpoint), else Steam's own hero art for the game, else a solid tone; never a random image (NOMAD, 2026-09-29). Steam's art is read at runtime from `<Steam>/appcache/librarycache/413150/library_hero.jpg` (Steam also keeps a `library_hero_blur.jpg`) and never bundled with Mortar.
    - **Text:** the profile name large and bold with a soft white glow, and always the same cards (Mods, Updated, Created) so the name sits at the same place for every profile, with no subtitle line.
  - A **Saves** card in the hero (NOMAD, 2026-09-29) with how many saves fit this profile ("2 of 4"), opening the **Saves** tab: each save in the Saves folder with its fit, "all mods present" or "has used N mods this profile lacks" with their names, from the save scan ([architecture.md](architecture.md#saves)); a mod can be dismissed for a save. Players pick their save inside the game, so this is where the check shows, before Play.
  - Tabs: **Mods** (default), **Saves**, **Notes**, **Console**, and a **Share** button at the end of the tab row.
- **Launching** covers the window with the launch overlay: a spinner, "Launching Stardew Valley", and SMAPI's first log lines as they arrive (started, mods loading, mods it will skip), with Open console and Hide, until the game is up or the launch fails. While the game runs, the Play button becomes a Running status with the elapsed time and a **Stop game** button, and the open profile's mod controls are locked ("Stop the game to change mods."): switching, removing, installing and dropping archives. While Mortar installs SMAPI by itself, Play reads "Installing SMAPI…".
- **Minimum size (768×432):** the sidebar collapses to a rail of profile initials with Play at its foot, the hero gives way to a one-line header (name and counts), search and actions fold into icon buttons, and the mod grid drops to two columns.

## Mods tab

- Two views, switched from the toolbar and remembered per user (NOMAD, 2026-09-29): **Grid** (default), and **List**, a table (switch, picture, name, author, source, category, status; author, source and category drop out below 960px). Category is the Nexus category name from the cached page details, an em dash for other sources or before they are read: opening the list shows every cached category at once without a network call, then reads the missing Nexus mods one at a time while signed in. Both share one details sidebar for the selected mod (NOMAD, 2026-09-30).
- Grid: a responsive grid of mod cards (`minmax(300px, 1fr)`; the 300px details aside is always beside it at normal width, so the default 1280px window shows two columns and three appear from about 1470px at the default sidebar width; the minimum window, where the aside becomes a drawer, shows two).
- A card: the mod's icon (its Nexus `picture_url`, fetched once when the mod is installed and cached on disk, in place of the letter tile; a letter tile when there is none, NOMAD 2026-09-29), its name, author and version from its manifest, and badges for an available update or a problem. The card view is deliberately simple: no switch (the enable toggle lives in the sidebar). Clicking a card selects it and fills the sidebar. One card per mod; Remove acts on the whole archive entry the mod came in (its confirm names the other mods in it).
- **Mod menu** (NOMAD, 2026-09-30): right-clicking a card or a list row opens Mortar's own menu (`ModMenu.tsx`), and so do the Menu key or Shift+F10 on a focused card or row; the card's ⋯ button opens the same list. Actions, from one list in `modActions.ts`: Enable or Disable (the label follows the state), More details, Open on Nexus or Open on GitHub (by the page's host; only when the mod has a page), Show files, and after a divider Remove, in red (absent for the bundled SMAPI entry; it opens the confirm dialog). Enable, Disable and Remove are disabled while the profile is locked.
- Above the grid: a search field that filters by name, the problem summary with one-click fixes ([architecture.md](architecture.md#mod-data)), and buttons to add mods (open Nexus, or pick an archive). Dropping an archive on any page of the game (profiles, settings included) installs it into the open profile.
- SMAPI's bundled Console Commands and Save Backup, and Mortar's console bridge, are installed in every profile and not shown, listed, counted or removable (NOMAD, 2026-09-30).
- Empty (a new profile): "No mods yet", a line on how to add them, **Browse Nexus** and **Add archive**, **Or import a shared profile**, and a note that archives can be dropped anywhere. The hero uses Steam's art.

## Mod detail

- **Sidebar** (both views, 300px on the right; a right drawer below 960px): the mod's picture (letter tile when none), name, author · source, the endorsement count, the enable toggle at the top right (its aria-label states the mod; no "Enabled in" text), Version, UniqueID, and for a Nexus-installed mod what its cached page says, once it arrives and without holding up the rest: the summary (two lines) and a two-column grid of Latest on Nexus (in the accent when newer than the one in use), Category, Downloads (compact) and Updated, each cut short with the whole value in a tooltip; then the update banner and the problem line when there are any, the mods from the same download, and at the foot **More details** (primary), **Show files** and **Remove**. Remove is absent for the bundled SMAPI entry. Selecting the selected mod again keeps its fetched data. The sidebar, the list and the dialog share one read of a mod's Nexus details per session.
- **More details** opens a modal dialog titled with the mod's name and closed with **Close**: the page link (GitHub or Nexus), **Needs** (each dependency and whether it is installed), **In the same download** (mods that update and roll back together), **Versions** (the one in use and the one kept for rollback, with Roll back, which asks first), **Settings** (`config.json`, whether it was changed, Open and Reset, which asks first) and **Needed by**. A Nexus-installed mod also gets, under the page link, what its Nexus page says (from the cache; signed out with nothing cached, a line asking to sign in): adult-content and not-available chips when they apply, the summary, a three-column grid (Latest on Nexus, in the accent when newer than the one in use; Category; Uploaded by, linking the uploader's profile; Endorsements; Downloads with unique downloads; Created · updated), the installed file (name, version · size · upload date, its description), and three folds closed by default: **Current files on Nexus** (the first five, newest first, without old or archived files), **Recent changes** (the last five changelog versions) and **Description**. BBCode is reduced to paragraphs, headings, list items, bold, italic and http(s) links opened in the browser; images, video and every other tag's markup are dropped, and no HTML from Nexus is rendered.

## Saves tab

Every save in the Saves folder with its fit for this profile: a coloured edge and a chip ("All mods present", "Has used N mods it lacks"), the mods it has used as removable chips (dismissing one for that save), and Add to this profile for mods Mortar can install. A line explains that the save is picked inside the game.

## Notes tab

A plain text area for the profile's notes, saved automatically ("Saved · 2 min ago"); notes travel in the `.mortar` file but not in links.

## In-between moments

- **Dropping an archive:** while an archive is dragged over the window, a dashed drop zone filling the window 12px in from its edges says where it goes ("Drop to install into Cookie farm"), with the file name and the supported formats.
- **A Nexus link:** while a game's profile screen is open, the link installs into the open profile at once, with a toast naming the mod ("Downloading <mod> into <profile>"). On any other screen a dialog, "Install <mod>?", asks which profile: the open one (primary), Other profile… (a menu of the others) or Ignore; further links queue behind it. While the window is minimised a desktop notification says the same, and its click (Show) raises the window. The mod's name is its Nexus page title, with "Nexus mod <id>" when signed out or offline. A refused link (wrong game, another account, expired, malformed) is a toast saying why.
- **Two copies of one mod:** a dialog shows both (source, version, what depends on it), preselects the newer or Nexus-sourced one, and switches the other off rather than deleting it; Decide later leaves both as a problem.

## Console tab

- SMAPI's log as it is written, monospace, in columns: a level bar, time, level, mod and message; warnings and errors get a tinted row.
- Filters: a search box, toggles for SMAPI's six levels (Trace, Debug, Info, Warn, Error, Alert; `Pathoschild/SMAPI` `src/SMAPI/LogLevel.cs`) each with its line count (Trace and Debug off by default), a mod picker whose choices show as removable chips, "Showing X of Y lines" with Clear filters, and Jump to first error. Toggles for timestamps and follow-tail (on by default).
- **Copy**, and **Get help**: shows the log with its local paths and asks before uploading it to smapi.io/log, then copies the link and offers Open.
- Before the first launch: a line saying the console fills when the game runs.
- **Input line** at the bottom (monospace, prompt `>`): Enter sends the command to the running game through the Mortar SMAPI Bridge and echoes `> <command>` into the log as a Mortar line; the output arrives with SMAPI's own lines, and Follow is switched back on so it scrolls into view. Up and Down browse the last 100 commands of that game (in memory only). Disabled with the hint "Start the game to run commands" while the game is not running; a failed send shows a toast.

## Profile management

A page, as Concrete's: a header with a back button, **Import** and **New profile**; beside the list, a **Recently deleted** panel (profiles in the 30-day trash, each with its days left and Restore).

- **Rows:** every profile as a sortable row: drag handle, name (with a Hidden chip when hidden) and its summary, **Share**, and a ⋯ menu holding Rename, Duplicate, Hide from sidebar (or Show) and Delete. Reordering uses dnd-kit's sortable list with a drop indicator.
- **Delete** asks first and says the profile stays restorable for 30 days.
- **Share** opens a dialog with two tabs, **Link** and **.mortar file with settings**. Over about 240 mods it suggests the file.
- **Link tab:** the `https://mortar.rethunk.tech/stardew/p#...` link with Copy link, a meter of its length against Discord's 2,000 characters, Copy as a message, the included mods by source, and what is left out (local archives, switched-off mods). Beside it, a preview of the page the recipient sees.

## Import

A wide "Import profile from…" dialog over the dimmed game screen, with tabs **Link** and **.mortar file** (NOMAD, 2026-09-29: archives go into an existing profile through Add archive or a drop, not through Import). Nothing downloads before the user confirms.

- A dense grid of mod tiles: icon with an include tick, name, author (with "different file" or "unverified" where they apply), and on the right the mod's import state (installed, download, dependency, check later, unavailable) in place of a version number. An unticked mod is left out.
- A status bar: "Ready to import", the profile name, the mod count, the counts per state, and the approximate download size (the sum of each file's `size_kb` from Nexus).
- Problems found before download, one compact item each with its own action: a mod removed from Nexus (with its page), a mod broken for this game version (from SMAPI's update API, with Leave out), missing dependencies (from the mod dataset); and for a free account, an item saying each download takes one click on Nexus.
- **Reset**, **New profile from link** (**from file** on the file tab), which creates the profile (named "<name> (2)", the next free number, when that name exists, and the toast says so) and fills the download queue (the queue sheet opens only when something was queued), and **Add to <profile>** when Import was opened from one; closing leaves nothing behind. When the import needs Nexus downloads and no account is signed in, a banner offers Nexus settings and the import buttons stay off. The Link tab reads the clipboard only from **Paste from clipboard**; a `mortar://` link, a dropped `.mortar` file or a second launch opens the dialog with it filled in and previewed.

## Download queue

A side sheet. The header gives totals (done, in progress, failed, left, size) over a progress bar split by state, with **Pause all**, which stops new downloads from starting and lets the one in flight finish. Sections, top down: **Needs your click** (free accounts: the head item with Open download page; the next page opens when it finishes), **Choose a file** (a GitHub release with several archives: one button per archive and Skip), **Check this download** (a GitHub download whose mod SMAPI's update API does not tie to its repo: "This download's mod is not known to come from <owner>/<repo>", with Install anyway and Skip), **Failed** (with Retry failed; each row says why and has Retry and Skip on its right edge), **In progress** (size, speed and Cancel per row; Cancel is offered on a download in progress, not during install), **Up next** and **Done** (both collapsed to one line). Every row's action sits on its right edge. Premium accounts download without clicks. A rate limit, Nexus's or GitHub's, shows when downloads resume. A Done row whose GitHub source could not be verified says so. Closing the sheet keeps the queue running.

Update review offers **Update** per mod and **Update all**, with **Open page** kept as the secondary action. The problem bar's missing dependency offers **Add**, with **Open page** kept. While signed out, both show a toast linking to Settings › Nexus Mods.

## First run

Only when Stardew Valley is not found, or on a first launch (SMAPI never installed and no profiles); never otherwise:

1. Find Stardew Valley in Steam and show what was found, with **Browse** for another folder; when nothing is found, say so, with Browse and a retry.
2. Install SMAPI if it is missing: it starts by itself, unattended, with no button (NOMAD, 2026-09-30), and a failure shows its error with **Retry**. On Windows with Steam, show the launch-options line to paste into Stardew's Steam properties, with a copy button.
   Installing SMAPI shows its progress as steps (downloaded, files added, launcher replaced, bundled mods added).
3. Create the first profile: two cards, **Start empty** (with a name field) and **From a shared link** (with a link field, which asks for the Nexus sign-in first when the import needs downloads).

## Confirmations

Short toasts, bottom right, stacked (at most three; the oldest goes first), each with a coloured edge by kind and a dismiss button, dismissing themselves after 5 s (10 s for errors and warnings), with hover pausing the countdown, and an undo-style action where one exists: "Link copied", "SpaceCore installed" (Undo), "… rolled back" (Redo update), "Couldn't reach Nexus" (Retry now). A toast about a mod shows its picture.

## Settings

Mortar's Settings holds only what is Mortar-wide, never one game's (NOMAD, 2026-09-30). It is a full page with a back button (Esc also returns), reached from the app menu and Ctrl+,, with a section list on the left and the section beside it; while it is open the title bar shows a "Settings" tab and no game tab.

- **Appearance** (first, and the one Settings opens on): accent colour and background.
- **Data:** Mortar's data folder with its total size and an **Open folder** link.
- **Nexus Mods:** the personal API key, or once signed in a green alert with the account name, a Premium or Free chip and **Sign out** inside it (saving it the first time asks whether Mortar should handle `nxm://` links), and the switch **Handle Nexus "Mod Manager Download" links**, which names the app that owns them now, asks before taking them, and gives them back when turned off.
- **Updates:** the installed Mortar version, the check's state and **Check now**, then **Download and install** and **Restart now** as the update progresses; a development build says it does not check. Mod and SMAPI updates stay where they are (each profile's mod list, the game settings), linked from here. Under **Save backups**, the number field **Backups kept** (1 to 50, default 5) sets how many save backups Mortar keeps ([architecture.md](architecture.md#storage)).
- **About:** licences and credits, including SMAPI and the Stardew mod dataset (CC-BY-SA 4.0).

**Game settings** are per game and live on their own page, opened from the game screen by a settings icon button (Lucide Settings2, labelled "Stardew Valley settings") at the right of the tab row. The page is titled "Stardew Valley settings", has a back button to the game (Esc also returns), and the title bar keeps the game tab highlighted. It holds the game folder, found in Steam or chosen, with **Browse…** (a folder dialog, checked before it is saved, with an inline error when the folder is not a Stardew install) and **Use Steam's** while a chosen folder overrides Steam; and the installed SMAPI version with Install, Reinstall or Update, showing the same step checks as the banner.
