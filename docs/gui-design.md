# GUI design

The design system lives in code: `frontend/src/theme/` (palette, density, the MUI theme), `frontend/src/shell/` (dialogs in `shell/dialog/`, menu rows in `shell/MenuAction.tsx`, `TitleMenu.tsx`'s headings and rules, `Tooltip.tsx`, `EmptyState.tsx`) and `frontend/src/brand/`. `frontend/src/lint/patterns.test.ts` and `bun run --cwd frontend i18n:dupes` fail a change that builds around them; `frontend/e2e/deck.pw.ts` checks the Steam Deck size, controller reach and 44px targets. Read those before any UI change. This page holds what code cannot enforce, then per screen the rules a change could break.

## Conventions

- Dark is the lead theme: screenshots, the README and store listings show it.
- Dialog footers: Cancel or Close is a text button, leftmost; at most one contained primary, rightmost; a destructive primary is contained `color="error"`.
- Layout reads spacing roles from `space` (`theme/density.ts`) rather than hand-set pixels, so a Density change moves every page together.
- A trailing "…" only on a control that opens further input (a dialog, a picker, a confirmation); clicking it never starts the work, and the dialog's own title has none.
- **Remove** takes something out of a profile, group or list; **Delete** destroys it on disk. The button, its confirmation and its toast use the same verb.
- Sentence case everywhere; labels, buttons, headings and toast titles take no full stop, body text does. No article where the meaning holds without it. A label reused mid-sentence is lowercased in code (`i18n/lowerFirst.ts`).
- One placeholder name per concept (`{name}`, `{count}`, `{profile}`, `{game}`, `{version}`, `{path}`, `{when}`, `{source}`, `{manager}`, `{loader}`, `{launcher}`, `{os}`, `{app}`); a plural holds only the counted clause.
- **Command palette** (Ctrl+K): a shortcut that does what an action does shows its keys on that action, never as a second row; an action needing an open profile is greyed with a reason.

## Game Select

Mortar opens on the last game used; Game Select shows at start only until a game has been opened, and otherwise from the title bar. Code: `frontend/src/games/GameSelect.tsx`.

One list of full-width rows of one fixed height, however many games the catalog holds; it never wraps into columns. Rows touch (no gap, no rounded corner) and the page scrolls. Sections, each foldable by its heading: **Recently played** (the three games opened with a profile or played most recently), **On this computer** and **Other games**. Names sort as typed, letters and digits only, so R.E.P.O. files under R.
- **Art** is the game's Steam `library_hero.jpg` read at runtime, never bundled, edge to edge under a light solid dim layer; text sits straight on it, kept legible by a text shadow, with no panel behind. A row never changes height or shifts or zooms its art on hover or focus, and no tile carries an edge stripe. A tile that cannot open (a game coming later) is desaturated under a darker dim.
- **Profile cards** (set-up games) cap at six then a **+N more** card; clicking one opens that profile and does not fire the row's open handler. No Play on a card.
- **Play** starts the game's last-played profile through the same Start path as the sidebar button (SMAPI install and Steam checks included); a game not set up shows **Set up** in its place.
- **Type to find:** typing outside any field, menu or dialog jumps to the first game whose name starts with (else contains) the letters, ignoring case, spaces and punctuation, and focuses the row so Enter opens it.

## Main screen

A left sidebar and a detail pane. Code: `frontend/src/game/MainScreen.tsx`.
- **Sidebar:** one vertical tab list ("Profile sections"): one tab stop on the open section, arrow keys and a gamepad's shoulder buttons move between sections, each tab points at the workspace `tabpanel` by `aria-controls`. A section the loader or game does not support is absent, and a group left empty loses its header. Settings is not in the sidebar (it stays in the Mortar menu and on Ctrl+,).
- **Play** (`frontend/src/launch/PlayControl.tsx`) is a full-width primary button split with a caret; its menu lists launch presets, **Set default preset…**, **Launch options…** (Edit profile, where the launch options are) and **Play without mods** (starts the game with no profile and does not lock mods).
- **Compact layout** (below 960px wide): the sidebar is always the icon rail, the hero becomes a one-line header, the mods search folds into a filter icon, other actions fold into icon buttons, the mod grid drops to two columns.

### Home

The hero, then the **Home page** (`frontend/src/game/Home.tsx`): an action row and a grid of cards (one column when compact, where no card sits alone).
- **Hero:** bleeds to the pane's edges; the name carries no actions beside it (they are in the action row and the profile menus). The cover is the image under a light solid dim layer that fades to transparent over the bottom 20% by an alpha mask; the hero has no background of its own.
- **Cover source, in order:** an image the user picked, the Nexus picture of the profile's most-endorsed mod, Steam's hero art for the game (read at runtime, never bundled), a solid tone. Never a random image.
- A crash card after a crashed run offers the cause, evidence, **Disable it and play** and **Find the cause…** (a crash bisect, [architecture.md](architecture.md#launch)).

### Launching

Code: `frontend/src/launch/`. While the game runs, Play becomes a Running status with **Stop game** and the profile's mod controls lock ("Stop the game to change mods."): switching, removing, installing, dropping archives.

The **launch screen** (`LaunchScreen.tsx`, `LaunchStages.tsx`) covers the game page from Play until the loader says the mods have loaded (a loader whose log has no such line: until its log starts), then says the game is running for a moment and closes. It is opaque: the game's hero art fills it, dimmed with the art-dim set.
- A row of marks: the launcher that started the game (only when one did), the game's square icon (else its hero art), and the loader's own icon (shipped with Mortar, never fetched; a plain mark for one without). The mark in progress carries the spinner on its border; it runs on to the line out, the line draws, then the next mark starts, so marks never jump ahead of each other. A loader that names each mod as it loads shows "Loading n of N mods" and a bar under its mark.
- The log box under the row has a fixed height so nothing moves; it leaves out the per-mod loading lines the bar already counts and colours warnings and errors. The slow notice (from twice the profile's usual start, else 90 s) and a failure cover the log box instead of adding height, each with its one contained action (**Keep waiting**, **Retry**). The notice names the stage that is stuck.
- **Cancel** calls off a launch at any stage and ends only the game's own processes, never the launcher.

Pre-Play dialogs (`PrePlayDialog.tsx`, `UpdateWarnDialog.tsx`, `SaveWarnDialog.tsx`, `GraphicsDialog.tsx`) are each omitted when nothing applies and follow the dialog footer convention ([Conventions](#conventions)). Pre-Play's opt-out is stored on the profile; the game-version warning never shows on a first launch; an unreadable save does not ask; the graphics answer is saved to the game's Graphics API setting, and Play asks once more if the last run exited within seconds under a non-recommended choice. A Mortar-started launch that ends with errors or a SMAPI crash opens the closed-with-errors dialog (`frontend/src/console/CrashDialog.tsx`).

## Tab header

Every tab except Home opens with a strip (`TabHeader`, rendered by `ProfileWorkspace`) tinted from the profile's colour (the accent when it has none): the profile name, count chips (a zero count is left out; each opens the screen it counts), the first line of the profile's notes (hidden when compact) and the page-actions slot. A tab fills the slot with `<PageActions>` (a portal). Every button, icon button and split-button half in the slot is `--m-control` tall, set once on the slot (`pageActionsSx`), never per tab.

## Mods tab

Code: `frontend/src/mods/ModsTab.tsx`. The profile's mods as a **Grid** or a **List** (a table), both virtualized, sharing one details panel ([Mod detail](#mod-detail)).
- SMAPI's bundled Console Commands and Save Backup, and Mortar's console bridge, are installed in every profile and never shown, listed, counted or removable.
- The last loaded list for a profile stays on screen when the tab is left and returns, refreshed in the background.
- **List columns:** cells never wrap (ellipsis, full text in `title`); Author, Source, Category and the extras drop out below 960px even when enabled. The menu groups them as the mod (name, author, category, notes, its own id), its versions (installed, plugin version, latest, released, updated), its health (status, needs, kind), its place in this profile, then its source (source, rating, downloads). A column shows only where it means something: the id under the loader's own name (UniqueID, GUID, ModID), Kind and Plugin version for BepInEx packages, the source figures for Nexus and Thunderstore mods (`contributedColumns.ts`). Figures come from cached Nexus page details or the cached Thunderstore listing (`ModFacts`): opening the list makes no request per mod. Column choice is per game with a global default (`settings.json` `games.<id>.listColumns`, `listColumns`); sort is global and applies within each group.
- **Grid cards** are deliberately simple: no switch (the enable toggle lives in the details panel). Remove acts on the whole archive entry the mod came in, and its confirm names the other mods in it.
- The New dot on a card means the Nexus page has a newer file or changelog than the user last saw ([architecture.md](architecture.md#nexus-mods)).

### Grouping

**Group by** (`listGroupBy`) gives collapsible headers with counts in both views; collapse state is per game, and the top group's header stays pinned on a solid surface.
- A mod with several tags appears under its first, which a tooltip says. Status takes the first match of Problems, Update available, Enabled, Disabled. Framework puts content packs under the framework they are for.
- **Implicit mods** (`implicitModsView`): Always shown, Nested (under the first mod that requires it, in whichever group that mod is) or Hidden (except while a search or Show › Implicit asks).
- **Install reason:** Explicit for a mod the player asked for, Implicit for one Mortar brought in for another. A download holding several mods has one main mod that takes the download's reason; the others read Implicit and offer no Mark items. Only **Mark explicit** and **Mark implicit** change it, each writing a history event. Removing mods with implicit leftovers nothing staying requires adds a checked "Also remove N libraries only these needed" to the confirm (Undo restores them). Disabling follows the game's **Disable dependencies** setting (per-profile override): Ask opens a dialog with checked "Also disable N mods that need it" and "Also disable N libraries nothing else needs" when either applies, Always takes them along, Never disables only the mod; enabling mirrors it with **Auto-enable requirements**.
- **Mod groups** (`profile.json` `groups`): a header switch toggles every mod in the group; deleting a group keeps its mods.

### Selection, removal and menus

- Every bulk action except **Share selection** locks while the game runs the profile.
- **Removing a mod that enabled mods require** (their `Needs` minus optional, or ContentPackFor) warns "N mods need this", naming them, with **Remove anyway** and **Remove all N+1**; the mod menu and selection bar share the confirm. Undo restores every entry the remove took out.
- **Mod menu** (`ModMenu.tsx`): groups split by dividers with Remove last, in red; everything that changes the profile is disabled while it is locked. **Move up (loses files)** and **Move down (wins files)** exist for a game whose profile holds its loader, since a later package wins a file two mods both provide.
- The endorsement prompt is a snackbar after clean launches of an enabled Nexus mod the user has not endorsed or abstained, when Settings › Accounts asks for it.

### Optional files

Nexus files without a manifest laid over a mod's main file ([architecture.md](architecture.md#storage)).
- Each shows as a row nested under its main mod (own switch and Remove with Undo; no selection, sorting or columns). Removing the main mod says in the same confirm that its optional files go too.
- Updates re-apply them: a checked **Also update N optional files** box queues a newer version after the main file (matched by the author's `file_updates` chain, else the file name without its version).
- Files that replace some of the same files are alternatives under **Choose one**: enabling one disables the others, with a toast naming them and Undo.
- When Mortar cannot tell where an optional file's files go, **Place <file>** asks for the source and target folders.

### Adding mods and callouts

- Dropping an archive on any page of the game (settings included) installs it into the open profile; after an install, a toast offers **Add them** for required dependencies the profile lacks (optional ones are left to Problems). An add that finds no manifest or several variants asks which, with **Show archive contents**. A batch install toasts once, with failures counted in the body and **Undo all** (one history event).
- Compact Alerts above the list: **Files an update no longer includes** (Show files, Keep, and a contained error **Delete** that moves them to Mortar's trash with Undo); **Mods in the game's Mods folder that differ from this profile** (clears once the profile matches; an unreadable folder never raises it; **Copy into profile** leaves the game folder untouched, **Move into profile** also removes those folders; one undoable history event); **Hidden inside this mod** (**Unhide** asks first, then renames the dotted folders without their dots so SMAPI loads the mod).

## Mod detail

The **details panel** (`frontend/src/mods/DetailsAside.tsx`) is shared by both Mods views: laid over the right edge of the list while a mod is selected, so opening it never changes the list's width or its columns; resized by dragging its left edge or with the arrow keys on that handle. **More details** in a mod's menu selects the mod; there is no separate details dialog.

The enable toggle sits at the top right with an aria-label naming the mod. Pinning stops update offers until unpinned. **Change version…** installs any version the source lists, asking first on a pinned or frozen mod; **Roll back** confirms; **Reset** on the settings section deletes the file after a confirm so the mod writes a fresh one. The foot has **Show files** and **Remove** side by side, Remove outlined in the readable destructive red, as destructive menu items are. BBCode renders as paragraphs, headings, list items, bold, italic and http(s) links; other markup is dropped.

## Config page

The **Config** tab lists every mod with a config source and edits the chosen one beside the list. Code: `frontend/src/mods/typedConfig/`. The edit model is typed rows ([architecture.md](architecture.md#profile-operations)).

- The list follows the Mods page's **Group by**, sort and collapse state (the same settings, so a collapsed group stays collapsed in both). A last group, **Loader and other**, holds `.cfg` and text files no mod owns (such as `BepInEx.cfg`). One `configsvc.Mods` call feeds it, cached per profile; the mod menu's **Edit config** reads that cache, so opening a menu fetches nothing.
- **Text files** have no schema: one monospace box (up to 1 MB). Save writes atomically, keeps a UTF-8 byte-order mark the file had, and is refused while the game runs or when a JSON (comments and trailing commas allowed), YAML or XML file does not parse.
- Each change saves a moment after it is made; a failed write (including one refused because the game runs the profile) shows **Could not save: <reason>** under the row.
- **Type** comes from the JSON type of the current value, else of the shipped default (a null, or a number or boolean written as text, takes the default's), and a write keeps it: a boolean stays `true`, never `"true"`.
- A **Content Patcher** pack's values are all strings, so its `content.json` `ConfigSchema` decides and a write stays text (`"false"`, several choices comma-joined): `AllowValues` of exactly true and false is a switch, other `AllowValues` a select (multi-select with `AllowMultiple`), none a text field; `Default` is what reset returns to.
- A **keybind** (`None`, a SMAPI key list `LeftShift + F5`, or a Unity InputSystem path `<Keyboard>/backquote`) gains **Record**, which replaces the whole value in its own syntax, and a clear button (writes `None`, or nothing for an InputSystem path). A comma list or a number list or vector keeps its own separator.
- **In-game menu:** a mod whose Generic Mod Config Menu the bridge captured has no `config.json`; its changes are **Applied when the game next starts**, a colour, image or custom widget reads **Change it in the game**, **Presets** is hidden, and **Reset all** asks to discard every waiting change.

## Saves tab

Every save in the Saves folder with its fit for the profile, as a list or a card grid (remembered per game). Code: `frontend/src/saves/SavesTab.tsx`.
- A calendar game's save with no SaveGameInfo date reads **Unknown** with a tooltip; a game without a calendar shows a dash. The needs line appears only when the last run's recorded mod list has mods the profile lacks or has disabled.
- Add all, Add missing mods, Copy from another profile and Restore read "Stop the game to change mods." (or "to restore saves") while the game runs.
- **Back up now** zips that save only, labelled Manual and kept until deleted; **Keep forever** disables Delete. **Restore** names the saves it overwrites and says the current Saves folder is backed up first. The **Save backups…** dialog is read afresh on every open and excludes zips another profile took.

## Browse tab

Finds mods to add to the open profile; the shared header has no page actions here. Code: `frontend/src/browse/BrowsePage.tsx`.
- **Source:** **All sources** (when more than one is searchable) then a chip per source in catalog order. A gated source (itch.io without an API key) is disabled with its reason as the tooltip. With nothing typed every source lists its top mods, so the tab opens on results.
- **Sort** offers best match and only the sorts the chosen source orders by on its own server (`browseSort.ts`). **All sources** offers best match, most downloaded and recently updated: the last two rank every source's hits as one list, leaving out a source that cannot sort that way (GitHub under most downloaded) and naming it in the footer; best match interleaves each source's own page. A sort the source does not offer falls back to best match.
- **Show:** **Mods in this profile**, **Obsolete** and **Broken**, each **Off**, **Gray out** (dimmed, its action stays readable) or **Hide**; Obsolete and Broken also **Flag**. **Broken** is disabled with a tooltip when the game has no compatibility list.
- **Cards:** in the grid the action sits under the picture and the card carries no open-page button (the page opens from its menu and details); list rows keep an open-page icon. Action labels stay short, each with its full sentence as tooltip and accessible name. A hit that may copy a better-downloaded mod of the same name by someone else says **Possibly a re-upload of … by …** under its author. A mod several sources list is one card with a mark per source that picks where it installs from.
- The footer bar under the results, outside the scroll area, shows the result count and what is missing from it: hits Show hid, a source that did not answer, a source a ranked sort leaves out.
- **Action** (`cardState.ts`): Nexus **Download** for Premium (**Sign in** while signed out), **Get** (opens the files page) for a free account, **Add** on every other source. The game's loader is not listed (Mortar installs it from the game's page); a companion Mortar installs itself shows **Built in**.

## Develop tab

Code: `frontend/src/develop/`. In its own sidebar group (**Create**), only for a game whose loader names a mod template. **New mod…** in the page-actions slot opens the wizard; each project is a card whose row of actions never wraps a label, with the last build's outcome as compact Alerts below and each compiler error a link to its file and line. **Add a project folder…** takes a folder already made from the template, and **Clone from GitHub…** (owner/name or address) clones one into the wizard's projects folder first. The wizard says why **Start mod** is off in its tooltip, remembers the author, team, id prefix and projects folder, and offers the GitHub repository only when `gh` is signed in.

## Problems tab

A full-height tab listing the profile's problems by section. Code: `frontend/src/mods/ProblemsTab.tsx`, `problemFix/` (each row's fixes as one split button: `fixActions.tsx`, `arrangeFixes.ts`); each kind's plain-language explanation is `problemWhy.ts`, backed by [user-guide.md](user-guide.md).

Under the shared tab header, every non-empty section is a heading with its count (in the error colour for a section holding something broken) above its rows on one quiet panel; an empty section is never shown. The advice and history sections (Slow startup, Redundant, Cleanup, Replaced, Dismissed) are links with counts under the list instead; choosing one opens it below them, and choosing it again closes it, remembered per profile while Mortar runs.
- **Find the mod that crashes the game…** (page action; a crash bisect) is enabled only when the newest run crashed with no suspected mod, otherwise off with the reason as tooltip; the palette entry follows the same rule. **Asset map** and **Copy report** are the other page actions. Checks that could not run offline show the connection warning.

### Sections

Section order and membership: `problemSection.ts`. Rules a change could break:

- **The count** comes from one helper, `problemCount`, feeding the tab chip, sidebar badge, the Home hero and cached profile badges, so it is zero only when the tab has nothing to fix. The Mods tab carries no problems banner. While problems load, the chip shows the profile's last known count.
- **Never counted:** Cosmetic or harmless, Dismissed, Slow startup, Cleanup and Redundant (warnings, never the chip). Conflicts count except harmless cosmetic asset conflicts.
- **Conflicts:** the same mods with the same winner and fix are one row listing every asset.
- **Slow startup:** a mod adding 3 s or more, or a content pack adding 1 s or more, from the latest startup report; a framework that loads content packs (Content Patcher) is never a row itself.
- **Redundant:** mods replaced by an enabled replacement, packs every edit of which later packs overwrite, and C# mods doing the same job (one row with a **Remove one…** menu). Every row has a quiet **Dismiss**; dismissed rows return by themselves when a different set of mods makes the mod redundant.
- **Damaged files:** **Repair** downloads again from the source, or for a local mod re-extracts from its archive if still in downloads.
- **Paths too long:** **Open storage settings** for a file the game reads in the profile, where moving the data folder shortens it; no button for one placed in the game's own folder, which only a shorter game folder fixes.

### Rows and fixes

- A warning row leads with the warning triangle, an information row (Cleanup included) with an info icon. A missing or optional-dependency row links to that UniqueID in the Load order tab. The mod list shows the same problem triangle beside the name, so it stays visible when the Status column is narrow or hidden.
- **Each row's fixes are one split button** on the right, after a quiet **Why?**: the button is the heaviest fix (the suggested fix, a Content Patcher **Set … to …** or **Make a pack win** when there is none), filled in the warning colour on real conflicts and outlined on cosmetic ones; its arrow opens the other fixes, then **Dismiss** after a divider. A row with only quiet fixes (a dismissed row's **Restore**) shows them as text. Setting fixes on one conflict are alternatives, so none applies them all at once. Applying a Content Patcher **Set to** suggestion remembers that field value so the reverse hint stays hidden until the setting changes ([architecture.md](architecture.md#saves)).
- **Why?** opens the row's explanation beneath it; on a conflict row it is the evidence ending with a **Show conflicts** link into the **Asset map**, which lists per asset each mod's edit in load order with the winner marked; a loser offers **Make win** only when loading it later decides the asset.
- **File conflicts** are info rows by winner and the mods it beats, with **Make … win** moving a loser to just after the winner.
- A load-order dependency chip names the dependency: an installed copy's name, else the Nexus page the metadata dataset lists for its id, else the bare id.

## Load order tab

A read-only list of the open profile's enabled mods in the order the game loader chooses ("The game loader chooses this order. Mortar does not change it."). SMAPI sorts by dependencies: required and optional dependencies before dependents, content packs after their ContentPackFor framework, otherwise alphabetical. Code: `frontend/src/mods/LoadOrderTab.tsx`; `mortar profile load-order` prints the same list. A required dependency the profile lacks is a red chip and a dependency cycle is flagged on its rows.

## In-between moments

- **Dropping an archive:** a dashed drop zone fills the window and says where it goes. It shows no file name: WebKitGTK and Wails expose none before the drop.
- **A Nexus link:** on a game's profile screen the link installs into the open profile at once with a toast; on any other screen a dialog asks which profile, and further links queue behind it. A link never brings an open window to the front, so clicking downloads in browser tabs keeps the browser in front; only a window closed to the tray reopens, and a minimised window gets a desktop notification whose Show action raises it. A refused link (wrong game, another account, expired, malformed) is a toast saying why.
- **Two copies of one mod** (`frontend/src/mods/DuplicateDialog.tsx`): the dialog preselects the newer or Nexus-sourced copy and disables the other rather than deleting it; **Decide later** leaves both as a problem.
- **Loading:** a pane that waits on data shows the shared `SkeletonRows` in the shape of the content (announced by a label), never an empty pane.

## Console tab

The loader's log as it is written, monospace, in columns; warnings and errors get a tinted row. Code: `frontend/src/console/ConsoleTab.tsx`.

- **Log actions** (copy, save, search all runs, clear, share) are icon buttons in the tab header's page actions, not the filter row. **Share log** shows the size, says the log becomes public at a link, and asks before uploading to smapi.io/log.
- **Runs picker:** This session, or one of the profile's last 20 recorded launches; a past run loads its stored log read-only, and Save log and Share log use it.
- **Links:** in visible rows only, an exact installed mod name or `UniqueID` selects the mod, and a path under the profile's `mods/` folder or the game folder opens its folder.
- **Input line:** hidden until this profile is running and This session is shown. Enter sends the command through the Mortar SMAPI Bridge and echoes it into the log as a Mortar line prefixed `>`; the last 100 commands of that game are kept in memory only.

## Performance tab

Two summary cards head the page and switch it: **Startup** (time to the title screen or main menu, what mods add) and **In game** (saved per-profile measurements, compared in the tab, or a prompt to measure while the game runs). In game opens first while the game runs. Code: `frontend/src/console/PerformanceTab.tsx`.

- Up to three one-line findings from the selected run, each with one button; absent when none. Thresholds: `startupView.ts`.
- **Slowest mods:** a sortable table; the first six show until **Show all**, and an opened row always shows. Mods under 50 ms in both runs fold into **N other mods**, and the unattributed rest is one **Game and loader** row. Mortar's bridge row carries a **Mortar** label.
- In-game Start measuring and Show report are off while the game is not running this profile or a past run is shown. The source depends on the loader's `perf` capability:
  - **Console fallback** (no companion measurement): SMAPI's `performance enable` and `performance summary`.
  - **SMAPI with its bridge** (the same panel as BepInEx's; it replaces the console report): the bridge times each mod's handlers of the per-frame events; no measured launch is needed. The **Baseline** line is the frame less the mods' timed code, not a launch with no mods; Harmony patches and work outside events stay in it.
  - **BepInEx:** the bridge measures only on a launch measured from Startup, so the empty state says to use **Measure next launch** first. A row counts Harmony prefixes, postfixes and finalizers and the plugin's own Update, LateUpdate and FixedUpdate, exclusive of nested timed calls, not transpiled code or other threads.

## Profile management

A page, as Concrete's: every profile as a sortable row with colour and icon mark, summary, **Share** and a ⋯ menu of the shared profile items, with asides for saved groups and recently deleted profiles. Code: `frontend/src/profiles/ProfilesPage.tsx`, `frontend/src/share/`.

- **Damaged:** a profile folder whose `profile.json` cannot be read gets **Repair** (from the latest history snapshot: keeps `mods/`, moves the broken file aside as `profile.json.damaged-<timestamp>`, toasts Undo; off with a reason when no snapshot exists), **Open folder** and **Move to a folder…**. Damaged rows are omitted from the sidebar and every profile picker.
- **Recently deleted** is the 30-day trash ([architecture.md](architecture.md#profile-operations)). **Delete permanently** and **Empty trash** are error-coloured and confirm with **This cannot be undone.**; **Delete** on a profile says it stays restorable and toasts Undo.
- The switcher's context menu and the row ⋯ menu share the profile items. Sending, the Nexus collection draft and Thunderstore exports are reached from **Share…**. Send, Compare and Delete dialogs stay mounted so they survive the menu closing.
- **Check this profile** (`frontend/src/profiles/HealthDialog.tsx`) lists findings with a **Repair** each (none when Mortar cannot fix it) and **Repair all**. Mortar also runs the check quietly once a week per profile (and after recovering from a crash), never while the game runs and only after startup's first problem check; a finding shows an amber shield pill on the profile's sidebar and Profiles rows. It never toasts.
- **Templates:** applying previews, merges, queues missing downloads and toasts Undo. It records its own history event, after a baseline event when the newest event is not the current mods; Undo reverts to the baseline and restores the game settings and launch options the template replaced.
- **Saved groups** (a per-game library made from a group's **Save to library…**): adding one creates the group in the profile, a clashing name gets " (2)", " (3)", and one Undo reverts the whole add. Removing a mod from a saved group removes its optional files too; the last mod cannot be removed.

### Change history

The open profile's History (`frontend/src/profiles/HistoryDialog.tsx`) is a wide dialog: a timeline grouped by day with **Mark current as known good** restore points and **Compare two…**. **Undo these changes** takes the profile back to before the entry, so newer entries are undone too; **Revert this mod** undoes only that mod's change. The event count is capped by Settings (`historyEventsKept`); snapshot file bytes are not ([architecture.md](architecture.md#profile-operations)). A revert that cannot find store files names the mods and offers **Download missing mods**.

### Edit profile

A solid-paper dialog without a transition, as tall as its tallest tab. Code: `frontend/src/profiles/EditProfileDialog.tsx`.

- **Keep this profile's saves separate:** turning it on asks first and explains that while the profile runs the shared saves folder is set aside and returns when the game closes, with nothing deleted; turning it off applies at once.
- **Launch options:** Mortar sets `--mods-path` itself and refuses `--mods-path`, `--no-terminal` and `--skip-terminal`. **Launch prefix** and **Launch environment** apply to direct launches only (unavailable on Windows for the prefix); Steam launches receive neither ([architecture.md](architecture.md#launch)). Launch presets save at once, not with the dialog's Save.

### Compare, merge and share

- **Compare…** diffs this profile with another of the user's, a friend's link or file, or (Stardew) a multiplayer host's list. **Make <other> match <A>** applies only the checked rows to the target: mods only in the source and enabled-state differences are copied from the store, version and source differences switch the target's entry to the source's, and mods only in the target stay (no removal). Buttons are disabled with a tooltip while the game runs the target.
- **Add these mods to another profile…** (the merge) never changes the source and removes nothing from the target; added mods keep enabled state and config, and the target's History gets one undoable event. **For mods in both** keeps the target's versions or takes the newer.
- **Share** (`frontend/src/share/ShareDialog.tsx`) is two steps in a dialog of constant width: destination tiles, then the destination's pane. A share from the Mods selection limits link, file and mod list to those mods. The destination used last for the game takes focus with a **Last used** caption.
- A tile is greyed only when it cannot carry anything, and says why on the tile: with only local archives, Nexus Mods and Thunderstore read "Local archives cannot go here" while Mortar (a `.mortar` file includes them), Nearby computer and Mod list stay open.
- **Mortar pane:** **Link | File**; File is preselected over about 240 mods or when the link is too large. A link shows a meter of its length against Discord's 2,000 characters. The **Included** block lists each option in force; libraries and supporting mods are greyed out with the reason on a link, since a link has no room. Every share carries pins, groups and install reasons. A Discord mod list splits into messages of at most 2,000 characters, copied one at a time.
- **Nearby computer:** sending to an unpaired computer opens a links-only confirmation (config files stay here); a paired send copies store files first, then opens Import on the receiver ([architecture.md](architecture.md#sharing)).

## Import

A wide "Import profile from…" dialog over the dimmed game screen. Nothing downloads before the user confirms; closing leaves nothing behind. Archives go into an existing profile through Add archive or a drop, not Import. Bundled SMAPI and the bridge are omitted from every preview. Code: `frontend/src/share/ImportDialog.tsx`.

- **Preview:** mod tiles with an import state in place of a version (installed, download, dependency, check later, unavailable); an unticked mod is left out. Problems before download (removed from Nexus, broken for this game version, missing dependencies, one click per download on a free Nexus account) are compact items with their action.
- **Footer:** **New profile from link** creates the profile (named "<name> (2)" with the next free number when the name exists) and fills the download queue; **Replace <profile>** asks first, lists the entries it removes, keeps local-only mods and writes a history event so the change can be reverted.
- The link input reads the clipboard only from its **Paste** button; a `mortar://` link, a dropped `.mortar` file or a second launch opens the dialog filled in and previewed. When the import needs Nexus downloads and no account is signed in, a banner offers Nexus settings and the import buttons stay off.

## Sync offers

With a sync folder set (Settings › Mods and profiles), a change another computer made to a profile is announced once by an info toast with **Review**, which opens **Changes from your other machines** (`frontend/src/sync/SyncOffers.tsx`). When both sides changed a profile Mortar does not merge: the block reads **Both sides changed it, so Mortar will not merge them. Pick one side.** (**Use theirs** or **Keep mine**). Mod files are never part of a sync; mods an apply adds download from their sources.

## Download queue

A side sheet (`frontend/src/queue/`; Ctrl+J, the title bar pill or the palette). Skip, Cancel and dismiss are icon buttons (Retry on a failed row stays a labelled button). Closing the sheet keeps the queue running; history length is `internal/queue/history.go`.

- **Needs your click** (free accounts): the head item with Open download page and Skip; the next page opens when it finishes. An update or latest-file item is skipped on its own once the profile already holds that file or a newer one from the same page ([architecture.md](architecture.md#nexus-mods)). Premium accounts download without clicks.
- **Choose a file** (a GitHub release with several archives) and **Check this download** (a GitHub download whose mod SMAPI's update API does not tie to its repo: Install anyway or Skip) ask before installing.
- A download over 2 GB opens a warning-coloured confirm (`frontend/src/queue/LargeDownloadDialog.tsx`) before it starts: the mod's name, its size in GB, **Cancel** (focused, and what Esc does) and **Continue anyway**; several held downloads ask one after another, and the queue sheet shows the same choice as a card.
- **Download now** moves a waiting row to the front, pausing the last running download if every slot is busy. A failed row's button reads **Resume** when a partial file is on disk (else Retry); Cancel is offered only while downloading, not installing.

### Update review

Code: `frontend/src/mods/updateReview/`.

- Skipping or pinning is how a mod is left out of **Update all**; both apply even to updates Mortar cannot download. A row with a caution shows **Update despite this**; until ticked its Update is off and Update all leaves it out.
- A Nexus update with no file that supersedes the installed one (`pickFile`, [architecture.md](architecture.md#nexus-mods)) stays listed without Update, noting **Pick the file on Nexus**; Update all, Update before Play and the run-error **Update** skip it. Unofficial SMAPI versions are labelled and never included in Update all.
- Pinned mods never appear in the list or Update all; a collapsed **Kept at this version (N)** group shows each with **Unpin**, which re-checks updates. Implicit mods' updates come last under **Libraries (N)**.
- A warning triangle on a name means the update notes mention a breaking change or new requirement. After an update, **Update <mod> in other profiles** names pinned destinations and skips them.

## First run

Mortar's own setup finds launchers, not games, and opens until finished once (`launchersConfirmed`); there is no skip. Code: `frontend/src/firstrun/`, `frontend/src/firstrunTour/`. "Welcome to Mortar" lists each launcher Mortar reads on this OS as an accordion; a folder belongs to one launcher (Minigalaxy's install folder is not also GOG's). With none found the first row starts open, and detection runs again when the window regains focus. Settings › Launchers shows the same rows. **Continue** is always enabled.

Each game runs its own setup when first opened from Game Select while not set up (folder unknown or no profile): a non-interactive stepper **Game folder**, the game's loader step when it has one, **First profile**. **Choose folder…** falls back to a typed path field where there is no native folder dialog (server mode).
- **Loader** (if missing) starts by itself, unattended; a failure shows its error with **Retry** and **Skip for now**. On Windows with Steam, **One step in Steam** offers **Set it in Steam for me**, which fails with a toast while Steam is running.
- **First profile:** import from the game's `Mods` folder when it holds mods, start empty, import from another mod manager, or from a shared link (asks for the Nexus sign-in first when the import needs downloads).

## Confirmations

**Toasts** (`frontend/src/toasts/`): bottom right, stacked (at most three; the oldest goes first), a coloured edge by kind and a dismiss button; hover pauses the countdown, and errors and warnings stay longer. A toast about a mod shows its picture from the card cache. Toasts stay in the notification history.
- **Undo** is offered on reversible actions: a history revert, deleting a profile, removing a mod, dismissing a save warning, bulk edits from the Mods selection bar.
- **New downloads:** per **Offer new downloads**, the downloads folder raises "Add X to Y?" for one new archive and a **Review** toast for several; never while the game runs.

**Empty states** (`frontend/src/shell/EmptyState.tsx`): a tab with nothing to show centres an icon, a title, a line on how to fill it and, where one helps, a button. **First-open tips** (`frontend/src/tips/TipBanner.tsx`): the Mods tab, Saves tab, Console and Share dialog each show a short dismissible banner the first time they open; Settings › General has a **Tips** row to show them again.

## Settings

Mortar's Settings holds only what is Mortar-wide, never one game's (the one exception is Downloads' default profile for Nexus links, labelled with its game). Per-game settings are the [Game settings](#game-settings) page. Code: `frontend/src/settings/SettingsPage.tsx` and `sections/` (one file per page); the registry is `internal/settings/registry.go`.

A full page with a back button (Esc also returns), a section list on the left and the page beside it; while open, the title bar shows a "Settings" tab and no game tab. A search field filters rows by label and description across every page. Every control is a row (label and a wrapping description on the left, the control on the right) in its own rounded tile, filled rather than outlined. A choice whose options each carry an explanation renders as radio cards, two to four short options as a segmented strip, longer or dynamic lists as a dropdown. Lists and multi-step flows open dialogs rather than growing the page.

Rules a change could break:

- A setting another setting forces (Reduce motion holds Touch ripples off) shows disabled, its tooltip naming the setting to change first (`PrefKeys` `disabledBy`, `PrefRow` `disabledReason`).

- **General:** a flagged download is refused with **Install anyway**, which asks in an error-coloured confirm, records the override in the profile's History and installs unscanned; a scan that could not finish installs the mod and warns it was installed unscanned. **Settings file** import lists what would change by section and applies only the ticked ones ([architecture.md](architecture.md#settings-file)).
- **Mods and profiles:** the **Sync folder** holds profile settings, mod lists and configs, offered to other machines using it; mod files are not copied ([Sync offers](#sync-offers)).
- **Downloads:** **Handle "Mod Manager Download" links** confirms before taking them from another app.
- **Accounts:** Nexus **Untrack unused…** confirms that Nexus has no undo. GitHub uses the `gh` login when present, which raises the rate limit. The itch.io API key is kept in the system keyring and **Test** stores nothing on rejection.
- **Updates:** the background pass over every profile is fixed; the open profile's recheck while running is a setting (`prefSpecs.ts`).
- **Notifications:** a matrix of events by in Mortar (toast) and desktop (an OS notification sent whether or not the window is focused).
- **Storage:** **Move…** copies the data folder to a chosen empty location, verifies the copy, records `data-location` at the default path, removes the old copy and restarts; it is off with a reason while the game runs and refuses when space is short or the target is inside the current folder or not empty. Per-game save backup and profile history rows live on the game's settings pages.
- **Shortcuts:** click a keycap to record a combo (Esc cancels); a combo used by another row is refused, naming that action. Except Esc, shortcuts stay silent while typing or with a dialog open.
- **About:** **Open-source licences** lists every library, icon and artwork built into Mortar; things Mortar downloads at run time (SMAPI, BepInEx, the mod dataset) are not listed.

### Game settings

Per game, on their own page (`frontend/src/settings/GameSettingsPage.tsx`), opened by the last icon button in the game screen's tab row after **Share**. It reuses Settings' layout (nav with a back button to the game, search, Esc returns); the title bar keeps the game tab highlighted. Pages: **Install**, the game's loader, **Play**, **Mods**, **Save backups** (only for a game with saves), **Console** (only for a loader with a console) and **Streaming**. Anything tied to a game's loader, launcher, saves or content format lives here.

- **Scope:** with a profile open, a switch over the pages picks *All <game> profiles* or *This profile: <name>*. A profile's Home opens the profile scope, every other way in opens all profiles. In the profile scope a row a profile may override shows the profile's value, the game's beside it and **Use the game's**; any other row shows the game's value, held. The profile scope also shows what is only ever per profile: the loader's options in the profile's own files (BepInEx's console window and log level) on the loader's page, and the game's startup options (window mode, resolution, volume) on **Play**, each saved as it changes.

- **Install › Game folder:** **Change folder…** is checked before save. **Reset game install** (outlined, error colour) deletes the folder, SMAPI and the mods placed there while keeping Saves and Mortar profiles ([architecture.md](architecture.md#finding-the-game)).
- **SMAPI:** installing a pinned version asks first since every profile switches.
- **Mods › Preferred source:** an ordered list; when a mod is on several sites its card installs from the first.
- **Streaming › Stream overlay** (`frontend/src/settings/sections/StreamOverlay.tsx`): changes apply on the next Play. The bridge reports a `game` in `/state` that selects the OBS page's layout; each field has a live preview polled from `/state` and a copy icon for its OBS `file://` URL.

## Browser extension

The MV3 extension ([Rethunk-Tech/mortar-browser-extension](https://github.com/Rethunk-Tech/mortar-browser-extension), own repo and version) on `www.nexusmods.com`. When the two sides do not share a native-messaging protocol, its popup, button and filter section say **Update Mortar: it is too old for this browser extension** or **Update the browser extension: it is too old for this Mortar**. Native-host requests: [architecture.md](architecture.md#nexus-mods).

- **Popup:** a mode (**Off (do not mark Nexus pages)**, **Highlight** or **Hide mods in profile**, stored in `chrome.storage.local` as `mode`) and a status line naming the connection state.
- **Listings, search and collections:** in Highlight, mod tiles whose Nexus id is in the open profile get an **In profile** badge, or **Update available** when Mortar knows of one; Hide removes those tiles; Off draws nothing, the Mortar filter section included. The app's **Browser extension connection: Off** has the same effect everywhere. Mortar's UI appears only on games the native host lists as supported. A disabled row always shows its reason as text.
- **Listing filters:** a Mortar section at the top of Nexus's filter sidebar with three rows that dim tiles in place (the listing keeps its length), saved per game: **Gray out mods in profile**, **Gray out obsolete** (the same words and rule as the Problems check's author-marked mods, `authorStatusWord` in `internal/problems/author_marked.go`, pinned by the extension's tests; needs no connection to Mortar) and **Gray out broken** (SMAPI's compatibility list for the version last played; needs Mortar installed but not running).
- **Mod page:** a read-only panel under the title showing the profile state, other profiles holding the mod, update availability and **Open in Mortar** (`mortar://<game>/mod/<id>`). Off removes the panel.
