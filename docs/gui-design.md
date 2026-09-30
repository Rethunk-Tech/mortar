# GUI design

How Mortar's screens are laid out and styled. Concrete (`LethalModding/Concrete`, archived) is one reference for the look, not a template to copy.

## Window

- Frameless and translucent (see [design.md](design.md#look)): 1280×720 by default, 768×432 minimum, as Concrete.
- The app draws its own title bar, 36px, marked `--wails-draggable: drag`: the logo in a darker square, the app name, then location tabs ("Game Select", or the current game) underlined in the primary colour, and minimise, maximise and close on the right. Double-click maximises. The window has a 1px light border and rounded corners, since GNOME draws no shadow for a frameless window.
- `user-select: none` everywhere except text fields and the console.
- Icons come from one set, Lucide (MIT), at 1.5–2px stroke; no hand-drawn or mixed icons.
- Every focusable control shows a visible focus ring when reached by keyboard: a 2px outline in the primary colour, 2px outside the control.
- Button and chip labels never wrap (`white-space: nowrap`); a long message gets its own full-width row, truncating with an ellipsis rather than squeezing the buttons beside it.

## Surfaces and colour

- No gradients anywhere, not as backgrounds, scrims over art, fallbacks or placeholder art (NOMAD, 2026-09-29): legibility over art comes from a solid translucent layer, and a missing image becomes a solid tone.
- Surfaces are translucent so the backdrop shows through: background `rgba(25,25,30,0.80)`, paper `rgba(50,50,60,0.80)`, dialog backdrops `rgba(0,0,0,0.75)`, the launch overlay `rgba(0,0,0,0.80)`.
- Text: primary `rgba(255,255,255,0.90)`, secondary `rgba(200,200,200,0.90)`. Links are secondary text with a dotted underline.
- Status colours from Concrete: info `#2B8BDA`, success `#0CDF64`, warning `#F3B416`, error `#C70A0A`. The primary colour is the user's choice (NOMAD, 2026-09-29), in Settings › Appearance: Sand `#D6B17A` (the default), Moss `#93B86A`, Copper `#D98C5F` or Sky `#79AEDC`, applied at once everywhere the primary appears. It is the same for every game; game colour appears only in hero art. Appearance also has a switch to turn the window's translucency off for a solid background.
- Scrollbars are themed: 0.5em, primary-dark track, primary thumb, primary-light on hover.

## Type

- Open Sans (300, 400, 500, 700) from `@fontsource/open-sans`, `htmlFontSize` 18 with MUI's responsive font sizes, as Concrete.
- Every string goes through Lingui.

## Game Select

Full-width banner rows stacked down the window, about 300px tall at 1080p, one per game. Rows share the window's height equally. Each shows the game's key art edge to edge under a light solid dim layer, with plain text straight on the art kept legible by a text shadow, no panel behind it; the row that opens on click has a primary-colour left edge; on the left the game name with "<loader> | <store>" under it ("SMAPI 4.5.2 | Steam"), and the profile count; on the right, square badges for its mod sources. Art is Steam's `library_hero.jpg` read at runtime, never bundled. In v1 Stardew Valley is the only live row; Lethal Company shows dimmed as coming later. Choosing a row opens its main screen; the title bar's game tab returns here.

## Main screen

A grid of a resizable sidebar and a detail pane, as Concrete's dashboard:

- **Sidebar** (150–300px, resized from an 8px handle on its right edge, 2px primary right border): a "Profiles" header that opens profile management, then one row per profile in the user's order, the open one selected. Each row shows a problem or update badge when it has any.
- **Bottom block** pinned under the sidebar: icon buttons for Support (a menu: Get help, which opens the Console tab's upload, and Report a Mortar bug), Settings and the download queue (with a count while it has items), then a full-width **Play** button in the primary colour.
- **Detail pane** for the open profile:
  - A hero, 300px tall (200px below 900px wide), bleeding to the pane's edges: the profile's cover image dimmed (`blur(1px) brightness(0.75) opacity(0.5)`), the profile name large and bold with a soft white glow, and small cards for updated, created, mod count and problems. The cover is an image the user picked, else the Nexus picture of the profile's most-endorsed mod (`endorsement_count` from the mod endpoint), else Steam's own hero art for the game, read at runtime from `<Steam>/appcache/librarycache/413150/library_hero.jpg` (Steam also keeps a `library_hero_blur.jpg`; never bundled with Mortar), else a solid tone; never a random image (NOMAD, 2026-09-29).
  - A **Saves** card in the hero (NOMAD, 2026-09-29) with how many saves fit this profile ("2 of 4"), opening the **Saves** tab: each save in the Saves folder with its fit, "all mods present" or "has used N mods this profile lacks" with their names, from the save scan in design.md; a mod can be dismissed for a save. Players pick their save inside the game, so this is where the check shows, before Play.
  - Tabs: **Mods** (default), **Saves**, **Notes**, **Console**, and a **Share** button at the end of the tab row.
- **Launching** covers the window with the launch overlay: a spinner, "Launching Stardew Valley", and SMAPI's first log lines as they arrive (started, mods loading, mods it will skip), with Open console and Hide, until the game is up or the launch fails. While the game runs, the Play button becomes a Running status with the elapsed time and a **Stop game** button.
- **Minimum size (768×432):** the sidebar collapses to a rail of profile initials with Play at its foot, the hero gives way to a one-line header (name and counts), search and actions fold into icon buttons, and the mod grid drops to two columns.

## Mods tab

- Two views, switched from the toolbar and remembered per user (NOMAD, 2026-09-29): **Grid** (default), and **List**, a table (switch, picture, name, author, source, status) with an inspector for the selected mod on the right. The hero shrinks to a 96px band in List view to fit more rows.
- Grid: a responsive grid of mod cards (`minmax(330px, 1fr)`; 250px below 900px wide, 240px below 700px).
- A card: the mod's icon (its Nexus `picture_url`, fetched once when the mod is installed and cached on disk; a letter tile until it loads or when there is none, NOMAD 2026-09-29), its name, author and version from its manifest, an enable switch (the dot-folder toggle), and badges for an available update or a problem. One card per mod; its menu (open the Nexus page, update, roll back when there is a previous version, remove) acts on the whole archive entry the mod came in, and when that entry holds other mods the menu names them first.
- Above the grid: a search field that filters by name, the problem summary with one-click fixes ([design.md](design.md#mod-data)), and buttons to add mods (open Nexus, or pick an archive). Dropping an archive anywhere on the window installs it into the open profile.
- Empty (a new profile): "No mods yet", a line on how to add them, **Browse Nexus** and **Add archive**, a link to import a shared profile, and a note that archives can be dropped anywhere. The hero uses Steam's art.

## Mod detail

Clicking a card opens a panel on the right (List view's inspector links to the same content): picture, name, author and Nexus page, the enable switch, an update banner, **Needs** (each dependency and whether it is installed), **In the same download** (mods that update and roll back together), **Versions** (the one in use and the one kept for rollback, with Roll back), **Settings** (`config.json`, whether it was changed, Open and Reset), **Needed by**, and Show files and Remove from profile.

## Saves tab

Every save in the Saves folder with its fit for this profile: a coloured edge and a chip ("All mods present", "Has used N mods it lacks"), the mods it has used as removable chips (dismissing one for that save), and Add to this profile for mods Mortar can install. A line explains that the save is picked inside the game.

## Notes tab

A plain text area for the profile's notes, saved automatically ("Saved · 2 min ago"); notes travel in the `.mortar` file but not in links.

## In-between moments

- **Dropping an archive:** while an archive is dragged over the window, a dashed drop zone says where it goes ("Drop to install into Cookie farm"), with the file name and the supported formats.
- **A Nexus link while minimised:** a desktop notification says what is downloading and into which profile, with Show. An `nxm://` link Mortar did not ask for gets a notification with the open profile, Other profile… and Ignore.
- **Two copies of one mod:** a dialog shows both (source, version, what depends on it), preselects the newer or Nexus-sourced one, and switches the other off rather than deleting it; Decide later leaves both as a problem.

## Console tab

- SMAPI's log as it is written, monospace, in columns: a level bar, time, level, mod and message; warnings and errors get a tinted row.
- Filters: a search box, level toggles each with its line count (Trace off by default), a mod picker whose choices show as removable chips, "Showing X of Y lines" with Clear filters, and Jump to first error. Toggles for timestamps and follow-tail (on by default).
- **Copy**, and **Get help**: shows the log with its local paths and asks before uploading it to smapi.io/log, then copies the link.
- Before the first launch: a line saying the console fills when the game runs.

## Profile management

A page, as Concrete's: a header with a back button, **New**, **Import** and **Recently deleted** (profiles in the 30-day trash, each with Restore) buttons, then every profile as a sortable row: drag handle, name (with a Hidden chip when hidden) and its summary, **Share**, and a ⋯ menu holding Rename, Duplicate, Hide from sidebar (or Show) and Delete. Reordering uses dnd-kit's sortable list with a drop indicator. Delete asks first and says the profile stays restorable for 30 days. **Share** opens a dialog with two tabs, **Link** and **.mortar file with settings**. The Link tab has the `https://mortar.rethunk.tech/stardew/p#...` link with Copy link, a meter of its length against Discord's 2,000 characters, Copy as a message, the included mods by source, and what is left out (local archives, switched-off mods). Beside it, a preview of the page the recipient sees. Over about 240 mods it suggests the file.

## Import

A wide "Import profile from…" dialog over the dimmed game screen, with tabs **Link**, **.mortar file** and **Archives**. Nothing downloads before the user confirms.

- A dense five-column grid of mod tiles: icon, name, author, and on the right the mod's import state (installed, download, dependency, check later, unavailable) in place of a version number.
- A status bar: "Ready to import", the profile name, the mod count, the counts per state, and the approximate download size (the sum of each file's `size_kb` from Nexus).
- Problems found before download, one compact item each with its own action: a mod removed from Nexus (with its page), a mod broken for this game version (from SMAPI's update API, with Leave out), missing dependencies (from the mod dataset); and for a free account, an item saying each download takes one click on Nexus.
- **Reset** and **New profile from link**, which creates the profile and fills the download queue; closing leaves nothing behind.

## Download queue

A side sheet. The header gives totals (done, in progress, failed, left, size) over a progress bar split by state, with **Pause all**. Sections, top down: **Needs your click** (free accounts: the head item with Open download page; the next page opens when it finishes), **Failed** (with Retry failed; each row says why and has Retry and Skip on its right edge), **In progress** (size, speed and Cancel per row), **Up next** and **Done** (both collapsed to one line). Every row's action sits on its right edge. Premium accounts download without clicks. Closing the sheet keeps the queue running.

## First run

Only when no game is set up, never on later launches:

1. Find Stardew Valley in Steam and show what was found, with **Browse** for another folder; when nothing is found, say so, with Browse and a retry.
2. Install SMAPI if it is missing (one click, unattended). On Windows with Steam, show the launch-options line to paste into Stardew's Steam properties, with a copy button.
   Installing SMAPI shows its progress as steps (downloaded, files added, launcher replaced, bundled mods added).
3. Create the first profile: two cards, **Start empty** (with a name field) and **From a shared link** (with a link field, which asks for the Nexus sign-in first when the import needs downloads).

## Confirmations

Short toasts, bottom right, stacked, each with a coloured edge by kind and a dismiss button, and an undo-style action where one exists: "Link copied", "SpaceCore installed" (Undo), "… rolled back" (Redo update), "Couldn't reach Nexus" (Retry now). A toast about a mod shows its picture.

## Settings

A dialog with sections **Game** (folder, SMAPI version with Reinstall, Mortar's data folder and size), **Nexus Mods**, **Updates** (mod, SMAPI and Mortar update checks; backups kept), **Appearance** (accent colour, translucency) and **About**. In short: game folders, the Nexus personal API key (saving it the first time asks whether Mortar should handle `nxm://` links), whether Mortar handles `nxm://` links (asks before taking them from another manager, and gives them back when turned off), update checks, and About (licences and credits, including SMAPI and the Stardew mod dataset, CC-BY-SA 4.0).
