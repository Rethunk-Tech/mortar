# GUI design

How Mortar's screens are laid out and styled. Concrete (`LethalModding/Concrete`, archived) is one reference for the look, not a template to copy.

## Window

- Frameless and translucent (see [design.md](design.md#look)): 1280×720 by default, 768×432 minimum, as Concrete.
- The app draws its own title bar, 36px, marked `--wails-draggable: drag`: the logo in a darker square, the app name, then location tabs ("Game Select", or the current game) underlined in the primary colour, and minimise, maximise and close on the right. Double-click maximises. The window has a 1px light border and rounded corners, since GNOME draws no shadow for a frameless window.
- `user-select: none` everywhere except text fields and the console.

## Surfaces and colour

- No gradients anywhere, not as backgrounds, scrims over art, fallbacks or placeholder art (NOMAD, 2026-09-29): legibility over art comes from a solid translucent layer, and a missing image becomes a solid tone.
- Surfaces are translucent so the backdrop shows through: background `rgba(25,25,30,0.80)`, paper `rgba(50,50,60,0.80)`, dialog backdrops `rgba(0,0,0,0.75)`, the launch overlay `rgba(0,0,0,0.80)`.
- Text: primary `rgba(255,255,255,0.90)`, secondary `rgba(200,200,200,0.90)`. Links are secondary text with a dotted underline.
- Status colours from Concrete: info `#2B8BDA`, success `#0CDF64`, warning `#F3B416`, error `#C70A0A`. The primary colour is Mortar's own, the same for every game (NOMAD, 2026-09-29), a warm sand tone replacing Lethal Company's amber `#e8982f`; the exact value is picked with NOMAD during milestone 1. Game colour appears only in hero art.
- Scrollbars are themed: 0.5em, primary-dark track, primary thumb, primary-light on hover.

## Type

- Open Sans (300, 400, 500, 700) from `@fontsource/open-sans`, `htmlFontSize` 18 with MUI's responsive font sizes, as Concrete.
- Every string goes through Lingui.

## Game Select

Full-width banner rows stacked down the window, about 300px tall at 1080p, one per game. Each row shows the game's key art edge to edge under a solid dim layer (`rgba(0,0,0,0.4)`), with the game's logo image (Steam's `logo.png`) or its name; on the left the game name with "<loader> | <store>" under it ("SMAPI 4.5.2 | Steam"), and the profile count; on the right, square badges for its mod sources. Art is Steam's `library_hero.jpg` read at runtime, never bundled. In v1 Stardew Valley is the only live row; Lethal Company shows dimmed as coming later. Choosing a row opens its main screen; the title bar's game tab returns here.

## Main screen

A grid of a resizable sidebar and a detail pane, as Concrete's dashboard:

- **Sidebar** (150–300px, resized from an 8px handle on its right edge, 2px primary right border): a "Profiles" header that opens profile management, then one row per profile in the user's order, the open one selected. Each row shows a problem or update badge when it has any.
- **Bottom block** pinned under the sidebar: icon buttons for Support (a menu: Get help, which opens the Console tab's upload, and Report a Mortar bug), Settings and the download queue (with a count while it has items), then a full-width **Play** button in the primary colour.
- **Detail pane** for the open profile:
  - A hero, 300px tall (200px below 900px wide), bleeding to the pane's edges: the profile's cover image dimmed (`blur(1px) brightness(0.75) opacity(0.5)`), the profile name large and bold with a soft white glow, and small cards for updated, created, mod count and problems. The cover is an image the user picked, else the Nexus picture of the profile's most-endorsed mod (`endorsement_count` from the mod endpoint), else Steam's own hero art for the game, read at runtime from `<Steam>/appcache/librarycache/413150/library_hero.jpg` (Steam also keeps a `library_hero_blur.jpg`; never bundled with Mortar), else a solid tone; never a random image (NOMAD, 2026-09-29).
  - A **Saves** card in the hero (NOMAD, 2026-09-29) with how many saves fit this profile ("2 of 4"), opening the **Saves** tab: each save in the Saves folder with its fit, "all mods present" or "has used N mods this profile lacks" with their names, from the save scan in design.md; a mod can be dismissed for a save. Players pick their save inside the game, so this is where the check shows, before Play.
  - Tabs: **Mods** (default), **Saves**, **Notes**, **Console**, and a **Share** button at the end of the tab row.
- **Launching** covers the window with the launch overlay, a spinner and "Launching Stardew Valley" until SMAPI's log shows the game started or the launch fails.

## Mods tab

- A responsive grid of mod cards (`minmax(330px, 1fr)`; 250px below 900px wide, 240px below 700px).
- A card: the mod's icon (its Nexus `picture_url`, fetched once when the mod is installed and cached on disk; a letter tile until it loads or when there is none, NOMAD 2026-09-29), its name, author and version from its manifest, an enable switch (the dot-folder toggle), and badges for an available update or a problem. One card per mod; its menu (open the Nexus page, update, roll back when there is a previous version, remove) acts on the whole archive entry the mod came in, and when that entry holds other mods the menu names them first.
- Above the grid: a search field that filters by name, the problem summary with one-click fixes ([design.md](design.md#mod-data)), and buttons to add mods (open Nexus, or pick an archive). Dropping an archive anywhere on the window installs it into the open profile.
- Empty: a short line and the same two add buttons.

## Console tab

- SMAPI's log as it is written, monospace, colour by level (trace dimmed, warn in warning colour, error in error colour), follow-tail on by default.
- Filters by level and mod, copy, and **Get help**: shows the log with its local paths and asks before uploading it to smapi.io/log, then copies the link.
- Before the first launch: a line saying the console fills when the game runs.

## Profile management

A page, as Concrete's: a header with a back button, **New**, **Import** and **Recently deleted** (profiles in the 30-day trash, each with Restore) buttons, then every profile as a sortable row (drag handle, name, rename, duplicate, share, show or hide, delete). Reordering uses dnd-kit's sortable list with a drop indicator. Delete asks first and says the profile stays restorable for 30 days. **Share** copies the `https://mortar.rethunk.tech/stardew/p#...` link and always offers the `.mortar` file "with settings", and suggests it over the link above about 240 mods.

## Import

A wide "Import profile from…" dialog over the dimmed game screen, with tabs **Link**, **.mortar file** and **Archives**. Nothing downloads before the user confirms.

- A dense five-column grid of mod tiles: icon, name, author, and on the right the mod's import state (installed, download, dependency, check later, unavailable) in place of a version number.
- A status bar: "Ready to import", the profile name, the mod count, the counts per state, and the approximate download size (the sum of each file's `size_kb` from Nexus).
- Problems found before download: missing dependencies (from the mod dataset) and mods broken for this game version (from SMAPI's update API); for a free account, a line saying each download takes one click on Nexus.
- **Reset** and **New profile from link**, which creates the profile and fills the download queue; closing leaves nothing behind.

## Download queue

A side sheet listing each item: waiting for your click (free accounts), downloading with progress, installing, done, or failed with a retry. For free accounts the head item has **Open download page**, and the next page opens when an item finishes. Premium accounts download without clicks. Closing the sheet keeps the queue running.

## First run

Only when no game is set up, never on later launches:

1. Find Stardew Valley in Steam and show what was found, with **Browse** for another folder; when nothing is found, say so, with Browse and a retry.
2. Install SMAPI if it is missing (one click, unattended). On Windows with Steam, show the launch-options line to paste into Stardew's Steam properties, with a copy button.
3. Create the first profile: empty, or from a pasted share link, which asks for the Nexus sign-in first when the import needs downloads.

## Settings

A dialog: game folders, the Nexus personal API key (saving it the first time asks whether Mortar should handle `nxm://` links), whether Mortar handles `nxm://` links (asks before taking them from another manager, and gives them back when turned off), update checks, and About (licences and credits, including SMAPI and the Stardew mod dataset, CC-BY-SA 4.0).
