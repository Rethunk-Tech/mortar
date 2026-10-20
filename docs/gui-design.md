# GUI design

The design system and screen rules for Mortar's UI. Read it before any UI change. The code in `frontend/src` is the record of every label, tooltip, menu entry and state: this doc gives the rules that apply everywhere, then per screen its purpose, structure and the rules a change could break. Concrete (`LethalModding/Concrete`, archived) is one reference for the look, not a template to copy.

## Window

- **Logo:** the running-bond mark, seven rounded bricks in three courses, filled in the primary colour (`frontend/src/brand/`); the title bar (18px), the app icon on a dark rounded square, and desktop notifications use it.
- Frameless on a solid window ([architecture.md](architecture.md#stack)): 1280×720 by default, 768×432 minimum. The frame is a 1px light border with rounded corners, since GNOME draws no shadow for a frameless window.
- `user-select: none` everywhere except text fields and the console. The WebKit default context menu is suppressed except on an editable field or with a text selection.
- Icons come from one set, Lucide (MIT), at 1.5-2px stroke; no hand-drawn or mixed icons.

### Title bar

48px (40px compact), draggable with its buttons not draggable; double-click maximises. Code: `shell/TitleBar.tsx`, `AppMenu.tsx`, `titleMenus.ts`. Left to right: the **Mortar menu** (logo and name; app-level actions, update check, help, bug report, source offer), the **game switcher** (the open game; its menu follows Game select's order), the **profile switcher** after a "/" while a game is open (its menu lists the game's profiles and the management entries; its context menu is the shared profile menu, [Profile management](#profile-management)), then on the right the **downloads pill**, the **notifications bell** and the window buttons. With nothing under way the title bar stays quiet: the pill shows only while the queue has items under way.

- **Notifications:** a bell with an unread count opening a solid dark panel (Ctrl+Shift+N). **New** holds this session's unread toasts; **Earlier** holds read toasts, then the open profile's change history. A toast action such as Undo is offered only while it still applies.
- **Startup repair:** a one-time "Mortar tidied up N things" notification after a launch that repaired something.
- **Mortar update ready:** a full-width info banner under the title bar once a background self-update is staged (it applies when Mortar closes).
- **What's new** (`updates/WhatsNewDialog.tsx`): once per version after an upgrade, a dialog of that release's GitHub notes; offline skips until a later start ([architecture.md](architecture.md#release)).
- **Tray** (`main.go`, `tray_other.go`, `tray_windows.go`; **Keep Mortar in the tray**, Settings › General): closing the window keeps Mortar running; off means close quits. GNOME needs the AppIndicator extension (the helper text says so). A desktop notification when a Mortar-started run ends raises the window on that profile's Console.

### Menus and the command palette

- Every menu uses the one dark paper, a 1px light border and 8px corners (`MuiMenu` in `frontend/src/theme/theme.ts`); an item with an obvious icon carries a Lucide one. Dividers are the shared `MenuRule`, never hand-rolled. Tooltips on menu rows and rails use `OneTip` (one at a time, closed by a press). One set of profile items (`game/ProfileMenuItems.tsx`) serves the switcher and the Profiles row menu.
- **Command palette** (`commandPalette/`, Ctrl+K): fuzzy search over actions, destinations and the shortcuts that run from anywhere. A shortcut that does what an action does shows its keys on that action, never as a second row; shortcuts for the focused list are left out; an action needing an open profile is greyed with "Open a profile first." SMAPI's bundled mods and the Mortar bridge are omitted, as on the Mods tab.

### Input, focus and accessibility

- Every focusable control shows a visible focus ring on keyboard focus: a 2px outline in the primary colour, 2px outside the control; a whole-row or whole-tile cover button draws it 2px inside, since its row clips what spills over.
- Everything works at 1280×800 (the Steam Deck) with the arrow keys, Tab and Enter that Steam Input maps a controller to, and no screen scrolls sideways. Grids move focus by the nearest item in the next row (`shell/arrowFocus.ts`); the mod list drops optional columns until the name keeps 160px (`fitListColumns`). `e2e/deck.pw.ts` checks it.
- **Gamepad** (`shell/gamepad.ts`, `gamepadInput.ts`, `e2e/gamepad.pw.ts`): the d-pad and stick send arrow keys to the focused control first, so lists, tabs, menus and grids keep their own movement, and otherwise move focus to the nearest control, staying inside an open dialog, menu or drawer. Focus the pad moves always shows the ring (`html[data-input="gamepad"]`), since browsers ring script focus only after keyboard use.
- **Touch or a gamepad as the main input** (`theme/roomy`): every control is at least 44×44px and compact density gives way; a resize handle keeps its thin line with an invisible 44px strip. A link in a line of text keeps its size (WCAG 2.5.8's inline exception). `e2e/deck.pw.ts` checks target sizes and runs axe-core.
- **Play prompt:** in play mode (`--play … --steam-session` from a Steam shortcut, [architecture](architecture.md#command-line)) the only thing drawn before a fix is the launch dialog that blocked Play, alone on the window background in a 560×480 window with 44px targets (`frontend/e2e/play.pw.ts`).

### Buttons and dialogs

- **Dialog actions:** Cancel or Close is a text button, leftmost; at most one contained primary sits rightmost; a destructive primary is contained `color="error"`; other actions beside them stay text. Saving a name that exists asks **Replace <name>?**; deleting a preset says **Delete preset**.
- Button and chip labels never wrap; a long message gets its own full-width row and truncates with an ellipsis rather than squeezing the buttons beside it.
- Dialogs use solid paper with no open or close transition ([Surfaces and colour](#surfaces-and-colour)). Shared pieces: `shell/ConfirmDialog.tsx`, `PromptDialog.tsx`, `useDiscardGuard.tsx`.

## Density

One spacing scale per **Density** level (Settings › Appearance › Display), set as CSS variables on `:root` (`theme/density.ts`, applied in `theme/theme.ts`). Layout code reads them through `space` and never hand-sets pixels for these roles, so a level change moves every page together. Touch or a gamepad still raises every target to 44px.

| Token | Role | Comfortable | Compact |
| ----- | ---- | ----------- | ------- |
| `--m-gutter` | a page's left and right edge, a controls row's inset | 16px | 12px |
| `--m-gap` | between controls in a row; between a row and what follows | 8px | 6px |
| `--m-pad` | inside a card or panel | 16px | 12px |
| `--m-row` | a table or list row | 36px | 30px |
| `--m-menu-y` | above and below a menu item's text, and a menu's own top and bottom | 6px | 4px |
| `--m-control` | buttons, dropdowns, fields and the page-actions slot that share a row | 36px | 32px |

Fixed on purpose: spacing of 6px and under (icon-to-label gaps, chip and badge insets, a row's inner rhythm), vertical margins between a heading and its text, indents, zero padding resets, chrome that is not a layout role (window buttons, title bar), and widths such as grid column minimums and panel widths. The tab header is `--m-control` plus twice `--m-gap` tall (52px comfortable, 44px compact).

## Surfaces and colour

- **No gradients** anywhere: not as backgrounds, scrims over art, fallbacks or placeholder art. Legibility over art comes from a solid layer with alpha; a missing image becomes a solid tone. The one exception is an image's edge fading to transparent through an alpha mask.
- **The window is solid.** Dark uses an opaque base, light an opaque `rgb(236,236,241)`. Behind the surfaces sits the wallpaper under a static tint (`--mortar-tint`). Hard-coded chrome colours live as `--mortar-*` tokens from `frontend/src/theme/palette.ts`, so both palettes share one set of surfaces.
  - Dark surfaces keep their alpha and composite over the tint; light surfaces are solid, with no translucent fills that turn muddy over the wallpaper.
  - Dialog paper (`MuiDialog` in `theme/theme.ts`) and the download queue sheet are opaque so the text they cover never shows through. Dialog and drawer backdrops are a static overlay (`--mortar-overlay-30`); dialogs and their backdrops open and close without a transition; the launch overlay uses `--mortar-overlay-80`. The drop overlay and first-run tour spotlight dim with `--mortar-scrim`.
  - Every popover (menus, the Saves backups list, pickers) is the solid menu surface with a hairline border, never the translucent paper colour.
- **Theme** (Settings › Appearance): Dark, Light or Follow system; app chrome only, game and profile settings are unchanged. Dark is the lead theme: screenshots, README and store listings show it.
  - Both themes keep the chosen accent as a fill, with white or near-black text on it, whichever reaches 4.5:1.
  - Text and icons tinted with the accent use `--mortar-accent-ink`: the accent on dark, the normal ink on light, where pastel accents are unreadable as text.
- **Background** (Settings › Appearance): **Image**, **Desktop** or **Solid**, all opaque. Image and Desktop draw the wallpaper full-window behind everything (title bar included), `object-fit: cover`, under the tint; it never fades or filters. The window loads it from `/background?mode=` with `Cache-Control: no-store`; under Reduce motion an animated image shows its first frame.
  - **Desktop** reads the user's own wallpaper at runtime: GNOME `org.gnome.desktop.background` `picture-uri-dark` under the dark colour scheme, else `picture-uri`; Windows the `HKCU\Control Panel\Desktop` `Wallpaper` value. It falls back to Image's choice when unreadable.
  - **Default wallpaper:** Fedora 44's `f44-01-night` (Fedora Design Team, CC-BY-SA-4.0, credited in About): a bundled 2560px JPEG, shrunk to the window's pixel width (the system copy `/usr/share/backgrounds/f44/default/f44-01-night.jxl` is 4096px square and cannot be shrunk here, so it is served only to a request that names no width).
- **Text:** primary `rgba(255,255,255,0.90)`, secondary `rgba(225,225,230,0.95)`. Links are secondary text with a dotted underline.
- **Status colours** from Concrete: info `#2B8BDA`, success `#0CDF64`, warning `#F3B416`, error `#C70A0A` on dark; light darkens them until they meet 4.5:1 on paper.
- **Primary colour** is the user's choice from the palette in `frontend/src/theme/accents.ts`. It applies at once everywhere the primary appears, is never darkened, and is the same for every game; game colour appears only in hero art.
- **Scrollbars** are themed in the primary colour at low alpha (`theme/theme.ts`).
- **One-line status alerts** (a callout above a list, a read error with Retry, a settings status line) are the theme's compact Alert (`<Alert variant="compact" severity=…>`): 38px tall, the severity's callout fill and line. Size and colour come from the variant, never from a component's own `sx`. A window-wide banner, an alert with a title, and notes inside a dialog keep the standard Alert. Callout actions order text secondaries left and one contained primary right.
- **Fix and action buttons are never blue**; severity colours (warning for real conflicts, error for destructive) carry weight, and harmless rows use outlined buttons.

## Type

Open Sans (300 to 700) from `@fontsource/open-sans`, `htmlFontSize` 18 with MUI's responsive font sizes, as Concrete. Every string goes through Lingui.

## Copy

How every string reads. One wording per message: `bun run --cwd frontend i18n:dupes` fails when two msgids differ only in case, punctuation or placeholder names.

- **Enable / Disable** a mod, plugin or option; never "Switch on/off", "Turn on/off" or "Activate".
- **Remove** takes something out of a profile, group or list and leaves the thing itself; **Delete** destroys it on disk (a profile, saved group, template, preset, backup, downloaded file). The button, its confirmation and its toast use the same verb.
- Sentence case everywhere: "Open settings", not "Open Settings". A fragment that must be lowercase is lowercased in code, so one msgid serves both.
- Labels, buttons, headings and toast titles take no full stop; a sentence or line of body text does. Change lines and disabled-source reasons start with a capital ("Added <mod>").
- One placeholder name per concept: `{name}`, `{count}`, `{profile}`, `{game}`, `{version}`, `{path}`, `{when}`, `{min}`, `{max}`. No positional `{0}` and no per-site names (`{aName}`); plurals use `{count}`.
- A trailing "…" only on a control that opens further input (a dialog, a picker); the dialog's own title has none. A control that acts at once has none, and "…" is never a loading marker inside a sentence.
- No em dashes in text; use a colon, a full stop or a comma (an empty table cell may show one). Straight quotes and apostrophes only; a name goes in a placeholder without quotes.
- Errors read "Could not <verb> <thing>", never "Couldn't", "Failed to" or "Unable to"; no "Please".
- Shortest wording that keeps every fact: what it does, and when it matters.
- **Times:** a moment within the last week reads relative ("3 minutes ago", "yesterday"); older ones show the date, with the time where the list is a timeline (runs, backups, profile history, notifications). This covers every column, hero, card and status line that shows a time.
- **Names that cannot be resolved** read **Unknown mod** with the UniqueID in the tooltip, shown only when no id is known either.

## Game Select

Mortar opens on the last game used; Game Select shows at start only until a game has been opened, and otherwise from the title bar. Code: `games/GameSelect.tsx`, `LetterRail.tsx`, `order.ts`.

One list of full-width game rows (168px tall), however many games the catalog holds; it never wraps into columns. Rows touch (no gap, no rounded corner) and the page scrolls. The three games used most recently come first under **Recently played**, the rest by name under **All games** (names sort as typed, letters and digits only, so R.E.P.O. files under R).

- **Art:** the game's key art edge to edge under a light solid dim layer, with plain text straight on it kept legible by a text shadow and no panel behind. Art is Steam's `library_hero.jpg` read at runtime, never bundled.
- **Hover:** the row under the pointer or holding keyboard focus grows 72px taller over 450ms to show more art; the art never shifts sideways or zooms, rows never dim on hover, and no tile carries an edge stripe. A tile that cannot open (a game coming later) is desaturated under a darker dim.
- **Profile cards** (set-up games): compact, in the left column, capped at six then a **+N more** card; clicking one opens that profile and does not fire the row's open handler. No Play on a card.
- **Play** sits right under the source badges and starts the game's last-played profile through the same Start path as the sidebar button (SMAPI install and Steam checks included); a game not set up shows **Set up** in its place.
- **Letter rail:** from eight games under **All games**, a rail of initials replaces the scrollbar; the letter of the first row in view shows in `--mortar-accent-ink`. **Type to find:** typing outside any field, menu or dialog jumps to the first game whose name starts with (else contains) the letters, ignoring case, spaces and punctuation, and focuses the row so Enter opens it.

## Main screen

A left sidebar and a detail pane. Code: `game/MainScreen.tsx`, `Sidebar.tsx`, `ProfileWorkspace.tsx`.

- **Sidebar:** 220px expanded, 76px as a rail (icon-only buttons with name and tooltip; always the rail below 960px wide), a `--mortar-nav` surface with a 1px hairline. It is one vertical tab list ("Profile sections"): one tab stop on the open section, arrow keys and a gamepad's shoulder buttons move between sections, each tab points at the workspace `tabpanel` by `aria-controls`, each group is a labelled `role="group"`.
  - Sections: **Home**, then **Library** (Mods with an update badge, Saves, Config), **Get mods** (Browse), **Health** (Problems with a warning badge, Load order, Performance), **Logs** (Console). A section the loader does not support is absent, and a group left empty loses its header. Ctrl+1 to Ctrl+8 select sections; Collapse and Play pin to the foot. Settings is not in the sidebar (it stays in the Mortar menu and on Ctrl+,).
- **Play** (`launch/PlayControl.tsx`, `playMenu.ts`): a full-width 64px primary button split with a caret. Its menu lists launch presets, **Set default preset…** and **Play without mods** (starts the game with no profile and does not lock mods).
- **Detail pane:** Home alone shows the hero; every other section opens with the [Tab header](#tab-header).
- **Compact layout** (below 960px wide): the icon rail, the hero becomes a one-line header, the mods search folds into a filter icon, other actions fold into icon buttons, the mod grid drops to two columns.

### Home

The hero, then <a id="home-page"></a>the **Home page** (`game/Home.tsx`, `Hero.tsx`, `HomePanel.tsx`): an action row and a grid of cards (`HomePanel.tsx`; one column when compact).

- **Hero:** bleeds to the pane's edges; the name carries no actions beside it (they are in the action row and the profile menus, [Profile management](#profile-management)). The cover (`game/cover.ts`) is the image under a light solid dim layer whose alpha fades to transparent over the bottom 20% by an alpha mask; the hero has no background of its own. Under the name, `HeroGlance.tsx` lists the profile's facts on one wrapping line: mods, updates, problems, last played, created and updated (the compact strip keeps its own short counts).
- **Cover source, in order:** an image the user picked, the Nexus picture of the profile's most-endorsed mod, Steam's hero art for the game (read at runtime, never bundled), a solid tone. Never a random image.
- **Cards:** a crash card after a crashed run (cause, evidence, **Disable it and play**, **Find the cause…**, a crash bisect, [architecture.md](architecture.md#launch)), notes (saved as typed, capped), changes since last run, saves. No card sits alone at one column.

### Launching

Code: `frontend/src/launch/`.

- **Launch overlay** (`LaunchLayer.tsx`): an MUI Modal over the window (focus trapped, Esc hides it, the window controls stay clickable through a cut-out) until the game is up or the launch fails.
- While the game runs, Play becomes a Running status with **Stop game** and the profile's mod controls lock ("Stop the game to change mods."): switching, removing, installing, dropping archives.
- **Pre-Play dialogs** are each omitted when nothing applies and follow the dialog action rule ([Buttons and dialogs](#buttons-and-dialogs)): `PrePlayDialog.tsx` (missing dependencies, conflicts, pending updates, broken mods, paths too long for Windows, a newest save last played with another profile; opt-out stored on the profile), `UpdateWarnDialog.tsx` (game version changed since the last launch, never on a first launch), `SaveWarnDialog.tsx` (the newest save uses mods the profile lacks; an unreadable save does not ask), `GraphicsDialog.tsx` (the catalog offers graphics APIs and the profile resolves none; the answer is saved to the game's Graphics API setting, and Play asks once more if the last run exited within seconds under a non-recommended choice).
- **After a run:** a crash shows the Home crash card; a Mortar-started launch that ends with errors or a SMAPI crash opens the closed-with-errors dialog (`console/CrashDialog.tsx`).

## Tab header

Every tab except Home opens with a 52px strip (`TabHeader`, rendered by `ProfileWorkspace`) tinted from the profile's colour (the accent when it has none), with a hairline under it: the profile name, count chips (a zero count is left out; each opens the screen it counts), the first line of the profile's notes (hidden when compact), a spacer, and the page-actions slot. A tab fills the slot with `<PageActions>` (a portal), so each tab keeps its buttons in one place. Every button, icon button and split-button half in the slot is 34px tall, set once on the slot (`pageActionsSx`), never per tab.

## Mods tab

Code: `mods/ModsTab.tsx`, `ModList.tsx`, `ModCards.tsx`, `ModListVirtual.tsx`, `ModMenu.tsx`, `ModsHeaderActions.tsx`, `ListColumnMenu.tsx`. The profile's mods as a **Grid** or a **List** (a table), both virtualized, sharing one details panel ([Mod detail](#mod-detail)). Under the header: an "N updates ready" banner, then the controls row (view toggle, **Group by**, filter, **Show**).

- SMAPI's bundled Console Commands and Save Backup, and Mortar's console bridge, are installed in every profile and never shown, listed, counted or removable.
- The last loaded list for a profile stays on screen when the tab is left and returns, refreshed in the background; the first load shows skeleton rows.
- **List columns:** cells never wrap (ellipsis, full text in `title`); Author, Source, Category and the extras drop out below 960px even when enabled. Column choice, order and sort are global (`settings.json` `listColumns`, `listSortColumn`, `listSortDir`) and sort applies within each group. Category is the Nexus category from cached page details: opening the list makes no network call for it.
- **Grid cards** are deliberately simple: no switch (the enable toggle lives in the details panel). Remove acts on the whole archive entry the mod came in, and its confirm names the other mods in it. Card picture: Nexus `picture_url`, fetched once at install, cached under `cache/`, served at `/mod-picture/`.

### Views

Both views show a pin icon for a kept version and last-run error and warning badges; the New dot on a card means the Nexus page has a newer file or changelog than the user last saw ([architecture.md](architecture.md#nexus-mods)). **Startup** column: the mod's share of the latest startup report ([Startup timings](architecture.md#startup-timings)).

### Grouping

**Group by** (`listGroupBy`) gives collapsible headers with counts in both views; collapse state is per game, and the top group's header stays pinned on a solid surface.

- A mod with several tags appears under its first, which a tooltip says. Status takes the first match of Problems, Update available, Enabled, Disabled. Framework puts content packs under the framework they are for.
- Custom categories are stored per game in `<datadir>/categories/<game>.json`; deleting one moves its mods to Uncategorized.
- **Implicit mods** (`implicitModsView`): Always shown, Nested (under the first mod in its group that requires it) or Hidden (except while a search or Show › Implicit asks).
- **Install reason:** Explicit for a mod the player asked for, Implicit for one Mortar brought in for another. A download holding several mods has one main mod that takes the download's reason; the others read Implicit and offer no Mark items. Only **Mark explicit** and **Mark implicit** change it, each writing a history event. Removing mods with implicit leftovers nothing staying requires adds a checked **Also remove N libraries only these needed** to the confirm (Undo restores them).
- **Mod groups** (`profile.json` `groups`): a header switch toggles every mod in the group; deleting a group keeps its mods.

### Selection, removal and menus

- **Bulk select:** click, Ctrl/Cmd click, Shift click for a range in the current order, Ctrl+A for every visible filtered mod, Esc to clear. With two or more selected a one-line bar acts on them in one confirm; every action except **Share selection** locks while the game runs the profile.
- **Removing a mod that enabled mods require** (their `Needs` minus optional, or ContentPackFor) warns **N mods need this**, naming them, with **Remove anyway** and **Remove all N+1**; the mod menu and selection bar share the confirm. Undo restores every entry the remove took out.
- **Mod menu** (`ModMenu.tsx`): right-click, the Menu key or Shift+F10 on a focused item, or the card's ⋯. Groups split by dividers with Remove last, in red; everything that changes the profile is disabled while it is locked. **Move up (loses files)** and **Move down (wins files)** exist for a game whose profile holds its loader, since a later package wins a file two mods both provide. Problems' **File conflicts** tab lists those files as info rows (`fileConflict`, grouped by `groupConflicts`) by winner and the mods it beats, with **Make … win** moving a loser to just after the winner (`MakePackageWin`).
- **Endorsement prompt** (`mods/EndorsePrompt.tsx`): a snackbar after clean launches of an enabled Nexus mod the user has not endorsed or abstained, when Settings › Accounts asks for it.

### Optional files

Nexus files without a manifest laid over a mod's main file ([architecture.md](architecture.md#storage)). Code: `mods/ExtraFilesChip.tsx`, `mods/updateReview/OptionalUpdates.tsx`, the details panel's Optional files section.

- Each shows as a row nested under its main mod (own switch and Remove with Undo; no selection, sorting or columns), dimmed while it or its main mod is disabled. Removing the main mod says in the same confirm that its optional files go too.
- Updates re-apply them: a checked **Also update N optional files** box queues a newer version after the main file (matched by the author's `file_updates` chain, else the file name without its version).
- Files that replace some of the same files are alternatives under **Choose one**: enabling one disables the others, with a toast naming them and Undo.
- When Mortar cannot tell where an optional file's files go, **Place <file>** asks for the source and target folders; one whose main file is not in the profile fails with a toast offering **Open files page**.

### Adding mods

- Dropping an archive on any page of the game (settings included) installs it into the open profile; after an install, a toast offers **Add them** for required dependencies the profile lacks (optional ones are left to Problems).
- An add that finds no manifest or several variants opens **Choose the mod folder** / **Choose a variant**, with **Show archive contents** (the same tree as the downloads dialog's preview).
- A batch install (several archives, a multi-select pick, the extra mods or downloads folder) toasts once with failures counted in the body and **Undo all** (one history event).

### Library callouts

Compact Alerts above the list ([Surfaces and colour](#surfaces-and-colour)):

- **Files an update no longer includes:** Show files, Keep, and a contained error **Delete** that moves them to Mortar's trash with Undo.
- **Mods in the game's Mods folder that differ from this profile** (`mods/GameModsReviewDialog.tsx`, `profiles/GameModsDialog.tsx`): it clears once the profile matches and an unreadable folder never raises it. **Copy into profile** leaves the game folder untouched; **Move into profile** also removes those folders from it. Both are disabled while the profile is locked, and one undoable history event covers the sync.
- **Hidden inside this mod** (`mods/HiddenInside.tsx`, with `showDotHiddenMods` on): **Unhide** asks first, then renames the dotted folders without their dots so SMAPI loads the mod.

## Mod detail

The **details panel** (`mods/DetailsAside.tsx`, `DetailsHeader.tsx`, `NexusDetails.tsx`, `ModDependencyTree.tsx`) is shared by both Mods views: a right-hand column while a mod is selected and absent otherwise, so the list takes the full width; at least 300px, at most half the tab, resized by dragging its left edge or with the arrow keys on that handle. **More details** in a mod's menu selects the mod; there is no separate details dialog. The panel and list share one read of a mod's Nexus details per session.

Top to bottom: identity (the enable toggle at the top right with an aria-label naming the mod), update banner and problem line, pin and skip controls (pinning stops update offers until unpinned), note and tags, then sections for dependencies (a transitive tree, cycles shown once), other profiles holding the mod, versions (pin reason, **Change version…** installs any version the source lists, asking first on a pinned or frozen mod, and **Roll back** after a confirm), settings ([Config page](#config-page); **Reset** deletes the file after a confirm so the mod writes a fresh one), compatibility, and for a Nexus-installed mod **More from the Nexus page**. BBCode renders as paragraphs, headings, list items, bold, italic and http(s) links; other markup is dropped (`mods/MarkdownView.tsx`). The foot has **Show files** and **Remove** side by side, Remove outlined in the readable destructive red, as destructive menu items are.

## Config page

The **Config** tab lists every mod with a config source and edits the chosen one beside the list. Code: `mods/typedConfig/` (`ConfigTab.tsx`, `ConfigPane.tsx`, `Widgets.tsx`, `StringWidgets.tsx`, `keybind.ts`, `convert.ts`). The edit model is typed rows ([architecture.md](architecture.md#profile-operations)).

- The list follows the Mods page's **Group by**, sort, collapse state and headers (the same settings, so a collapsed group stays collapsed in both). A last group, **Loader and other**, holds `.cfg` and text files no mod owns (such as `BepInEx.cfg`). One `configsvc.Mods` call feeds it, cached per profile; the mod menu's **Edit config** reads that cache, so opening a menu fetches nothing.
- The editor takes the full width with no close button and no title (the selected row names the mod). A mod with several files gets a **Config file** dropdown listing a SMAPI `config.json`, each BepInEx `.cfg` and each other text file the mod keeps in BepInEx's config folder.
- **Text files** have no schema: one monospace box (up to 1 MB). Save writes atomically, keeps a UTF-8 byte-order mark the file had, and is refused while the game runs or when a JSON (comments and trailing commas allowed), YAML or XML file does not parse.
- **Saving:** each change saves a moment after it is made; a failed write (including one refused because the game runs the profile) shows **Could not save: <reason>** under the row.

### Rows and value types

A row shows the entry's name (the mod's label, else the key split into words at camelCase and number boundaries, raw key as tooltip), an info button when the mod gave a description, the control, and a **Reset to default** icon when the value differs. Control by type is `Widgets.tsx` and `StringWidgets.tsx`.

- **Type** comes from the JSON type of the current value, else of the shipped default (a null, or a number or boolean written as text, takes the default's), and a write keeps it: a boolean stays `true`, never `"true"`.
- A **Content Patcher** pack's values are all strings, so its `content.json` `ConfigSchema` decides and a write stays text (`"false"`, several choices comma-joined): `AllowValues` of exactly true and false is a switch, other `AllowValues` a select (multi-select with `AllowMultiple`), none a text field; `Default` is what reset returns to.
- A text value gets the editor its shape calls for, written back in the form the mod already uses: a **keybind** (`None`, a SMAPI key list `LeftShift + F5`, or a Unity InputSystem path `<Keyboard>/backquote`) gains **Record** that replaces the whole value in its own syntax and a clear button (writes `None`, or nothing for an InputSystem path); a comma list of words or a number list or vector keeps its own separator. A description that lists its values keeps the suggestion field.
- **In-game menu:** a mod whose Generic Mod Config Menu the bridge captured has no `config.json`; its changes are **Applied when the game next starts**, a colour, image or custom widget reads **Change it in the game**, **Presets** is hidden, and **Reset all** asks to discard every waiting change.

## Saves tab

Every save in the Saves folder with its fit for the profile, as a list or a card grid (remembered per game). Code: `saves/SavesTab.tsx`, `SaveRow.tsx`, `SaveListRow.tsx`, `BackupsDialog.tsx`, `SaveBackupsButton.tsx`.

- The list is built like the Mods list (sticky header, `--m-row` rows, the Mods list's hover); the "Needs N mods" line spans the row beneath. A calendar game's save with no SaveGameInfo date reads **Unknown** with a tooltip; a game without a calendar shows an em dash.
- **Needs line:** when the last run's recorded mod list has mods the profile lacks or has disabled, with **Show** and **Add the missing mods**; hidden when nothing is missing.
- **Locks:** Add all, Add the missing mods, Copy from another profile and Restore read "Stop the game to change mods." (or "to restore saves") while the game runs.
- **Back up now** zips that save only, labelled Manual and kept until deleted. Backup rows show their cause; **Keep forever** disables Delete.
- **Restore** names the saves it overwrites and says the current Saves folder is backed up first. The **Save backups…** dialog is read afresh on every open and excludes zips another profile took. The per-save backups popover is anchored to its button, closed by Esc with focus back on the button.

## Browse tab

Finds mods to add to the open profile; the shared header has no page actions here. Code: `browse/` (`BrowsePage.tsx`, `BrowseToolbar.tsx`, `BrowseFilters.tsx`, `BrowseShow.tsx`, `browseSort.ts`, `ResultCard.tsx`, `CardAction.tsx`, `cardState.ts`).

- **Source:** **All sources** (when more than one is searchable) then a chip per source in catalog order. A gated source (itch.io without an API key) is disabled with its reason as the tooltip. With nothing typed every source lists its top mods, so the tab opens on results.
- **Sort** offers best match and only the sorts the chosen source orders by on its own server, named for that source's metric (`browseSort.ts`). **All sources** offers best match, most downloaded and recently updated: the last two merge every source's page into one order, and a hit without a value (a GitHub repo has no download count) lands after those that have one. A sort the source does not offer falls back to best match.
- **Show** (filter-icon button, primary while any row is not Off): **Mods in this profile**, **Obsolete** and **Broken**, each **Off**, **Gray out** (dimmed, its action stays readable) or **Hide**, and Obsolete and Broken also **Flag** (a warning triangle before the name, in the warning colour or for broken the error colour, named in the title's accessible label; no outline); **Broken** is disabled with a tooltip when the game has no compatibility list.
- **Results:** a card has a thumbnail read through the local picture cache, with the mark of the source Add installs from in its top-right corner, name, author (accent colour), two-line summary and stats line. In the grid the action sits under the picture, as wide as it; the card carries no open-page button (the page opens from its menu and details); list rows keep an open-page icon. Action labels stay short, each with its full sentence as tooltip and accessible name. A mod several sources list is one card with a mark per source that picks where it installs from: stacked in the picture's corner in the grid, a badge line under the stats in list rows.
- **Footer bar** under the results, outside the scroll area: "N results", "· N hidden" for hits Show hid and "· <source> did not answer" for a failed source. More than 20 results page at the foot.
- **Action** (`CardAction.tsx`, `cardState.ts`): Nexus **Download** for Premium (**Sign in** while signed out), **Get** (opens the files page) for a free account, **Add** on every other source; progress shows in place, then **Added**. The game's loader is not listed (Mortar installs it from the game's page, `applyModes`); a companion Mortar installs itself shows **Built in**.

## Problems tab

A full-height tab showing one section of the profile's problems at a time. Code: `mods/ProblemsTab.tsx`, `problemSection.ts`, `problemGroups.ts`, `ProblemRows.tsx`, `problemFix/`, `problemReport.ts`, `shell/SectionStrip.tsx`; each kind's plain-language explanation is `problemWhy.ts`, backed by [user-guide.md](user-guide.md).

Under the shared tab header, a full-width **section strip** (one MUI tab per non-empty section, label and count chip) scrolls sideways when it does not fit and never changes height; an empty section is never shown.

- **Which section shows:** the first that holds something broken (missing requirements, broken mods, damaged files, run errors, failed plugins, plugins shipped twice), else the first with anything. A choice is remembered per profile while Mortar runs.
- **Page actions:** **Find the mod that crashes the game…** (a crash bisect: it turns mods off in halves and relaunches until the crash stops; enabled only when the newest run crashed with no suspected mod, otherwise off with the reason as tooltip, and the palette entry follows the same rule), **Asset map** and **Copy report**.
- Checks that could not run offline show the connection warning.

### Sections

Section order and membership: `problemSection.ts`. Rules a change could break:

- **The count** comes from one helper, `problemCount`, feeding the tab chip, sidebar badge, the Home hero and cached profile badges, so it is zero only when the tab has nothing to fix. The Mods tab carries no problems banner. While problems load, the chip shows the profile's last known count.
- **Never counted:** Cosmetic or harmless, Dismissed, Slow startup, and Redundant (warnings, never the chip). Conflicts count except harmless cosmetic asset conflicts.
- **Conflicts:** the same mods with the same winner and fix are one row listing every asset.
- **Slow startup** (`mods/slowStartups.ts`): a mod adding 3 s or more, or a content pack adding 1 s or more, from the latest startup report; a framework that loads content packs (Content Patcher) is never a row itself.
- **Redundant:** mods replaced by an enabled replacement, packs every edit of which later packs overwrite, and C# mods doing the same job. A same-job group is one row with a **Remove one…** menu. Every row has a quiet **Dismiss** for a finding the player judges wrong; dismissed rows return by themselves when a different set of mods makes the mod redundant.
- **Damaged files:** **Repair** downloads again from the source, or for a local mod re-extracts from its archive if still in downloads.
- **Paths too long:** **Open storage settings** for a file the game reads in the profile, where moving the data folder shortens it; no button for one placed in the game's own folder, which only a shorter game folder fixes.

### Rows and fixes

- Warnings are amber tiles; information rows (Cleanup included) are neutral tiles with an info icon. A missing or optional-dependency row links to that UniqueID in the Load order tab. The mod list shows the same problem triangle and tooltip as a card, beside the name so it stays visible when the Status column is narrow or hidden.
- **Fix buttons run in order of weight:** the suggested fix first (a Content Patcher **Set … to …**, or **Make a pack win** when there is none) filled in the warning colour on real conflicts, then **Disable** outlined, then **Dismiss** as a quiet text button last; dismissed rows swap Dismiss for **Restore**. Applying a Content Patcher **Set to** suggestion remembers that field value so the reverse hint stays hidden until the setting changes ([architecture.md](architecture.md#saves)).
- **Conflict rows** expand a **Why?** section (`mods/ConflictWhy.tsx`) ending with a **Show conflicts** link into the **Asset map** (`mods/AssetMapDialog.tsx`), which lists per asset each mod's edit in load order with the winner marked; a loser offers **Make win** only when loading it later decides the asset.
- A load-order dependency chip names the dependency: an installed copy's name, else the Nexus page the metadata dataset lists for its id (`DependencyNames`), else the bare id.

## Load order tab

A read-only list of the open profile's enabled mods in the order the game loader chooses ("The game loader chooses this order. Mortar does not change it."). SMAPI sorts by dependencies: required and optional dependencies before dependents, content packs after their ContentPackFor framework, otherwise alphabetical (ModResolver). Code: `mods/LoadOrderTab.tsx`, `loadOrderText.ts`; `mortar profile load-order` prints the same list. The filter fills the controls row (the shared `ControlsRow` with a growing `SearchField`, as on Mods, Saves and Console); a required dependency the profile lacks is a red chip and a dependency cycle is flagged on its rows.

## In-between moments

- **Dropping an archive:** a dashed drop zone fills the window 12px in from the edges and says where it goes. It shows no file name: WebKitGTK and Wails expose none before the drop.
- **A Nexus link:** on a game's profile screen the link installs into the open profile at once with a toast; on any other screen a dialog asks which profile (the open one primary), and further links queue behind it. A link never brings an open window to the front, so clicking downloads in browser tabs keeps the browser in front; only a window closed to the tray reopens, and a minimised window gets a desktop notification whose Show action raises it. A refused link (wrong game, another account, expired, malformed) is a toast saying why.
- **Two copies of one mod** (`mods/DuplicateDialog.tsx`): the dialog preselects the newer or Nexus-sourced copy and switches the other off rather than deleting it; **Decide later** leaves both as a problem.
- **Loading:** a pane that waits on data shows the shared `SkeletonRows` in the shape of the content (announced by a label), never an empty pane.

## Console tab

The loader's log as it is written, monospace, in columns; warnings and errors get a tinted row. Code: `console/ConsoleTab.tsx`, `VirtualLog.tsx`, `LogActions.tsx`, `LevelToggles.tsx`, `RunsPicker.tsx`, `consoleLinks.ts`, `smapiUpdateNotes.ts`.

- Filter controls sit in the filter row; **log actions** (copy, save, search all runs, clear, share) are icon buttons in the tab header's page actions, not the filter row. **Share log** shows the size, says the log becomes public at a link, and asks before uploading to smapi.io/log.
- **Runs picker:** This session, or one of the profile's last 20 recorded launches; a past run loads its stored log read-only, and Save log and Share log use it.
- **Links:** in visible rows only, an exact installed mod name or `UniqueID` selects the mod, and a path under the profile's `mods/` folder or the game folder opens its folder.
- **Update alert tags:** each line of SMAPI's **You can update** alert ends with a **Mortar:** tag saying how Mortar treats that version (`smapiUpdateNotes.ts`, from `UpdatesResult.Held` and the update list).
- **Input line:** hidden until this profile is running and This session is shown. Enter sends the command through the Mortar SMAPI Bridge and echoes it into the log as a Mortar line prefixed `>`; the last 100 commands of that game are kept in memory only.

## Performance tab

One scrolling page with no inner switch; the tab header is its only toolbar. Code: `console/PerformanceTab.tsx`, `StartupPanel.tsx`, `StartupSections.tsx`, `StartupTable.tsx`, `PerformancePanel.tsx`, `PerformanceComparison.tsx`, `useBridgePerf.ts`.

- **What to do:** up to three one-line findings from the selected run, each with one button (a dominant mod, a slowdown against the previous launch, SMAPI's own mod loading); with none it says nothing stands out. Thresholds: `startupView.ts`.
- **Slowest mods:** a sortable table with a **Why** phrase from the mod's own timings; mods under 50 ms in both runs fold into **N other mods**, and **Game and SMAPI** holds the unattributed rest. Mortar's bridge row carries a **Mortar** label.
- **In game:** a collapsible section of per-profile saved reports compared in the tab. Empty states: before any measured launch, copy that says to play the profile; before an in-game measurement, **Start measuring**.

### In-game measurement

Start measuring and Show report are off while the game is not running this profile or a past run is shown. The source depends on the loader's `perf` capability:

- **Console fallback** (no companion measurement): SMAPI's `performance enable` and `performance summary`.
- **SMAPI with its bridge** (the same panel as BepInEx's; it replaces the console report): the bridge times each mod's handlers of the per-frame events; no measured launch is needed. The **Baseline** line is the frame less the mods' timed code, not a launch with no mods. A row is a mod's update, render and one-second handlers summed per frame; Harmony patches and work outside events stay in the baseline.
- **BepInEx:** the bridge measures only on a launch measured from Startup, so the empty state says to use **Measure next launch** first. A row counts Harmony prefixes, postfixes and finalizers and the plugin's own Update, LateUpdate and FixedUpdate, exclusive of nested timed calls, not transpiled code or other threads.

## Profile management

A page, as Concrete's. Code: `profiles/`, `game/ProfileMenuItems.tsx`, `game/NewProfileDialog.tsx`, `share/`. Header: back, **Import** menu, **New profile…**; below, every profile as a sortable row (dnd-kit) with colour and icon mark, summary, **Share** and a ⋯ menu of the shared profile items; asides for saved groups and recently deleted profiles.

- **Import** (the switcher's Import group mirrors it): the game's Mods folder, a link or file, a backup (**Restore from a file…** makes a new profile) and **Import…** from other managers' profiles (r2modman, Gale, Vortex, Mod Organizer 2, Stardrop). Bundled SMAPI and the bridge are omitted from every import preview.
- **Damaged:** each profile folder whose `profile.json` cannot be read gets **Repair** (from the latest history snapshot: keeps `mods/`, moves the broken file aside as `profile.json.damaged-<timestamp>`, toasts Undo; off with a reason when no snapshot exists), **Open folder** and **Move to trash**. Damaged rows are omitted from the sidebar and every profile picker.
- **Recently deleted:** the 30-day trash ([architecture.md](architecture.md#profile-operations)); **Delete permanently** and **Empty trash** are error-coloured and confirm with **This cannot be undone.**; **Delete** on a profile says it stays restorable and toasts Undo.

### Profile menus

The switcher's context menu and the row ⋯ menu share `ProfileMenuItems` (dividers split rename and cover, history and checks, copies and compare, saved groups and templates, shortcuts, Delete last). Sending, the Nexus collection draft and Thunderstore exports are reached from **Share…**. Send, Compare and Delete dialogs stay mounted so they survive the menu closing.

- **Check this profile** (`profiles/HealthDialog.tsx`): findings with a **Repair** each (none when Mortar cannot fix it) and **Repair all**. Mortar also runs the check quietly once a week per profile (and after recovering from a crash), never while the game runs and only after startup's first problem check; a finding shows an amber shield pill on the profile's sidebar and Profiles rows. It never toasts.
- **Templates:** applying previews, merges, queues missing downloads and toasts Undo. It records its own history event, after a baseline event when the newest event is not the current mods; Undo reverts to the baseline and restores the game settings and launch options the template replaced.
- **Saved groups** (a per-game library made from a group's **Save to library…**): adding one creates the group in the profile, a clashing name gets " (2)", " (3)", and one Undo reverts the whole add. Removing a mod from a saved group removes its optional files too; the last mod cannot be removed.

### Change history

The open profile's History (`profiles/HistoryDialog.tsx`, `HistoryTimeline.tsx`, `HistoryEventRow.tsx`) is a wide dialog: a timeline grouped by day with **Mark current as known good** restore points and **Compare two…**. **Undo these changes** takes the profile back to before the entry, so newer entries are undone too; **Revert this mod** undoes only that mod's change. Event count is capped by Settings (`historyEventsKept`); snapshot file bytes are not ([architecture.md](architecture.md#profile-operations)). A revert that cannot find store files names the mods and offers **Download missing mods**.

### Edit profile

A solid-paper dialog without a transition, as tall as its tallest tab. Code: `profiles/EditProfileDialog.tsx`, `OverrideRows.tsx`, `LaunchOptionsBlock.tsx`, `LaunchPresetsBlock.tsx`.

- **Keep this profile's saves separate:** turning it on asks first and explains that while the profile runs the shared saves folder is set aside and returns when the game closes with nothing deleted; turning it off applies at once.
- **Overrides** ("These apply only to this profile."): each row is **Use game setting (labelled value)** or an explicit value, for the per-game settings a profile may override.
- **Launch options:** Mortar sets `--mods-path` itself and refuses `--mods-path`, `--no-terminal`, `--skip-terminal`. **Launch prefix** and **Launch environment** apply to direct launches only (unavailable on Windows for the prefix), and a **Direct launch preview** shows the argv and environment; Steam launches receive neither ([architecture.md](architecture.md#launch)).
- **Launch presets** and loader-specific options save at once, not with the dialog's Save.

### Compare and merge

- **Compare…** diffs this profile with another of the user's, a friend's link or file, or (Stardew) a multiplayer host's list. **Make <other> match <A>** applies only the checked rows to the target: mods only in the source and enabled-state differences are copied from the store, version and source differences switch the target's entry to the source's, and mods only in the target stay (no removal). Buttons are disabled with a tooltip while the game runs the target.
- **Merge into…** (Profile › Copies) never changes the source and removes nothing from the target; added mods keep enabled state and config, and the target's History gets one undoable event. **For mods in both** keeps the target's versions or takes the newer.

### Share

**Share** (`share/ShareDialog.tsx`, `DestinationGrid.tsx`, `DestinationPanel.tsx`, `MethodPanes.tsx`, `Included.tsx`) is two steps in a dialog of constant width: destination tiles, then the destination's pane. A share from the Mods selection limits link, file and mod list to those mods.

- A tile is greyed only when it cannot carry anything and says why on the tile: with only local archives, Nexus Mods and Thunderstore read "Local archives can't go here" while Mortar (a `.mortar` file includes them), Nearby computer and Mod list stay open. The destination used last for the game takes focus with a **Last used** caption.
- **Mortar pane:** **Link | File**; File is preselected over about 240 mods or when the link is too large. A link (`https://mortar.rethunk.tech/stardew/p#...`) shows a meter of its length against Discord's 2,000 characters. The **Included** block lists each option in force; libraries and supporting mods are greyed out with the reason on a link, since a link has no room. Every share carries pins, groups and install reasons.
- **Mod list pane:** Markdown, Plain text or Discord; Discord's list is split into messages of at most 2,000 characters, copied one at a time.
- **Nearby computer:** sending to an unpaired computer opens a links-only confirmation (config files stay here); a paired send copies store files first, then opens Import on the receiver ([architecture.md](architecture.md#sharing)).

## Import

A wide "Import profile from…" dialog over the dimmed game screen with tabs **Link** and **.mortar file** (archives go into an existing profile through Add archive or a drop, not Import). Nothing downloads before the user confirms; closing leaves nothing behind. Code: `share/ImportDialog.tsx`, `ImportPreview.tsx`, `ImportFooter.tsx`, `useImportFlow.ts`.

- **Preview:** mod tiles with an import state in place of a version (installed, download, dependency, check later, unavailable); an unticked mod is left out. Problems before download (removed from Nexus, broken for this game version, missing dependencies, one click per download on a free Nexus account) are compact items with their action.
- **Footer:** **New profile from link** creates the profile (named "<name> (2)" with the next free number when the name exists) and fills the download queue; **Replace <profile>** asks first, lists the entries it removes, keeps local-only mods and writes a history event so the change can be reverted.
- The Link tab reads the clipboard only from **Paste from clipboard**; a `mortar://` link, a dropped `.mortar` file or a second launch opens the dialog filled in and previewed. When the import needs Nexus downloads and no account is signed in, a banner offers Nexus settings and the import buttons stay off.

## Sync offers

With a sync folder set (Settings › Mods and profiles), a change another computer made to a profile is announced once by an info toast with **Review**, which opens **Changes from your other machines** (`sync/SyncOffers.tsx`). When both sides changed a profile Mortar does not merge: the block reads **Both sides changed it, so Mortar will not merge them. Pick one side.** (**Use theirs** or **Keep mine**). Mod files are never part of a sync; mods an apply adds download from their sources.

## Download queue

A side sheet (`queue/`, `downloads/`; Ctrl+J, the title bar pill or the palette). Rows are 54px tall; Skip, Cancel and dismiss are icon buttons (Retry on a failed row stays a labelled button), every action on the row's right edge. Closing the sheet keeps the queue running; history length is `internal/queue/history.go`.

- **Needs your click** (free accounts): the head item with Open download page and Skip; the next page opens when it finishes. An update or latest-file item is skipped on its own once the profile already holds that file or a newer one from the same page ([architecture.md](architecture.md#nexus-mods)). Premium accounts download without clicks.
- **Choose a file** (a GitHub release with several archives) and **Check this download** (a GitHub download whose mod SMAPI's update API does not tie to its repo: Install anyway or Skip) ask before installing. A download over 2 GB opens a warning-coloured confirm (`queue/LargeDownloadDialog.tsx`) before it starts: the mod's name, its size in GB, a sentence that this is unusually large, **Cancel** (focused, and what Esc does) and **Continue anyway**; several held downloads ask one after another, and the queue sheet shows the same choice as a card.
- **Pause all** stops new starts; the in-flight download finishes. A downloading row has Pause and Cancel; a waiting row has move up, move down, **Download now** (to the front, pausing the last running download if every slot is busy) and Pause; a **Paused** section lists paused rows ("Paused at 1.2 GB of 4.9 GB", or "Waiting for a free slot" for one that made way) with Resume. Failed rows say why and their button reads **Resume** when a partial file is on disk (else Retry); Cancel is offered only while downloading, not installing.

### Update review

Code: `mods/updateReview/` (`ReviewList.tsx`, `Row.tsx`, `ReviewFooter.tsx`, `SourceGroups.tsx`, `KeptGroup.tsx`, `Changes.tsx`, `OptionalUpdates.tsx`).

- Skipping or pinning is how a mod is left out of **Update all**; both apply even to updates Mortar cannot download. A row with a caution shows **Update despite this**; until ticked its Update is off and Update all leaves it out.
- A Nexus update with no file that supersedes the installed one (`pickFile`, [architecture.md](architecture.md#nexus-mods)) stays listed without Update, noting **Pick the file on Nexus**; Update all, Update before Play and the run-error **Update** skip it. Unofficial SMAPI versions are labelled and never included in Update all.
- Pinned mods never appear in the list or Update all; a collapsed **Kept at this version (N)** group shows each with **Unpin**, which re-checks updates. Implicit mods' updates come last under **Libraries (N)**.
- A warning triangle on a name means the update notes mention a breaking change or new requirement. After an update, **Update <mod> in other profiles** names pinned destinations and skips them.

## First run

Mortar's own setup finds launchers, not games, and opens until finished once (`launchersConfirmed`); there is no skip. Code: `firstrun/` (`FirstRun.tsx`, `FindStep.tsx`, `GameSetup.tsx`, `LoaderStep.tsx`, `ProfileStep.tsx`), `firstrunTour/`.

- "Welcome to Mortar" lists each launcher Mortar reads on this OS as an accordion; a folder belongs to one launcher (Minigalaxy's install folder is not also GOG's). With none found the first row starts open, and detection runs again when the window regains focus. Settings › Launchers shows the same rows. **Continue** is always enabled.

### Game setup

Each game runs its own setup when first opened from Game Select while not set up (folder unknown or no profile): a non-interactive stepper **Game folder**, the game's loader step when it has one, **First profile**. **Choose folder…** falls back to a typed path field where there is no native folder dialog (server mode).

- **Loader** (if missing) starts by itself, unattended, with no button; a failure shows its error with **Retry**. On Windows with Steam, **One step in Steam** offers **Set it in Steam for me**, which fails with a toast while Steam is running.
- **First profile:** import from the game's `Mods` folder when it holds mods, start empty, or from a shared link (asks for the Nexus sign-in first when the import needs downloads).

## Confirmations

**Toasts** (`toasts/`): bottom right, stacked (at most three; the oldest goes first), a coloured edge by kind and a dismiss button; 5 s (10 s for errors and warnings), hover pauses the countdown. A toast about a mod shows its picture from the card cache. Toasts stay in the session's notification history ([Window](#window)).

- **Undo** is offered after a history revert, deleting a profile, removing a mod, dismissing a save warning and the Mods selection bar's bulk Tag, Set category, Pin and Skip updates.
- **New downloads:** per **Offer new downloads**, the downloads folder raises "Add X to Y?" for one new archive and a **Review** toast for several; never while the game runs.

**Empty states** (`shell/EmptyState.tsx`): a tab with nothing to show centres an icon, a title, a line on how to fill it and, where one helps, a button. A console for a profile that has never run shows one; an empty log with past runs shows the log. **First-open tips** (`tips/TipBanner.tsx`): the Mods tab, Saves tab, Console and Share dialog each show a short dismissible banner the first time they open; Settings › General has **Show tips again**.

## Settings

Mortar's Settings holds only what is Mortar-wide, never one game's (the one exception is Downloads' default profile for Nexus links, labelled with its game). Per-game settings are the [Game settings](#game-settings) page. Code: `settings/SettingsPage.tsx`, `SettingsShell.tsx`, `SettingsNav.tsx`, `PrefRow.tsx`, `prefSpecs.ts`, `sections/` (one file per page); the registry is `internal/settings/registry.go`. Docs never state a setting's default; `mortar settings get` shows it.

### Layout

- A full page with a back button (Esc also returns), a section list on the left and the page beside it. While open, the title bar shows a "Settings" tab and no game tab. A search field filters rows by label and description across every page.
- Every control is a row (label and a wrapping description on the left, the control on the right) in its own rounded tile; a section is a heading above its stack of tiles, in one content column capped at 880px. Under 560px wide a row puts its control under the label. Controls inside rows are filled, not outlined.
- A choice whose options each carry an explanation renders as radio cards; two to four short options as a segmented strip; longer or dynamic lists as a dropdown. Lists and multi-step flows open dialogs rather than growing the page.

### Pages

General opens first. Rows live in `settings/sections/`; rules a change could break:

- **General:** **Antivirus** is app-wide (Automatic, clamd socket, Custom command, Off): a flagged download is refused with **Install anyway**, which asks in an error-coloured confirm, records the override in the profile's History and installs unscanned; a scan that could not finish installs the mod and warns it was installed unscanned. **Settings file** import lists what would change by section and applies only the ticked ones ([architecture.md](architecture.md#settings-file)).
- **Appearance:** theme, accent, background ([Surfaces and colour](#surfaces-and-colour)), display (applied at once), mods view, sidebar.
- **Mods and profiles:** the **Sync folder** holds profile settings, mod lists and configs, offered to other machines using it; mod files are not copied ([Sync offers](#sync-offers)).
- **Downloads:** **Handle "Mod Manager Download" links** confirms before taking them from another app. The default profile for Nexus links is the one game-labelled row.
- **Accounts:** one section per account (Nexus Mods, GitHub, itch.io). Nexus **Untrack unused…** confirms that Nexus has no undo. GitHub uses the `gh` login when present, which raises the rate limit. The itch.io API key is kept in the system keyring and **Test** stores nothing on rejection.
- **Updates:** Mortar's own and the mods'; the background pass over every profile is fixed, the open profile's recheck while running is a setting (`prefSpecs.ts`).
- **Notifications:** a matrix of events by **In Mortar** (toast) and **Desktop** (an OS notification sent whether or not the window is focused); the update digest also paces the desktop notice.
- **Storage:** **Move…** copies the data folder to a chosen empty location, verifies the copy, records `data-location` at the default path, removes the old copy and restarts; it is off with a reason while the game runs, refuses when space is short or the target is inside the current folder or not empty, and shows determinate progress. **Usage** is measured in the background; **Clean up…** lists unused store items, older mod copies (newest kept) and leftover files, each ticked and sized; **Integrity** checks store files, and Mortar also verifies slowly in the background about once a week. **Retention** holds unused store item days, trash days and history events kept (per-game save backup and profile history rows live on the game's settings pages).
- **Shortcuts:** click a keycap to record a combo (Esc cancels); a combo used by another row is refused, naming that action; bindings persist in `settings.json`. Except Esc, shortcuts stay silent while typing or with a dialog open.
- **About:** **Open-source licences** lists every library, icon and artwork built into Mortar; things Mortar downloads at run time (SMAPI, BepInEx, the mod dataset) are not listed. **Diagnostics** runs one row per check kind with **Repair** where one exists, plus **Copy report** and **Save diagnostics…** (the redacted zip). A game that is not installed only warns when no game is.

### Game settings

Per game, on their own page (`settings/GameSettingsPage.tsx`), opened by the last icon button in the game screen's tab row after **Share**. It reuses Settings' layout (nav with a back button to the game, search, Esc returns); the title bar keeps the game tab highlighted. Pages: **Install**, the game's loader, **Play**, **Mods**, **Save backups**, **Console** (only for a loader with a console) and **Streaming**. Anything tied to a game's loader, launcher, saves or content format lives here, with a per-profile override ([Edit profile](#edit-profile)).

- **Install › Game folder:** **Browse…** is checked before save; **Use discovered** shows while a chosen folder overrides discovery. **Reset game install** (outlined, error colour) deletes the folder, SMAPI and the mods placed there while keeping Saves and Mortar profiles ([architecture.md](architecture.md#finding-the-game)).
- **SMAPI:** a version picker (Latest or pinned); **Install <version>…** for a pinned version asks first since every profile switches.
- **Mods › Preferred source:** an ordered list; when a mod is on several sites its card installs from the first.
- **Streaming › Stream overlay** (`sections/StreamOverlay.tsx`, `overlayUrl.ts`; stored as `overlayEnabled`, `overlayPort`, `overlayToken`): changes apply on the next Play. The bridge reports a `game` in `/state` that selects the OBS page's layout; each field has a live preview polled from `/state` and a copy icon for its OBS `file://` URL. The OBS Custom CSS selectors are in `overlayUrl.ts`.

## Browser extension

The MV3 extension ([Rethunk-Tech/mortar-browser-extension](https://github.com/Rethunk-Tech/mortar-browser-extension), own repo and version) on `www.nexusmods.com`. When the two sides do not share a native-messaging protocol, its popup, button and filter section say **Update Mortar: it is too old for this browser extension** or **Update the browser extension: it is too old for this Mortar**. Native-host requests: [architecture.md](architecture.md#nexus-mods).

- **Popup:** **On Nexus pages** (**Off (do not mark Nexus pages)** / **Highlight** / **Hide mods in profile**, stored in `chrome.storage.local` as `mode`) and a status line naming the connection state (Mortar not installed, not running, no profile open, the app's connection setting off, profile unreadable, connected).
- **Listings, search and collections:** in Highlight, mod tiles (`data-e2eid="mod-tile"` and Nexus's other tile attributes) whose Nexus id is in the open profile get an **In profile** badge, or **Update available** when Mortar knows of one; Hide removes those tiles; Off draws nothing, the Mortar filter section included. The app's **Browser extension connection: Off** has the same effect everywhere. Mortar's UI appears only on games the native host lists as supported. A disabled row always shows its reason as text.
- **Listing filters:** a Mortar section at the top of Nexus's filter sidebar (`#filters-panel`, built from Nexus's own classes) with three rows that dim tiles in place (the listing keeps its length), saved per game in the extension's storage: **Gray out mods in profile**, **Gray out obsolete** (title or summary says obsolete, deprecated or depreciated: the same words and rule as the Problems check's author-marked mods, `authorStatusWord` in `internal/problems/author_marked.go`, pinned by the extension's tests; needs no connection to Mortar) and **Gray out broken** (SMAPI's compatibility list for the version last played, [architecture.md](architecture.md#nexus-mods); needs Mortar installed but not running).
- **Mod page:** a read-only panel under the title showing the profile state, other profiles holding the mod, update availability and **Open in Mortar** (`mortar://<game>/mod/<id>`). Off removes the panel.
