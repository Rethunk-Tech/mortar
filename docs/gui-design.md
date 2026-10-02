# GUI design

How Mortar's screens are laid out and styled. Concrete (`LethalModding/Concrete`, archived) is one reference for the look, not a template to copy.

## Window

- **Logo:** the running-bond mark, seven rounded bricks in three courses, filled in the primary colour; used in the title bar at 20px, as the app icon (and the Linux desktop icon) on a dark rounded square, and on desktop notifications.

- Frameless, with a solid window (see [architecture.md](architecture.md#stack)): 1280×720 by default, 768×432 minimum, as Concrete.
- The app draws its own title bar, 36px, marked `--wails-draggable: drag`. Double-click maximises.
  - **Left:** the logo in a darker square and the app name, which open the app menu.
  - **Location tabs**, underlined in the primary colour: "Game Select", or the current game. First run shows a "Setup" tab. While Mortar's Settings page is open the game tab gives way to a "Settings" tab, and the game settings page keeps the game tab highlighted.
  - **Right:** minimise, maximise and close.
  - **App menu**, a side drawer in four groups split by dividers: Settings and Check for updates (opens Settings › Updates and checks at once); Open data folder, Save diagnostics… (a redacted zip for a bug report through the native save dialog, FileArchive icon) and Report a bug (a prefilled GitHub issue); About Mortar and Source code (the AGPL-3.0 source offer, opening `https://github.com/Rethunk-AI/mortar`); then Quit.
  - **Mortar update ready:** when a background self-update is staged, a full-width info banner under the title bar says the update applies when Mortar closes and offers **Restart now** (same action as Settings › Updates).
  - **What's new:** after an upgrade, a dialog once per version shows that release's GitHub notes with **Got it**; offline skips until a later start ([architecture.md](architecture.md#release)).
  - **Drawer foot:** the Nexus account, as the account name with a Premium or Free badge and Sign out, or "Not signed in" with Sign in, which opens Settings › Nexus Mods.
  - **Frame:** a 1px light border and rounded corners, since GNOME draws no shadow for a frameless window.
- `user-select: none` everywhere except text fields and the console.
- Menus (the mod menu, the app's other menus) use a dark paper, `rgba(28,28,34,0.99)`, with a 1px light border and 8px corners (`MuiMenu` in `frontend/src/theme/theme.ts`); items with an obvious icon carry a Lucide one.
- **Command palette:** Ctrl+K opens a solid dialog with a search field over actions (Play, Check for mod updates, Open Downloads, Import, Share, New profile, Stream overlay), destinations (this game's profiles, the open profile's user mods and each Settings section) and every keyboard shortcut. Matching is fuzzy, arrows move, Enter runs and Esc closes from anywhere while it is open (including when the search field is not focused). Focus lands in the search field when it opens. Choosing an item closes it. SMAPI's bundled mods and the Mortar bridge are omitted, as on the Mods tab. Like the other shortcuts it stays silent while a field has focus or a dialog is open, and Esc always works.
- Icons come from one set, Lucide (MIT), at 1.5-2px stroke; no hand-drawn or mixed icons.
- Every focusable control shows a visible focus ring when reached by keyboard: a 2px outline in the primary colour, 2px outside the control.
- Button and chip labels never wrap (`white-space: nowrap`); a long message gets its own full-width row, truncating with an ellipsis rather than squeezing the buttons beside it.

## Surfaces and colour

- No gradients anywhere, not as backgrounds, scrims over art, fallbacks or placeholder art: legibility over art comes from a solid layer with alpha, and a missing image becomes a solid tone. The one exception: an image's edge may fade to transparent via an alpha mask; no coloured gradients.
- The window is solid, an opaque `rgb(25,25,30)` base. What shows behind the surfaces is the wallpaper backdrop under a static `rgba(25,25,30,0.8)` tint. Surfaces keep their alpha and composite over it: background `rgba(25,25,30,0.80)`, paper `rgba(50,50,60,0.80)`, the app drawer `rgba(40,40,48,0.92)`; dialog paper (`MuiDialog` in `frontend/src/theme/theme.ts`, `rgb(40,40,48)`) and the download queue sheet are solid (`rgb(40,40,48)` and near it) so the text they cover never shows through; dialog and drawer backdrops a static `rgba(0,0,0,0.30)`; dialogs and their backdrops open and close without a transition, the launch overlay `rgba(0,0,0,0.80)`.
- **Background**, Settings › Appearance, a three-way toggle, all opaque and applied at once. Stored as `background` and `backgroundImage` in `settings.json`.
- **Keep Mortar in the tray** (Settings › General, default off): when on, the window close control and the frameless close button close the window but keep Mortar running, Show Mortar opens a fresh window that the desktop places as new, Wails' system tray shows the app icon with **Show Mortar**, up to three **Play** entries for the most recently launched profiles (disabled while the game runs), a **Stardew Valley is running** status line while it runs, and **Quit**; clicking the icon shows or raises the window. When a Mortar-started run ends, a desktop notification says **Game closed** or **Stardew Valley crashed** with how many mods logged errors; clicking it raises the window on that profile's Console. Helper text notes that GNOME may need the AppIndicator extension. When off, there is no tray icon and close quits.
- **Include beta releases** (Settings › Updates, default off): when on, Check for updates may offer Mortar nightlies published as GitHub prereleases with a signed manifest; when off, only the latest stable release is considered.
  - **Image** (the default): the chosen image, else the system Fedora wallpaper, else the bundled copy. Shows a preview with **Choose image…** (PNG, JPEG or WebP) and **Reset to default**.
  - **Desktop:** the user's own desktop wallpaper, read at runtime (GNOME `org.gnome.desktop.background` `picture-uri-dark` under the dark colour scheme, else `picture-uri`; Windows the `HKCU\Control Panel\Desktop` `Wallpaper` value, measured on Windows 11 26H2 on 2026-10-01; a wallpaper changed while Mortar is open shows at the next mode change), falling back to Image's choice when it cannot be read.
  - **Solid:** no image, the plain base.
  - The window loads the backdrop from `/background?mode=` with `mode` equal to the stored `background` value. The endpoint answers `Cache-Control: no-store`.
  - Image and Desktop draw the wallpaper full-window behind everything, the title bar included, `object-fit: cover`, fully opaque under the 0.8 tint, and it never fades or filters.
  - **Default wallpaper:** Fedora 44's `f44-01-night` (Fedora Design Team, CC-BY-SA-4.0, credited in About). Mortar uses the system copy, `/usr/share/backgrounds/f44/default/f44-01-night.jxl`, when it exists on Linux (WebKitGTK 2.54 decodes JPEG XL, measured), else a bundled 2560px JPEG.
- Text: primary `rgba(255,255,255,0.90)`, secondary `rgba(225,225,230,0.95)`. Links are secondary text with a dotted underline.
- Status colours from Concrete: info `#2B8BDA`, success `#0CDF64`, warning `#F3B416`, error `#C70A0A`. The primary colour is the user's choice, in Settings › Appearance: Sand `#D6B17A` (the default), Moss `#93B86A`, Copper `#D98C5F` or Sky `#79AEDC`, applied at once everywhere the primary appears. It is the same for every game; game colour appears only in hero art.
- Scrollbars are themed: 0.5em wide, in the primary colour at alpha 0.08 for the track, 0.45 for the thumb and 0.7 on hover.

## Type

- Open Sans (300, 400, 500, 600, 700) from `@fontsource/open-sans`, `htmlFontSize` 18 with MUI's responsive font sizes, as Concrete.
- Every string goes through Lingui.

## Game Select

Mortar opens on the last game used; Game Select shows at start only until a game has been opened, and otherwise from the title bar's Game Select tab.

Full-width banner rows stacked down the window, one per game, about 300px tall at 1080p; rows share the window's height equally. The title bar's game tab returns here.

- **Art:** the game's key art edge to edge under a light solid dim layer, with plain text straight on it kept legible by a text shadow and no panel behind. Art is Steam's `library_hero.jpg` read at runtime, never bundled.
- **Left:** the game name with "<loader> | <store>" under it ("SMAPI 4.5.2 | Steam": the loader's installed version, or its name alone before it is installed), and a status line: **Not set up · Open it to set it up** until the game has a folder and (for Stardew) a loader or a profile, else the profile count ("Installed · N profiles"). When that game has a last-played profile that still exists and setup is finished, the loader line continues with "last played" then the profile name and a relative time. If the stored profile is gone, the extra line is omitted.
- **Right:** square badges for its mod sources, each the source's logo above its name (Nexus's own mark; GitHub and Thunderstore from Simple Icons), and when a last-played profile still exists and the game is set up a large contained **Play** button the same height as the badges that opens the game on that profile and starts Play (the same Start path as the sidebar Play button, including SMAPI install and Steam checks).
- **The row that opens on click** has a primary-colour left edge. A live game's row is always clickable: not set up opens that game's setup; set up opens its main screen. Art for a row that cannot open (Lethal Company in v1) is desaturated under a darker dim. Game status is loaded again when the window regains focus.
- **No game found:** when no supported game is installed in the launchers, a footer line **No supported game was found in your launchers.** with **Launchers…** opening Settings › Launchers.
- In v1 Stardew Valley is the only live row; Lethal Company shows dimmed as coming later.

## Main screen

A grid of a resizable sidebar and a detail pane, as Concrete's dashboard:

- **Sidebar** (150-300px, resized from an 8px handle on its right edge, 2px primary right border): a "Profiles" header that opens profile management, then one row per profile in the user's order, the open one selected. Each row shows a problem or update badge when it has any, and a small coloured mark with the profile's icon when it has either.
- **Bottom block** pinned under the sidebar: a **New profile** row at the foot of the list, icon buttons for Support (a menu: Get help, which opens the Console tab's upload, and Report a Mortar bug) a Downloads icon that opens the download queue (with a count while it has items) and a Notifications bell with an unread count, which opens a solid dark panel of this session's recent toasts, newest first, with **Clear** and "No notifications yet" when empty; an action in it, such as Undo, is offered only while it still applies. Then a full-width **Play** button in the primary colour, split with a caret (More play options) whose dark menu holds **Play without mods** (Lucide Gamepad2); in the compact layout the menu opens on right-click only. Play without mods starts the game with no profile and does not lock mods. On Windows, when Steam's launch options run SMAPI, a dialog says "Steam will still start SMAPI" before offering Cancel or **Play without mods**. There is no Settings icon here: it displaced Play, and Settings stays in the app menu and on Ctrl+,.
- **Detail pane** for the open profile:
  - A hero, 190px tall in every view and tab ( only the minimum-size layout folds it to one line), bleeding to the pane's edges.
    - **Cover:** the profile's cover image at full opacity under a light solid dim layer (`rgba(20,20,24,0.18)`), its alpha fading to transparent over the bottom 20% by an alpha mask, so the art reaches the top of the tab row so the backdrop shows through into the tab row (the hero has no background of its own).
    - **Cover source, in order:** an image the user picked, else the Nexus picture of the profile's most-endorsed mod (`endorsement_count` from the mod endpoint), else Steam's own hero art for the game, else a solid tone; never a random image. Steam's art is read at runtime from `<Steam>/appcache/librarycache/413150/library_hero.jpg` (Steam also keeps a `library_hero_blur.jpg`) and never bundled with Mortar.
    - **Text:** the profile name large and bold with a soft white glow, and always the same cards (Mods, Updated, Created) so the name sits at the same place for every profile. Beside the name sit Rename, **Edit profile** (Palette icon), the cover image menu and a ⋯ **Profile menu** holding **History**, **Export profile…** (native save dialog), **Add a shortcut that plays this profile** (toast **Shortcut added** with the path) and **Add this profile to Steam** (toast **Added to Steam** or **Already in Steam**; errors include Steam still running). Under the name, the profile's description shows as one truncated line with the whole text in a tooltip, when it has one (hidden in the compact layout).
  - A **Saves** card in the hero with how many saves fit this profile ("2 of 4", "0 of 0" when there are none), opening the **Saves** tab:
    - each save in the Saves folder with its fit: "all mods present" or "has used N mods this profile lacks" with their names, from the save scan ([architecture.md](architecture.md#saves))
    - a mod can be dismissed for a save
    - Players pick their save inside the game; Play also warns from the newest save ([architecture.md](architecture.md#launch)).
  - Tabs: **Mods** (default), **Problems** (a count chip when the open profile has any problems), **Saves**, **Notes**, **Console**, then 32px bordered icon buttons with tooltips at the right of the row: a **Tools** menu (Wrench; lists configured tools for one-click launch, **Add tool…** and **Manage tools** for edit and remove, with **Add tool…** and a line saying what tools are for when none exist; the editor uses the native executable picker and documents placeholders), **Share**, and the game settings icon at the end of the tab row.
- **Launching** covers the whole window with the launch overlay, the title bar inert except its window controls:
  - a spinner, "Launching Stardew Valley", and SMAPI's first log lines as they arrive (started, mods loading, mods it will skip), with Open console and Hide, until the game is up or the launch fails
  - a launch where SMAPI exits without writing a log fails with its exit code
  - While the game runs, the Play button becomes a Running status with the elapsed time and a **Stop game** button, and the open profile's mod controls are locked ("Stop the game to change mods."): switching, removing, installing and dropping archives
  - While Mortar installs SMAPI by itself, Play reads "Installing SMAPI…"
  - When the installed Stardew version differs from the one the last launch recorded, a dialog, "The game was updated", says which version the profile last launched on and which is installed, lists the profile's mods SMAPI's API marks broken for the new one (or that none are), and offers **Cancel**, **Open problems** (the Mods tab) and **Play anyway**; a first launch does not ask
  - When the newest save uses mods this profile lacks or has switched off, a dialog, **Your last save needs other mods**, names the farm and lists those mods (**Not in this profile** or **Switched off in this profile**), with **Cancel**, **Open saves** and **Play anyway**; an unreadable save does not ask
  - When a launch Mortar started ends with errors or a SMAPI crash, a dialog, "Stardew Valley closed with errors", lists the mods that logged errors ("mod · N errors", each with its first message) with **Open Console**, **Get help** and **Dismiss**. When that list is empty, **Find the mod causing this** starts a crash bisect ([architecture.md](architecture.md#launch)).
- **Compact layout** (below 960px wide; the minimum window is 768×432): the sidebar collapses to a rail of profile initials with Play at its foot, the hero gives way to a one-line header (the name and "N mods · N updates · N problems", a part that is zero left out), the mods search folds into a filter icon that expands into the field, other actions fold into icon buttons, and the mod grid drops to two columns.

## Mods tab

- Two views, switched from the toolbar and remembered per user: **Grid** (default), and **List**, a table. The last loaded list for a profile stays on screen when leaving the tab and returning; it is refreshed in the background. The first load for a profile shows eight skeleton rows (`Loading mods`). Defaults shown are On, Name, Version, Author, Source, Category and Status. Further columns (Latest on Nexus, UniqueID, Endorsements, Downloads, Updated on Nexus, Installed, Needs, Notes and tags, Last run) start hidden; Size on disk is omitted because the store index does not keep a byte size. Cells never wrap (ellipsis, full text in `title`). Author, Source, Category and the extra columns drop out below 960px even when enabled. Category is the Nexus category name from the cached page details, an em dash for other sources or before they are read: opening the list shows every cached category at once without a network call, then reads the missing Nexus mods one at a time while signed in. Latest on Nexus uses the accent when newer than the installed version (`isNewer`). **Column menu:** right-click any header for a dark menu that shows or hides each column (On and Name stay on); **Reset to default columns** restores the defaults. **Sort:** left-click a header (not On) sorts ascending, click again reverses; an arrow marks the sorted header. **Reorder:** drag a column header (or focus it and use Space and the arrow keys); the whole table, rows included, takes the new order live while dragging, the header keeps its slot with a primary outline and tint while a ghost of it follows the pointer, Escape cancels, and dropping saves the order to `listColumns`. Column choice, order and sort are stored in `settings.json` (`listColumns`, `listSortColumn`, `listSortDir`) and apply to every profile. Both views share one details sidebar for the selected mod. A pinned mod carries a pin icon in the Version column and on its card. Last-run error and warning badges appear in the list, the grid and the details sidebar.
- **Group by:** a menu beside the view toggle (None, Status (default), Category, Source, Tag, Framework, Author), kept as `listGroupBy`. **Edit categories…** at the foot opens a dialog to add, rename, recolour (profile palette swatches) or delete custom categories stored per game under `<datadir>/categories/<game>.json`; deleting one moves its mods to Uncategorized. Collapsible headers with counts appear in the list and the grid; collapsed groups are remembered per game. Sort is within each group. Status uses Problems, Update available, Enabled, then Disabled, first match. Category uses each mod's primary category: a per-entry override (custom category or Nexus category name) when set, else the Nexus category from cached page details. Framework puts content packs under the framework they are for (ContentPackFor UniqueID, or the installed framework's name) and other mods under SMAPI mods. A mod with several tags appears under its first tag, which a tooltip says.
- **Bulk select** (list and grid): click selects a row or card and fills the sidebar; Ctrl or Cmd click toggles; Shift click selects a range in the current order (the list's sort, or the filtered order in the grid); Ctrl+A selects every visible filtered mod and Esc clears. With two or more selected, a bar above the mods shows the count and **Enable**, **Disable**, **Remove** (one confirm for all), **Share selection** and **Clear**. Enable, Disable and Remove are locked while the game runs the profile.
- Grid: a responsive grid of mod cards (`minmax(300px, 1fr)`; the 300px details aside is always beside it at normal width, so the default 1280px window shows two columns and three appear from about 1470px at the default sidebar width; the minimum window, where the aside becomes a drawer, shows two).
- A card: the mod's icon (its Nexus `picture_url`, fetched once when the mod is installed, cached on disk under `cache/` and served to the window at `/mod-picture/`, in place of the letter tile; a letter tile when there is none), its name, author and version from its manifest, and badges for an available update, a problem, a Nexus page that is hidden or removed, errors or warnings from the profile's last stored run, or a pin, a small first-tag chip when it has tags, and a primary-colour New dot on the letter tile when its Nexus page has a newer current file or changelog version than the user last looked at ([architecture.md](architecture.md#nexus-mods)). The card view is deliberately simple: no switch (the enable toggle lives in the sidebar). Clicking a card selects it and fills the sidebar. One card per mod; Remove acts on the whole archive entry the mod came in (its confirm names the other mods in it).
- **Mod menu**: right-clicking a card or a list row opens Mortar's own menu (`ModMenu.tsx`), and so do the Menu key or Shift+F10 on a focused card or row; the card's ⋯ button opens the same list. Actions, from one list in `modActions.ts`: Enable or Disable (the label follows the state), More details, Open on Nexus or Open on GitHub (by the page's host; only when the mod has a page), Show files, Open manifest.json, **Set category…** (Uncategorized, each custom category, and the mod's Nexus category when known), Pin this version or Unpin, Skip this update or Show skipped update (only when an update is offered or one is skipped), and after a divider Remove, in red (it opens the confirm dialog). Enable, Disable and Remove are disabled while the profile is locked.
- Above the grid: a search field that filters by name or author, a **Configurable** toggle for mods that have a `config.json`, the problem summary with one-click fixes ([architecture.md](architecture.md#mod-data)), and buttons to add mods (open Nexus, or pick an archive). Dropping an archive on any page of the game (profiles, settings included) installs it into the open profile. After an install finishes, a dialog ("<mod> needs <A> and <B>") offers **Add them** for required dependencies the profile still lacks, or **Not now**; optional ones are left to the Problems tab.
- SMAPI's bundled Console Commands and Save Backup, and Mortar's console bridge, are installed in every profile and not shown, listed, counted or removable.
- Empty (a new profile): "No mods yet", a line on how to add them, **Browse Nexus** and **Add archive**, **Or import a shared profile**, and a note that archives can be dropped anywhere. The hero uses Steam's art.

## Mod detail

- **Sidebar** (both views, 300px on the right while a mod is selected, absent otherwise so the list takes the full width; a right drawer below 960px):
  - Picture (letter tile when none), name, author · source, endorsement count, enable toggle at the top right (its aria-label states the mod).
  - Version, UniqueID.
  - For a Nexus-installed mod, once cached: **Removed from Nexus** or **Hidden on Nexus** (with the date) when the page is unpublished; a two-line summary; a two-column grid of Latest on Nexus (accent when newer), Category, Downloads (compact) and Updated, each truncated with a tooltip.
  - Update banner and problem line when present; **Pin this version** or Unpin; **Skip this update** or Show skipped update.
  - **Note** (500 characters) and **Tags** (up to 8, suggestions from tags already used in the profile).
  - **Dependencies:** indented tree of what the mod needs and what in the profile needs it, followed transitively, cycles shown once; each node is enabled, switched off, missing or broken; a link selects that mod; a missing one offers Add and Open page.
  - **Also in these profiles:** other profiles of this game whose `profile.json` names this UniqueID, each a link with version and enabled state.
  - Mods from the same download.
  - Foot: **More details** (primary), **Show files** and **Remove**.
  - Selecting the selected mod again keeps its fetched data. The sidebar, the list and the dialog share one read of a mod's Nexus details per session.
- **More details** opens a modal titled with the mod's name and closed with **Close**:
  - Page link (GitHub or Nexus); **Needs**; **In the same download**; **Versions** (in use and rollback, with Roll back after confirm); **Settings** (`config.json`: Edit, Open, Reset); **Needed by**.
  - Edit maps the file to switches, number and text fields, string chip lists and collapsible object groups; null and mixed arrays stay read-only with **Open in editor**; **Save** writes and shows "Saved".
  - For a Nexus-installed mod: cached page chips, summary, a three-column grid (Latest, Category, Uploaded by, Endorsements, Downloads, Created · updated), the installed file, Endorse/Abstain and Track/Untrack when signed in, **New since you last looked**, and folds **Current files on Nexus**, **Recent changes** and **Description**.
  - BBCode becomes paragraphs, headings, list items, bold, italic and http(s) links; other markup is dropped.

## Saves tab

Every save in the Saves folder with its fit for this profile:

- a coloured edge and a chip ("All mods present", "Has used N mods it lacks")
- the mods it has used as removable chips (dismissing one for that save)
- Add to this profile for mods Mortar can install
- a line explaining that the save is picked inside the game
- a card's second line reads `farmer · type farm · season day, year N · Nh played · goldg · last played date`, leaving out a part the save does not give

**Save backups** above the cards opens a dialog listing the backup zips newest first, each with its time, cause ("Before updating" the named profile, "Before a restore" or "Unknown"), size and the farms inside, with **Restore** (a dark menu: Restore all, or each save of that zip alone), **Open backups folder** and **Close**. Restore asks first, names the saves it will overwrite and says the current Saves folder is backed up first (solid paper, no transition), and reads "Stop the game to restore saves." while the game runs.

## Problems tab

A full-height scrollable list of every problem for this profile, grouped under headings (only non-empty groups shown):

- Missing requirements
- Conflicts
- Broken or outdated mods
- Errors in the last run
- Changed outside Mortar
- Duplicates
- Settings (Content Patcher compatibility setting suggestions)
- Cosmetic or harmless (never counts toward the tab's chip, the profile badge or the Mods-tab row)
- Cleanup (unused frameworks; never counted)

While the tab is open, a **Copy all problems** icon button in the tab row (where the Console keeps its log actions) copies every group as plain text: the heading, then one "- " line per problem with its author note indented under it, including Cleanup.

Each row shows the full wrapped text, severity icon, the Nexus author note when a listed requirement has one, and its fix actions. Cosmetic or harmless rows use small outlined inherit fix buttons; real conflicts keep the contained warning style. Cleanup rows read "<name>: Not needed by any enabled mod" with **Remove**.

- Loading: the same "Checking the mods for problems…" line as the summary
- None: "No problems found."
- Some checks could not run offline: the connection warning from the summary

On the Mods tab, problems take one clickable row (warning colour when any problem is a warning): the count, the first warning's text truncated, and **Open Problems**; clicking anywhere on it opens this tab.

## Notes tab

A header row like the Saves tab's explains that notes travel in a shared `.mortar` file, not in a share link, with the save state at its right ("Saved · 2 min ago", or **Retry** after a failed save); below it, a plain text area for the profile's notes, saved automatically. What a share carries is in [architecture.md](architecture.md#sharing).

## In-between moments

- **Dropping an archive:** while an archive is dragged over the window, a dashed drop zone filling the window 12px in from its edges says where it goes ("Drop to install into Cookie farm") and the supported formats. It shows no file name: WebKitGTK and Wails expose none before the drop.
- **A Nexus link:** while a game's profile screen is open, the link installs into the open profile at once, with a toast naming the mod ("Downloading <mod> into <profile>"). On any other screen a dialog, "Install <mod>?", asks which profile: the open one (primary), Other profile… (a menu of the others) or Ignore; further links queue behind it. A link never brings an open window to the front, so clicking downloads in browser tabs keeps the browser in front; only a window closed to the tray is reopened. While the window is minimised a desktop notification says the same, and so does the profile question while the window is behind another; its Show action raises the window. The mod's name is its Nexus page title, with "Nexus mod <id>" when signed out or offline. A refused link (wrong game, another account, expired, malformed) is a toast saying why.
- **Two copies of one mod:** a dialog shows both (source, version, what depends on it), preselects the newer or Nexus-sourced one, and switches the other off rather than deleting it; Decide later leaves both as a problem. When exactly one copy is from Nexus, **Keep the Nexus copy** is the recommended (contained) action; **Keep this one** keeps the selected non-Nexus copy.

## Console tab

- SMAPI's log as it is written, monospace, in columns: a level bar, time, level, mod and message; warnings and errors get a tinted row.
- Filters: a search box that grows to fill the row, toggles for SMAPI's six levels (Trace, Debug, Info, Warn, Error, Alert) each with its line count (Trace and Debug off by default), a mod picker whose choices show as removable chips, and Clear filters; icon toggles (tooltip and aria-label, aria-pressed for state) for timestamps (Clock) and following new lines (ArrowDownToLine, on by default) at the right of the filter row.
- Log actions are icon buttons beside the tabs, each with a tooltip and aria-label, so the tab row does not wrap:
  - Jump to first error (CircleAlert)
  - Clear (Eraser)
  - **Copy** (Copy)
  - **Save log…** (Download): a native save dialog, default name `SMAPI-<profile>-<date>.txt`; the raw SMAPI log file when this profile owns it, otherwise the Console's lines; off when there is nothing to save
  - **Get help** (LifeBuoy): shows the log with its local paths and asks before uploading it to smapi.io/log, then copies the link and offers Open
- **Runs** picker, on the filter row after the mod picker (its button reads This session or the run shown):
  - This session, or one of the profile's last 20 recorded launches (outcome Ran, Crashed or Failed, with its time)
  - a past run loads its stored log read-only, and Save log and Get help use it
  - "No recorded runs yet" before the first
- **Links:** in visible rows only, an exact installed mod name or `UniqueID` is an underlined link, in the row's own colour, that selects the mod, and a path under the profile's `mods/` folder or the game folder opens its folder.
- Before the first launch: a line saying the console fills when the game runs.
- **Reinstall:** when SMAPI's own log says it is incompatible with the game's version, an offer to reinstall SMAPI appears in the console.
- **Input line** at the bottom (monospace, prompt `>`):
  - Enter sends the command to the running game through the Mortar SMAPI Bridge and echoes it into the log as a Mortar line prefixed with `>`
  - the output arrives with SMAPI's own lines, and Follow is switched back on so it scrolls into view
  - Up and Down browse the last 100 commands of that game (in memory only)
  - Disabled with the hint "Start the game to run commands" while the game is not running; a failed send shows a toast

## Profile management

A page, as Concrete's:

- Header: back, **Import from the game's Mods folder**, **Import**, **Restore from zip…**, **New profile**.
- **Find a mod in all profiles** lists matches by name or UniqueID (profile, version, enabled state); each link opens that profile with the mod selected.
- **Recently deleted:** 30-day trash, days left and Restore.
- **Import from the game's Mods folder** previews name, version and source (switched-off copies marked, skipped/failed with reason; bundled SMAPI and the bridge omitted), then creates "Imported mods", opens it, and toasts counts with expandable Details.

- **Change history:** the open profile's History lists a bounded log of mods added, removed, updated, rolled back, or switched on or off, with Revert to restore an earlier state ([architecture.md](architecture.md#profile-operations)).
- **Rows:** every profile as a sortable row: drag handle, name (with a Hidden chip when hidden) and its summary (mod count, then cached update and problem counts when Mortar already knows them, then origin when the profile was imported from a link, a `.mortar` file, the game's Mods folder, or duplicated as a copy of the source profile's name), **Share**, and a ⋯ menu holding Rename, **Edit profile**, **Export profile…**, Duplicate, Compare with… (another profile of the same game: mods only in A, only in B, in both with a different version, in both with a different enabled state, and identical mods in a collapsed list; each differing row has **Copy to** the other profile through the store), Hide from sidebar (or Show) and Delete. Reordering uses dnd-kit's sortable list with a drop indicator. A row carries the profile's colour and icon mark, and its description as one truncated line with a tooltip.
- **Edit profile** (the hero's Palette button or the row menu): a solid-paper dialog without a transition with **Colour** (eight swatches), **Icon** (twelve Lucide icons), **Description** (280 characters), **Launch options** (extra SMAPI arguments; Mortar sets `--mods-path` itself, and `--mods-path`, `--no-terminal` and `--skip-terminal` are refused), **Launch prefix** (direct launches only, for example `gamemoderun mangohud`; unavailable on Windows) and **Launch environment** (one `VAR=value` per line; direct launches only). Steam launches do not receive prefix or environment ([architecture.md](architecture.md#launch)).
- **Delete** asks first and says the profile stays restorable for 30 days.
- **Share** opens a dialog with two tabs, **Link** and **.mortar file with settings**. Over about 240 mods it suggests the file.
- **Copy mod list:** under the tabs, a nowrap button and a dark icon menu choose Markdown, Plain text or Discord (kept for the session); Discord splits into messages of at most 2,000 characters, and when more than one is needed **Copy part N** buttons replace the single copy. Shares from the mods selection limit the link and the file to those mods.
- **Link tab:** the `https://mortar.rethunk.tech/stardew/p#...` link with Copy link, a meter of its length against Discord's 2,000 characters, Copy as a message, the included mods by source, and what is left out (local archives, switched-off mods). Beside it, a preview of the page the recipient sees.

## Import

A wide "Import profile from…" dialog over the dimmed game screen, with tabs **Link** and **.mortar file** ( archives go into an existing profile through Add archive or a drop, not through Import). Nothing downloads before the user confirms.

- A dense grid of mod tiles: icon with an include tick, name, author (with "different file" or "unverified" where they apply), and on the right the mod's import state (installed, download, dependency, check later, unavailable) in place of a version number. An unticked mod is left out.
- A status bar: "Ready to import", the profile name, the mod count, the counts per state, and the approximate download size (the sum of each file's `size_kb` from Nexus).
- Problems found before download, one compact item each with its own action: a mod removed from Nexus (with its page), a mod broken for this game version (from SMAPI's update API, with Leave out), missing dependencies (from the mod dataset); and for a free account, an item saying each download takes one click on Nexus.
- **Reset**, **New profile from link** (**from file** on the file tab), which creates the profile (named "<name> (2)", the next free number, when that name exists, and the toast says so) and fills the download queue (the queue sheet opens only when something was queued), **Add to <profile>** when Import was opened from one, and **Replace <profile>**, which asks first, lists the entries it will remove, keeps local-only mods not in the share, and writes a profile history event so the change can be reverted. Closing leaves nothing behind. When the import needs Nexus downloads and no account is signed in, a banner offers Nexus settings and the import buttons stay off. The Link tab reads the clipboard only from **Paste from clipboard**; a `mortar://` link, a dropped `.mortar` file or a second launch opens the dialog with it filled in and previewed.

## Download queue

A side sheet. The header gives totals (done, in progress, failed, left, size) over a progress bar split by state, with **Pause all** (stops new starts; the in-flight download finishes) and **Clear finished** when any done, failed, cancelled or skipped rows remain (those rows also have a dismiss control; queued and in-progress items stay).

Sections, top down:

- **Needs your click** — free accounts: the head item with Open download page and Skip; the next page opens when it finishes. An update or latest-file item is skipped on its own once the profile already holds that file or a newer one from the same page ([architecture.md](architecture.md#nexus-mods)).
- **Choose a file** — a GitHub release with several archives: one button per archive and Skip.
- **Check this download** — a GitHub download whose mod SMAPI's update API does not tie to its repo, with Install anyway and Skip.
- **Failed** — Retry failed; each row says why and has Retry and dismiss.
- **In progress** — size, speed and Cancel per row (Cancel only while downloading, not during install).
- **Up next**, **Done** (both collapsed to one line), then skipped or cancelled rows.

Every row's action sits on its right edge. Premium accounts download without clicks. A rate limit shows when downloads resume. A Done row whose GitHub source could not be verified says so. The sheet toggles between **Queue** and **History** (the last 1,000 finished items, with **All outcomes** and **All profiles** filters and **Clear history**). A failed download that has a partial file resumes on Retry. Closing the sheet keeps the queue running.

Update review offers **Update** per mod and **Update all**, with **Open page** kept as the secondary action. Unofficial SMAPI versions are labelled unofficial and are never included in **Update all**. Each Nexus update shows the cached changelog versions newer than the installed version and not newer than the latest (latest included), newest first, collapsed with a count ("3 versions of changes"); a mod with no changelog, none in that range, or nothing cached says so. GitHub updates have no Nexus changelog. The Problems tab's missing dependency offers **Add**, with **Open page** kept. While signed out, both show a toast linking to Settings › Nexus Mods.

## First run

Mortar's own setup finds launchers, not games, and opens until it is finished once (`launchersConfirmed`); there is no skip. "Welcome to Mortar" lists each launcher Mortar reads on this OS (Linux: Steam, Steam (Flatpak), Heroic, Lutris, Minigalaxy, GOG; Windows: Steam, Heroic, GOG Galaxy; a folder belongs to one launcher, so Minigalaxy's install folder is not also GOG's) as accordions in a two-column grid on a wide window, one column when narrow, centred vertically when they fit. Each row's summary shows the launcher's logo (from `simple-icons`, or Minigalaxy's own icon as a white silhouette; Flatpak launchers carry a small Flatpak badge; the size of the status icon) on the left, its name, the supported games found in it ("Not found", or "Found, with no supported games yet"), and a large status icon left of the chevron: a green check when found, a grey cross when not. Expanded, a row lists every folder it was found in, or where Mortar looked when it was not, the folders the user added (each with a remove button), **Add folder…** (filled when not found, outlined when found; the folder is checked to be that launcher's) and **Rescan**. When none of the launchers is found, the first row starts open. Detection runs again when the window regains focus. Below, a count ("3 of 5 launchers found", or a note that games can still be set up by folder) and **Continue**, always enabled, which leads to Game Select. Settings › Launchers shows the same rows.

Each game has its own setup, run when it is first opened from Game Select and the game is not set up (folder unknown, or no loader and no profile): "Set up <game>" with steps **Game folder**, the game's own loader step when it has one (SMAPI for Stardew Valley), and **First profile**.

1. Game folder: the game's art (Steam's cache, else Steam's public CDN image, dimmed until found). Found: the store it came from, the folder with **Change…**, the game and loader versions, and **Continue**. Not found: "<game> was not found in your launchers", a list of each launcher and whether it was found and holds the game, **Choose the game folder…** (filled) and a text **Rescan**, with no Continue until a folder is known.
2. Loader (Stardew Valley: SMAPI) if it is missing: it starts by itself, unattended, with no button, and a failure shows its error with **Retry**. On Windows with Steam, **One step in Steam** shows the launch-options line to paste, with **Copy** and **Set it in Steam for me** (disabled once the line is already set; toast **Launch options set in Steam**, or **Could not set it in Steam** while Steam is running).
   Installing SMAPI shows its progress as steps (downloaded, files added, launcher replaced, bundled mods added).
3. Create the first profile: when the game's `Mods` folder holds mods to import, three cards, **Import from the game's Mods folder** first (preview, then a new "Imported mods" profile), **Start empty** (with a name field) and **From a shared link** (with a link field, which asks for the Nexus sign-in first when the import needs downloads). Otherwise the two cards Start empty and From a shared link

## Confirmations

Short toasts, bottom right, stacked (at most three; the oldest goes first), each with a coloured edge by kind and a dismiss button, dismissing themselves after 5 s (10 s for errors and warnings), with hover pausing the countdown, and an action where one exists: "Link copied", "SpaceCore installed" and "Added X to Y" (Undo), "… rolled back" (Redo update), "Couldn't reach Nexus" (Retry now), "N downloads need your decision" (Show). A toast about a mod shows its picture, from the same cache as the cards. Toasts stay in the session's notification history (the sidebar bell, [Main screen](#main-screen); [architecture.md](architecture.md#stack)).

**First-open tips:** the Mods tab, the Saves tab, the Console and the Share dialog each show a short dismissible banner the first time they open; Settings › General has **Show tips again**.

## Settings

Mortar's Settings holds only what is Mortar-wide, never one game's. It is a full page with a back button (Esc also returns), reached from the app menu and Ctrl+,, with a section list on the left and the section beside it; while it is open the title bar shows a "Settings" tab and no game tab.

- **General** (first, and the one Settings opens on): **Keep Mortar in the tray** (default off) with helper text about GNOME's AppIndicator extension, and **Show tips again**; under **Mods**, **Enable mods when installed** (on by default).
- **Appearance:** accent colour and background.
- **Launchers:** the first-run launcher rows, to add, remove or rescan a launcher's folders later.
- **Data:** a card with Mortar's data folder path and outlined **Open folder** and **Move…** buttons, like the game's SMAPI card; **Move…** copies the folder to a chosen empty location with enough free space (refused while the game is running, if the target is inside the current folder, or if it is not empty), verifies the copy, records `data-location` at the default path, removes the old copy and restarts; disk use for each profile's `mods/` folder (the store counted apart), the store, the cache, save backups, the trash and the total, measured in the background with a "Measuring…" state; **Clean up unused**, which lists what it would remove (unreferenced store items, expired cache files, leftover temp folders) with the size, then removes it on **Clean up**; under **Save backups**, the number field **Backups kept** (1 to 50, default 5) sets how many save backups Mortar keeps ([architecture.md](architecture.md#storage)); and **Export settings…** and **Import settings…**, which write or read a JSON file through the native dialogs, the import showing what would change before **Import** applies it ([architecture.md](architecture.md#settings-file)).
- **Nexus Mods:** the personal API key, or once signed in a green alert with the account name, a Premium or Free chip and **Sign out** inside it (saving it the first time asks whether Mortar should handle `nxm://` links); a capped-width **Preferred download server** when Mortar has seen servers; the switch **Handle Nexus "Mod Manager Download" links** with helper text under its label (who owns the links now and what turning it off does); and when a previous handler is recorded, **Send other games' links to …** (default on). A line reads "<n> requests left today · <n> this hour" from the last response, or in secondary text says the counts appear after Mortar talks to Nexus, or that Nexus is throttling.
- **Updates:** the installed Mortar version, the check's state and **Check now**, then **Download and install** and **Restart now** as the update progresses; a development build says it does not check; **Include beta releases** (default off) offers Mortar nightlies from GitHub. Under **Mods**, switches for **Check for mod updates when Mortar starts** (on by default; also the background desktop notification, [architecture.md](architecture.md#profile-operations)), **Include pre-release mod versions** (off by default), and **Check only enabled mods** (off by default). Mod and SMAPI updates stay where they are (each profile's mod list and the game settings SMAPI card, which includes **Tell me when a new SMAPI is out**, on by default), linked from here as Review mod updates and Open Stardew Valley settings once a game has been opened.
- **Shortcuts:** one row per chord: Ctrl+K command palette, Ctrl+F focus the search on the open tab (the Mods filter or the Console log search), Ctrl+P play the open profile, F5 check for mod updates, Ctrl+, open Settings, and Esc to close a dialog or clear a selection. Except Esc they stay silent while typing or with a dialog open.
- **About:** licences and credits, including SMAPI and the Stardew mod dataset (CC-BY-SA 4.0), a **Credits** list of each bundled project's name, linking its page, with its licence id, and **Save diagnostics…** at the foot.

**Game settings** are per game and live on their own page, opened from the game screen by the last icon button in the tab row (Lucide Settings2, labelled "Stardew Valley settings"), after **Share** (Share2). The page is titled "Stardew Valley settings", has a back button to the game (Esc also returns), and the title bar keeps the game tab highlighted.

- **Game folder:** found in Steam or chosen, with **Browse…** (checked before save) and **Use Steam's** while a chosen folder overrides Steam. **Reset game install** (text, error colour) opens **Reset game install?** explaining that the folder, SMAPI and mods placed there are deleted while Saves and Mortar profiles are kept; **Delete and restore** runs it ([architecture.md](architecture.md#finding-the-game)). Errors from the backend (including a running game) show under the folder.
- **SMAPI card:** installed version with Install, Reinstall or Update, the same step checks as the banner, and **Tell me when a new SMAPI is out**.
- **Stream overlay** (same solid panel):
  - Title, one-line description that changes apply on the next Play, and an **Enable** switch (off by default).
  - When enabled: **Show labels** for this visit (not stored); **Values** with "Load a save to see values" when `inGame` is false, or "Start the game to see values" when the bridge is unreachable.
  - Groups World, Player and Skills. Each overlay field (plus All values) is a row with a live preview (polled from `/state` every 2 s) and a copy icon for that OBS `file://` URL (`field` omitted on All values; `label=1` when Show labels is on). An empty value is an em dash.
  - OBS how-to under the list (Sources, +, Browser, width 400, height 80, Custom CSS) with **Copy CSS** using `#player` / `.player`, `#money` / `.money`, `#skill-mining` / `.skill.mining`.
  - **Connection:** **Port** (default 8123, 1024–65535) and the token as a password field with show/hide and copy, plus **Regenerate token**.
  - Stored as `overlayEnabled`, `overlayPort`, `overlayToken`. The command palette's Stream overlay action opens this page.
