# GUI design

How Mortar's screens are laid out and styled. Concrete (`LethalModding/Concrete`, archived) is the visual reference; its unfinished parts are not.

## Window

- Frameless and translucent (see [design.md](design.md#look)): 1280×720 by default, 768×432 minimum, as Concrete.
- The app draws its own title bar, 36px, marked `--wails-draggable: drag`: app name, the current game, and minimise, maximise and close on the right. Double-click maximises. The window has a 1px light border and rounded corners, since GNOME draws no shadow for a frameless window.
- `user-select: none` everywhere except text fields and the console.

## Surfaces and colour

- Surfaces are translucent so the backdrop shows through: background `rgba(25,25,30,0.80)`, paper `rgba(50,50,60,0.80)`, dialog backdrops `rgba(0,0,0,0.75)`, the launch overlay `rgba(0,0,0,0.80)`.
- Text: primary `rgba(255,255,255,0.90)`, secondary `rgba(200,200,200,0.90)`. Links are secondary text with a dotted underline.
- Status colours from Concrete: info `#2B8BDA`, success `#0CDF64`, warning `#F3B416`, error `#C70A0A`. The primary colour changes for Stardew (**Open**, see below).
- Scrollbars are themed: 0.5em, primary-dark track, primary thumb, primary-light on hover.

## Type

- Open Sans (300, 400, 500, 700) from `@fontsource/open-sans`, `htmlFontSize` 18 with MUI's responsive font sizes, as Concrete.
- Every string goes through Lingui.

## Main screen

A grid of a resizable sidebar and a detail pane, as Concrete's dashboard:

- **Sidebar** (150–300px, resized from an 8px handle on its right edge, 2px primary right border): a "Profiles" header that opens profile management, then one row per profile in the user's order, the open one selected. Each row shows a problem or update badge when it has any.
- **Bottom block** pinned under the sidebar: icon buttons for Support, Settings and the download queue (with a count while it has items), then a full-width **Play** button in the primary colour.
- **Detail pane** for the open profile:
  - A hero, 300px tall (200px below 900px wide), bleeding to the pane's edges: the profile's cover image dimmed (`blur(1px) brightness(0.75) opacity(0.5)`), the profile name large and bold with a soft white glow, and small cards for updated, created, mod count and problems. A profile without a cover shows a gradient from its colour (**Open**), never a random image.
  - Tabs: **Mods** (default), **Notes**, **Console**.
- **Launching** covers the window with the launch overlay, a spinner and "Launching Stardew Valley" until SMAPI's log shows the game started or the launch fails.

## Mods tab

- A responsive grid of mod cards (`minmax(330px, 1fr)`; 250px below 900px wide, 240px below 700px).
- A card: the mod's icon, its name, author and version, an enable switch (the dot-folder toggle), and badges for an available update or a problem. Its menu: open the Nexus page, update, roll back, remove.
- Above the grid: a search field that filters by name, the problem summary with one-click fixes ([design.md](design.md#using-the-app)), and buttons to add mods (open Nexus, or pick an archive). Dropping an archive anywhere on the window installs it into the open profile.
- Empty: a short line and the same two add buttons.

## Console tab

- SMAPI's log as it is written, monospace, colour by level (trace dimmed, warn in warning colour, error in error colour), follow-tail on by default.
- Filters by level and mod, copy, and **Get help**: shows the log with its local paths and asks before uploading it to smapi.io/log, then copies the link.
- Before the first launch: a line saying the console fills when the game runs.

## Profile management

A page, as Concrete's: a header with a back button, **New** and **Import** buttons, then every profile as a sortable row (drag handle, name, rename, duplicate, share, show or hide, delete). Reordering uses dnd-kit's sortable list with a drop indicator. Delete asks first. **Share** copies the `https://mortar.rethunk.tech/p#...` link and offers the `.mortar` file when the profile has configs or is over about 240 mods.

## Import

Opening a link, pasting one, or opening a `.mortar` file shows one dialog before anything is downloaded:

- The profile's name and its mods, grouped: already installed, to download, dependencies added, unavailable (with the Nexus page).
- Problems found from the mod dataset (missing dependencies, broken for this game version).
- **Import** creates the profile and fills the download queue; **Cancel** leaves nothing behind.

## Download queue

A side sheet listing each item: waiting for your click (free accounts), downloading with progress, installing, done, or failed with a retry. For free accounts the head item has **Open download page**, and the next page opens when an item finishes. Premium accounts download without clicks. Closing the sheet keeps the queue running.

## First run

Only when no game is set up, never on later launches:

1. Find Stardew Valley (Steam, then GOG) and show what was found, with **Browse** for another folder.
2. Install SMAPI if it is missing (one click, unattended).
3. Create the first profile: empty, or from a pasted share link.

## Settings

A dialog: game folders, the Nexus account (the personal key field while testing), whether Mortar handles `nxm://` links (asks before taking them from another manager), and update checks.

## Open

- The primary colour for Stardew (Concrete's is Lethal Company amber `#e8982f`).
- Where profile colours and covers come from.
- Mod icons: the Nexus mod endpoint (`/v1/games/stardewvalley/mods/<id>.json`) returns `picture_url` (checked 2026-09-29), one call per mod, cached; the mod dataset has no picture. The alternative is a letter tile.
