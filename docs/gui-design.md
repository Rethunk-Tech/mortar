# GUI design

The design system lives in code: `frontend/src/theme/` (palette, density, the MUI theme), `frontend/src/shell/` (dialogs in `shell/dialog/`, menu rows in `shell/MenuAction.tsx`, `TitleMenu.tsx`'s headings and rules, `Tooltip.tsx`, `EmptyState.tsx`) and `frontend/src/brand/`. `frontend/src/lint/patterns.test.ts` and `bun run --cwd frontend i18n:dupes` fail a change that builds around them; `frontend/e2e/deck.pw.ts` checks the Steam Deck size, controller reach and 44px targets. Read those before any UI change. This page holds only the rules code cannot enforce.

## Conventions

- Dark is the lead theme: screenshots, the README and store listings show it.
- Dialog footers: Cancel or Close is a text button, leftmost; at most one contained primary, rightmost; a destructive primary is contained `color="error"`.
- Spacing uses the `space` roles (`theme/density.ts`) and no hand-set pixels, so a Density change moves every page.
- A trailing "…" only on a control that opens further input (a dialog, a picker, a confirmation); clicking it never starts the work, and the dialog's title has none.
- **Remove** takes something out of a profile, group or list; **Delete** destroys it on disk. The button, its confirmation and its toast use the same verb.
- Sentence case everywhere; labels, buttons, headings and toast titles take no full stop, body text does. No article where the meaning holds without it. A label reused mid-sentence is lowercased in code (`i18n/lowerFirst.ts`).
- One placeholder name per concept (`{name}`, `{count}`, `{profile}`, `{game}`, `{version}`, `{path}`, `{when}`, `{source}`, `{manager}`, `{loader}`, `{launcher}`, `{os}`, `{app}`); a plural holds only the counted clause.
- **Command palette** (Ctrl+K): a shortcut that does what an action does shows its keys on that action and gets no second row; an action needing an open profile is greyed with a reason.

## Game Select

Choose or set up a game; shown at start and from the title bar. Code: `frontend/src/games/GameSelect.tsx`.
- Art is Steam's `library_hero.jpg` read at runtime and not bundled, under a light solid dim layer with a text shadow and no panel. A row never changes height or moves its art on hover or focus, no tile has an edge stripe, and a tile that cannot open is desaturated under a darker dim.

## Main screen

A left sidebar and a detail pane. Code: `frontend/src/game/MainScreen.tsx`; Play is `frontend/src/launch/PlayControl.tsx`, Home `frontend/src/game/Home.tsx`.
- Below 960px the sidebar is always the icon rail, the hero a one-line header, the mods search folds into a filter icon, other actions into icon buttons, and the mod grid drops to two columns.
- The hero bleeds to the pane's edges, carries no actions beside the name, and its cover sits under a light solid dim layer that fades to transparent over the bottom 20% by an alpha mask.
- The launch screen (`frontend/src/launch/LaunchScreen.tsx`) covers the game page, opaque, from Play until the loader says the mods have loaded, with the game's hero art under the art-dim set. The slow notice and a failure cover the log box instead of adding height, each with one contained action (**Keep waiting**, **Retry**). **Cancel** ends only the game's own processes and leaves the launcher running.

## Mods tab

The profile's mods as a **Grid** or a **List** (a table), both virtualized, sharing one details panel. Code: `frontend/src/mods/ModsTab.tsx`.
- The last loaded list for a profile stays on screen when the tab is left and returns, refreshed in the background.
- List cells never wrap (ellipsis, full text in `title`); figures come from cached Nexus details or the Thunderstore listing, so opening the list makes no request per mod.
- The endorsement prompt is a snackbar after clean launches of an enabled Nexus mod the user has not endorsed or abstained, when Settings › Accounts asks for it.

## Mod detail

The details panel both Mods views share. Code: `frontend/src/mods/DetailsAside.tsx`.
- It lies over the right edge of the list, so opening it never changes the list's width or columns; its left edge resizes it by dragging or with the arrow keys.

## Browse tab

Finds mods to add to the open profile; the shared header has no page actions here. Code: `frontend/src/browse/BrowsePage.tsx`.
- In the grid a card's action sits under the picture with no open-page button (the page opens from its menu and details); list rows keep an open-page icon.
- The footer bar sits under the results, outside the scroll area, with the result count and what is missing from it.

## Develop tab

Creates and builds mods from the loader's template, in its own sidebar group (**Create**). Code: `frontend/src/develop/`.
- A project card's row of actions never wraps a label; the last build's outcome shows as a compact Alert below, each compiler error a link to its file and line.

## Problems tab

The profile's problems by section. Code: `frontend/src/mods/ProblemsTab.tsx`, `problemGroups.ts`, `problemFix/`; each kind's explanation is `problemWhy.ts`, backed by [user-guide.md](user-guide.md).
- **Find the mod that crashes the game…** is enabled only when the newest run crashed with no suspected mod, otherwise off with the reason as its tooltip; the palette entry follows the same rule.
- A row's main fix is filled in the warning colour on real conflicts and outlined on cosmetic ones.
- Applying a Content Patcher **Set to** suggestion remembers that field value, so the reverse hint stays hidden until the setting changes.

## In-between moments

Dropping an archive, Nexus links, two copies of one mod and loading. Code: `frontend/src/install/DropOverlay.tsx`, `frontend/src/nxm/`, `frontend/src/mods/DuplicateDialog.tsx`.
- A Nexus link never brings an open window to the front, so clicking downloads in browser tabs keeps the browser in front; only a window closed to the tray reopens, and a minimised window gets a desktop notification whose Show action raises it.

## Performance tab

Two summary cards head the page and switch it: **Startup** and **In game**. Code: `frontend/src/console/PerformanceTab.tsx`, thresholds in `startupView.ts`.
- The in-game source follows the loader's `perf` capability:
  - Console fallback: SMAPI's `performance` commands.
  - SMAPI with its bridge: times each mod's per-frame event handlers; **Baseline** is the frame less the mods' timed code (Harmony patches and work outside events stay in it).
  - BepInEx: measures only on a launch measured from Startup, counting Harmony prefixes, postfixes and finalizers and the plugin's Update, LateUpdate and FixedUpdate, exclusive of nested timed calls, not transpiled code or other threads.

## Profile management

Every profile as a sortable row with colour and icon mark, summary, **Share** and a ⋯ menu of the shared profile items, with asides for saved groups and recently deleted profiles. Code: `frontend/src/profiles/ProfilesPage.tsx` (History, Edit profile, Compare and Merge dialogs beside it), `frontend/src/share/`.
- The switcher's context menu and the row ⋯ menu share the profile items; Send, Compare and Delete dialogs stay mounted so they outlive the menu.
- Applying a template previews, merges, queues missing downloads and toasts Undo; its history event follows a baseline event when the newest is not the current mods, and Undo reverts to the baseline, restoring the game settings and launch options it replaced.
- History keeps `historyEventsKept` events; snapshot file bytes are not capped.
- Launch prefix and environment apply to direct launches only (no prefix on Windows).
- Compare's buttons are disabled with a tooltip while the game runs the target.

## Import

A wide "Import profile from…" dialog over the dimmed game screen. Code: `frontend/src/share/ImportDialog.tsx`.
- **Replace <profile>** asks first, lists the entries it removes, keeps local-only mods and writes a history event.

## Download queue

A side sheet (Ctrl+J, the title bar pill or the palette). Code: `frontend/src/queue/`; update review in `frontend/src/mods/updateReview/`.
- **Needs your click** (free Nexus accounts): the head item shows Open download page and Skip, and the next page opens when it finishes; an update or latest-file item skips itself once the profile holds that file or a newer one from the same page. Premium downloads without clicks.
- **Choose a file** (a GitHub release with several archives) and **Check this download** (a GitHub download SMAPI's update API does not tie to its repo) ask before installing.
- In update review, a warning triangle on a name means the update notes mention a breaking change or new requirement.

## First run

Mortar's own setup finds launchers, not games, and runs until finished once (`launchersConfirmed`); each game runs its own setup when first opened while not set up. Code: `frontend/src/firstrun/`, `frontend/src/firstrunTour/`.
- The loader step starts by itself, unattended; a failure shows its error with **Retry** and **Skip for now**. On Windows with Steam, **Set it in Steam for me** fails with a toast while Steam is running.

## Settings

Only what is Mortar-wide, with one exception: Downloads' default profile for Nexus links, labelled with its game. Code: `frontend/src/settings/SettingsPage.tsx` and `sections/`; the registry is `internal/settings/registry.go`. Per-game settings are their own page (`GameSettingsPage.tsx`), opened by the **<game> settings** button on Home (`frontend/src/game/Home.tsx`) or the command palette.
- **Install anyway** on a flagged download asks in an error-coloured confirm. **Settings file** import lists changes by section and applies only the ticked ones.
- The itch.io API key is kept in the system keyring; **Test** stores nothing on rejection.
- In a game's profile scope, a row a profile may override looks like the game's until the profile changes it, then shows an accent dot after its label and a reset button naming the game's value; other rows show the game's value, held. Values only ever per profile (the loader's options in the profile's files, the game's startup options) show only in that scope.
- **Reset game install** deletes the folder, SMAPI and the mods placed there, keeping Saves and Mortar profiles.
- **Preferred source** is an ordered list; a mod on several sites installs from the first.
- **Stream overlay** changes apply on the next Play; the bridge's `game` in `/state` selects the OBS page's layout.

## Browser extension

The MV3 extension ([Rethunk-Tech/mortar-browser-extension](https://github.com/Rethunk-Tech/mortar-browser-extension), own repo and version) runs on `www.nexusmods.com`; its UI rules live in that repo. When the two sides share no native-messaging protocol (`internal/nativehost` `Protocol`, `MinProtocol`, `MaxProtocol`), it says which side to update. Native-host requests: [architecture.md](architecture.md#nexus-mods).

## Where other screens live

| Screen | Code |
| --- | --- |
| Tab header (every tab but Home: a strip tinted from the profile's colour, its slot filled through `<PageActions>`) | `frontend/src/game/TabHeader.tsx` |
| Config page (typed rows, [architecture.md](architecture.md#profile-operations)) | `frontend/src/mods/typedConfig/` |
| Saves tab | `frontend/src/saves/SavesTab.tsx` |
| Load order tab (read-only; `mortar profile load-order` prints the same list) | `frontend/src/mods/LoadOrderTab.tsx` |
| Console tab | `frontend/src/console/ConsoleTab.tsx` |
| Sync offers | `frontend/src/sync/SyncOffers.tsx` |
| Toasts, empty states, first-open tips | `frontend/src/toasts/`, `frontend/src/shell/EmptyState.tsx`, `frontend/src/tips/TipBanner.tsx` |
