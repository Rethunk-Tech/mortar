# Mortar: decided, not yet built

This file holds only work that is decided but not built. Each item is written to be implemented cold: the shape, the evidence it rests on, the traps, and when it is done. When an item lands, delete it here and move any fact that stays true to [architecture.md](architecture.md); the code is the record of what exists. Screens and styling: [gui-design.md](gui-design.md). Lethal Company research for the second game: [lethal-company.md](lethal-company.md). Standing rules: [AGENTS.md](../AGENTS.md).

Specs under [Decided, ready to build](#decided-ready-to-build) have the operator's answers recorded in them; a spec under [Specs awaiting approval](#specs-awaiting-approval) ends in open questions, and nothing in it is built until they are answered.

Items marked **Measure** need a throwaway test first; those tests run outside this repo and only their results land here.

## Release

Remaining ([architecture.md](architecture.md#release)):

- The fork's fixes are offered upstream as wailsapp/wails#6200 (EXDEV staging), #6201 (AppImage), #6202 (OnUpdateApplied, a draft waiting on its WEP, #6203) and #6239 (service methods fall back to `Options.MarshalError`, which `errorkind_test.go` relies on). With #6197 (GTK4 transparency) all six are open and rebased on v3.0.0-beta.28, and Mortar pins the fork branch `mortar/v3.0.0-beta.28`. Once #6200, #6201, #6202 and #6239 ship in a tagged v3 beta, pin that beta and drop the `Rethunk-AI/wails` replace in `go.mod`; #6197 does not gate this, since it only serves the translucent window below. #6201 and #6202 both add `resolveTarget` in `v3/pkg/updater/spawn.go`, so whichever merges second conflicts and needs a rebase.
- **NOMAD-only, needs Windows Steam with Stardew and SMAPI** (the test VM has neither): with `"<game>\StardewModdingAPI.exe" %command%` set, Play once from a Windows account whose user name has a space, then read the first lines of `%APPDATA%\StardewValley\ErrorLogs\SMAPI-latest.txt`. Done when the log's mods path is the profile's `mods` folder (Steam forwarded `-applaunch … --mods-path <dir>` intact and kept the spaced path as one argument); if it is the game's `Mods` folder or a cut-off path, the Steam path needs a different hand-off ([architecture.md](architecture.md#launch) Windows). Whether a direct launch with **SMAPI console window** on shows SMAPI's output, given that Mortar redirects the game's stdout to its log file, is checked in the same sitting.

## Queued for v1

Decided 2026-10-07 (NOMAD), from the audit of how Mortar was described to CurseForge.

- **Profiles for any game, and shared-state games (The Sims 4 first).** Spec: [Generic-folder games](#generic-folder-games-and-shared-state), decided 2026-10-09 apart from which game proves the generic path next.

## Decided, ready to build

Decided 2026-10-09 (NOMAD). Facts about Mortar are anchored to the code as it is today; facts about EA, Patreon and The Sims 4 are marked **Verify** where they come from memory of the vendors' behaviour and need one throwaway check (run outside this repo, result recorded here) before the item is built.

### EA App game store

**Why.** Several wanted games (The Sims 4 first) are sold on EA's store, and the EA App is a store Mortar cannot see: `gamestore.All()` knows Steam, GOG, Lutris and Bottles only (`internal/gamestore/gamestore.go:61`). Steam sells The Sims 4 too, but a player who bought it from EA has no Steam install to find, and Mortar's "game not found" screen is the only outcome today.

**Decided shape.** One more `Store` driver, key `ea`, found the way GOG's offline installs are: by folders and a marker, never by reading the EA App's private state.

- **Discovery is folder based.** The driver searches library roots for `<root>/<stores.ea.folder>` holding the game's `marker` (`internal/components/components.go` `GameInfo.Marker`, matched by the same check the other stores use). Roots, in order: the user's added folders (`settings.LauncherRoots["ea"]`, `internal/settings/settings.go:63`), then the EA App's usual library folders for the platform (Windows: `%ProgramFiles%\EA Games`, `%ProgramFiles(x86)%\EA Games`, and `Electronic Arts` beside them; Linux: none of its own). No registry read and no parsing of the EA App's data in `%ProgramData%\EA Desktop` (undocumented and rewritten by EA's updates). **Verify** the default folders and that The Sims 4's install holds `Game/Bin/TS4_x64.exe` (the catalog marker is a file at the root or one `game` folder down, so the marker value must be checked against a real install, case included).
- **Linux has no native EA App.** An EA App install on Linux lives inside a Wine prefix, so it reaches Mortar through the two stores that already read prefixes: Bottles (`bottleGameDirs`, `internal/gamestore/bottles.go:27`, gains `Program Files/EA Games/<folder>` and `Program Files (x86)/EA Games/<folder>` under the bottle's `drive_c`, so `stores.bottles.folder` serves EA installs as it serves Steam and GOG ones) and Lutris (its `stores.lutris` slug and keyword already find a game by its Lutris entry). This means no new runtime: an EA game in a bottle is the existing `wine-prefix` runtime (`internal/runtime/wineprefix.go:18`), which resolves `{documents}`, `{appData}` and the rest inside the bottle.
- **Launch is a direct start.** `Starter.command` runs the game's executable for every store except Steam (`internal/game/start.go:131`, the `env.Direct || plan.Exe != ""` branch). An EA game started this way must ask the running EA App to authenticate; the executable does that itself, so no EA-specific branch is added. A relay through `origin2://` or `link2ea://` is not built (see open questions).

**Catalog change.**

- `GameStores` gains `EA *EAStore` with `folder` (`internal/components/components.go:217`), mirroring `BottlesStore` (`:225`). `GameInfo.Validate` adds `stores.ea.folder` to the unsafe-name loop (`:374`).
- `internal/gamestore`: `eaKey`, `StoreEA`/`LauncherEA` constants (`gamestore.go:13`, `:24`), `eaStore{}` in `drivers.go`, an entry in `All()` (`:61`), a case in `Has` (`:64`), `storeOrder` (`:97`; rank after GOG so a Steam install of the same game wins as default) and `launcherOrder` (`:103`). `LauncherOf` (`:118`) needs no case.
- `Launchers(goos)` returns one spec, id `ea`, name "EA App", `Usable` true for any existing folder; on Linux it returns none, so the setup screen shows no empty EA row there (Bottles and Lutris installs report under their own launchers).
- Frontend: `frontend/src/games/storeName.ts:4` (a name for the new store id) and `frontend/src/brand/launchers/LauncherLogo.tsx:8` (a tile; simple-icons has an EA mark: **Verify**, else a plain tile as Bottles has). Lingui catalogs re-extracted.
- `runtime.Install.Store` needs nothing: the `native` runtime claims a Windows build on Windows, and `wine-prefix` claims Bottles installs only (`internal/runtime/wineprefix.go:23`).
- `HUMANS.md` § Drivers step 1 and `docs/architecture.md` § Games and the catalog (Stores bullet, the store table at the "How it is found" section) list the new store.

**Traps.**

- `Discover` results are de-duplicated by folder (`gamestore.go:81`), but one game on Steam and EA at once gives two installs with different ids; `game.pick` (`internal/game/install.go:132`) chooses by `Rank`, so `storeOrder` decides the default. Do not make EA first.
- The marker check must tolerate EA's `Game` subfolder; a catalog marker that works for the Steam copy may not for the EA copy if EA's layout differs. Check both layouts of The Sims 4 before enabling.
- `steamSkipsLoader` and the Steam `-applaunch` branch are gated on `inst.Store`; an EA install must never reach them (`internal/game/start.go:121`, `:166`).
- `scripts/selftest.sh` builds a minimal Steam library only; an EA game needs a fixture folder under a fake `EA Games` root and `launcherRoots` set in the sandbox settings.
- The Flatpak build of Mortar sandboxes folder reads; a user-added EA folder needs the same grant text as other added folders (`game.ValidateLauncherRoot`, `internal/game/launchers.go:75`).

**Acceptance.**

- `go test ./internal/gamestore ./internal/game ./internal/components ./internal/source/all` pass; `TestEveryCatalogReferenceResolves` (`internal/source/all/catalog_test.go`) accepts `stores.ea`.
- A test in `internal/gamestore` finds a fixture install under `EA Games/<folder>` and under a Bottles bottle, and ranks it after Steam.
- A catalog-only test game (the pattern of `TestACatalogOnlyGameNeedsNoCode`, `internal/game`) with only `stores.ea` is selectable, found, and started through the direct branch with no Steam.
- Sandbox: Settings lists EA App with the added folder, the game's install appears, and a hidden-display launch against the copy starts the stub executable (never the maintainer's screen).

**Decisions (NOMAD, 2026-10-09).**

1. Folders only: the EA App's default library folders plus the user's added folders; the EA App's install list is never read.
2. The game's executable starts directly; no `origin2://` or `link2ea://` relay and no per-game offer id in the catalog.
3. Windows first. On Linux an EA game reaches Mortar through Bottles and Lutris, with no Heroic-style launcher for EA in the first release.
### Generic-folder games and shared-state

**Why.** Today a game is a catalog entry plus a loader (SMAPI, BepInEx), and `GameInfo.Validate` rejects an entry with none (`internal/components/components.go:358`). Most games have no loader: their mods are files dropped into a folder. And a game such as The Sims 4 keeps its mods and its saves in one shared Documents tree (`{documents}/Electronic Arts/The Sims 4`), so two profiles cannot be told apart by which install they use: the files that decide what the game loads are outside the install and shared by every launch.

**Decided shape.** Three small additions, each using what exists.

1. **A loaderless game.** A loader driver `folder` (`internal/loader/folder/`, registered in `internal/loader/all/all.go`; `loader.Register`, `internal/loader/registry.go:16`) that implements `Loader` with nothing to install: `Status` reports installed, not broken, not per-profile, no latest; `Install` returns an error "nothing to install"; `Contribute` adds the profile's content as plan files (below). The catalog names it like any loader (`loaders: [{"id": "folder", "name": "Mod folder"}]`), so `Validate` keeps its "needs a loader" rule untouched (`components.go:358`), `loader.For` (`registry.go:46`) and `game.PrimaryLoader` need no special case, and `TestEveryCatalogReferenceResolves` accepts it once `knownLoaders` lists `folder`. No new `deploy` method name: the game uses `profile` (`components.go:402`).
2. **The mod folder as a path role.** The catalog's `paths` map gains the role `mods`: where the game reads mods, outside or inside the install (`{documents}/Electronic Arts/The Sims 4/Mods`, or `{install}/Mods`). `game.PathFor` already resolves any role for an install (`internal/game/capabilities.go:37`); `GameInfo.Validate`'s token-start rule applies as is (`components.go:397`). `launchplan.PlanFile` (`internal/launchplan/launchplan.go:31`) gains a `Root` field naming a path role (empty means the install, as today); the `copy-into-install` deployer (`internal/deploy/place.go:32`) resolves `Dst` under that root instead of the install folder and keeps its refusal of a path that leaves it (`filepath.IsLocal`). Journal, displaced-file restore and crash recovery then work unchanged for files outside the install, because they key on absolute paths.
3. **Profile-isolated shared state.** A launch swaps the profile's content into the shared folder and takes it back, using the machinery that exists: the loader's plan files place the profile's enabled mods into the game's `mods` role folder for the length of the launch (journaled under `<data>/journal/<install id>`, `docs/architecture.md` Pipeline bullet), and the saves for a profile with `separateSaves` use `internal/savesiso` against the `saves` role. The two roles can sit in the same parent folder (The Sims 4's `Mods` and `saves` are siblings); each is swapped on its own, and nothing else in that tree (Options.ini, Tray, screenshots) is touched.

**Mod content and installs.** Targets already carry per-extension depth (`TargetDef.MaxDepth`, `components.go`; The Sims 4: **Verify** `.package` allowed several folders deep and `.ts4script` at most one folder deep); the `plain` installer maps any archive into targets (`internal/installer/plain.go:17`). The profile target is `{"id": "mods", "root": "{profileMods}", "maxDepth": {...}}`. `profile.Store.SyncPackages` (`internal/profile/deploy.go:86`) lays out only packages whose loader holds them; a folder-loader game's entries are archives installed into the profile's mods folder, so the folder loader's `Contribute` walks `{profileMods}` (the files of enabled entries) and emits one plan file per file. Enabled/disabled stays the profile's state, not a file move.

**Catalog and code changes.**

- `components.go`: no new `GameInfo` field. Docs list `mods` among the path roles (HUMANS.md § The catalog entry, `paths`).
- `internal/launchplan/launchplan.go:31` `PlanFile.Root`; `internal/deploy/place.go:32` resolve the root through a `View` callback (`deploy.View` already carries what the deployer needs; add `PathFor(role)`), `internal/launchsvc/pipeline.go:192` `startDeploy` passes it from `game.PathFor`.
- `internal/loader/folder` (new), `internal/loader/all/all.go`, the `knownLoaders` list (`internal/source/all/catalog_test.go:16`).
- `internal/game/capabilities.go`: a `PathMods = "mods"` beside `PathSaves`.
- The Sims 4 entry itself is catalog data: `stores.steam.appId` (**Verify** the id, it is the game's Steam app id), `stores.ea.folder` once the EA store lands, `marker`, `paths.mods`, `paths.saves`, `targets`, `sources` (CurseForge and GitHub shape; `Gated` CurseForge needs its key), `deploy: "profile"`, `loaders: [{"id": "folder"}]`, `enabled: false` until checked.
- `scripts/selftest.sh`: a fake Documents tree (`HOME` is already sandboxed) and a stub game; the Sims 4 is not copied into the sandbox, a stub executable is.

**Traps.**

- **The game's own switches.** The Sims 4 loads mods only when `Options.ini` has custom content and script mods enabled, and a patch can switch them off again; a folder swap cannot make the game read mods. Mortar shows a finding in Problems (the check pattern in `internal/problems`) and never edits `Options.ini` (decided below). **Verify** the file name and keys.
- **`Resource.cfg`.** The Sims 4's Mods folder needs a `Resource.cfg` that EA ships there; a swap that removes the folder's contents must keep it, or the game ignores subfolders. The folder loader treats files in the target root that no profile entry owns as the player's own and never displaces or removes them (the deployer's existing rule: only files Mortar placed are removed, `docs/architecture.md` Purge bullet); the shipped `Resource.cfg` is such a file and is left alone.
- **Windows file locking and cloud sync.** Documents may be under OneDrive; a swap that creates and removes thousands of files per launch is slow and syncs noise. Measure a 3,000-file Mods folder before enabling (**Measure**, throwaway test outside this repo: place and purge time, and the sync client's effect).
- **A running game.** The deployer already refuses to start while a journal exists (`ErrUnrecovered`) and recovery skips an install whose game still runs (`launchsvc.RecoverDeploys`, `internal/launchsvc/pipeline.go:248`); the `GameProcesses` of a catalog-only game come from `catalogOnly` (`internal/game/catalog_only.go:36`), so the entry needs the process names there or in the catalog.
- **No mod identity.** SMAPI mods have a UniqueID and BepInEx ones a Thunderstore package; a folder-loader game's mod has neither, so its identity is the source id recorded at install (a CurseForge project, a GitHub repository) or none, and update checks exist only for entries with a source. Problems checks that read manifests (`internal/manifest`) do nothing for these games; do not add generic "broken mod" guesses.
- **Shared state beyond mods and saves.** Anything else the game writes to the shared tree (the Sims 4 `Tray` folder of household and lot exports, screenshots, `Options.ini`) stays shared by every profile; only `mods` and `saves` are isolated.
- A game on Steam Cloud syncs its saves folder; `separateSaves` already documents that cost (`docs/architecture.md` Saves).
- The full unit and e2e budget (10 s warm, `AGENTS.md` Verify) applies: the folder loader's tests use a temporary tree, not a copy of a real game.

**Acceptance.**

- A test catalog entry with only `loaders: [{"id": "folder"}]`, `paths.mods`, a `profile` target and a Steam or EA store is selectable and found (the pattern of `TestACatalogOnlyGameNeedsNoCode`), a profile with two mods plus a third disabled places exactly the two enabled mods' files under the `mods` role folder, and Purge returns the folder to its previous bytes (hash before and after), including a file the player already had at one of the same paths (displaced and restored).
- A crash between place and purge (the existing `crashsafe` test pattern, `internal/deploy`) recovers at the next start with the folder restored, for a root outside the install.
- Two profiles of one game with different mods and `separateSaves` each launch with their own mods and saves, and a launch of one leaves nothing of the other in the shared tree.
- Problems shows the "custom content is off" finding when the fixture `Options.ini` has it off, and a test asserts the file's bytes are unchanged after the check.
- `go test ./internal/loader/... ./internal/deploy ./internal/launchplan ./internal/launchsvc ./internal/components ./internal/source/all ./internal/game` pass; sandbox regress with a stub game shows the swap on a hidden display.

**Decisions (NOMAD, 2026-10-09).**

1. Isolation covers `Mods` and saves only. `Tray`, `Options.ini` and the rest of the shared tree stay shared by every profile.
2. Problems reports the `Options.ini` mod settings (custom content and script mods off) and never edits the file.
3. A mod with no source and no manifest is added from a file and is never updated by Mortar.
4. Catalog-only: no user-defined games; every game is a catalog entry, as today.

**Open question for NOMAD.**

5. Which game after The Sims 4 should prove the generic path (any Steam game whose mods are plain files in a folder under its install)?
### Patreon mod source (decided: link and handoff, post URLs only)

**Why.** Some modders ship only to patrons. Mortar has no source for them, so those mods enter a profile only as files added by hand.

**Evidence, and what that forces.** Everything downloadable from a Patreon post sits behind the patron's session; Patreon's public API (v2, OAuth) serves a *creator's* own campaign data to the creator's token, and for a patron it offers identity and memberships, not post attachments. **Verify, researched 2026-10-09 from Patreon's documentation (no live probe run yet).**

- The public API v2 scopes are `identity`, `identity[email]`, `identity.memberships`, `campaigns`, `campaigns.members`, `campaigns.members[email]`, `campaigns.members.address`, `campaigns.posts`, `campaigns.lives` and two write scopes ([OAuth, Scopes](https://docs.patreon.com/#scopes)). `campaigns.posts` is described as "read access to the posts on a campaign", and `GET /api/oauth2/v2/campaigns/{campaign_id}/posts` requires it ([Resource endpoints](https://docs.patreon.com/#api-endpoints)). The docs say a client creator's token gets all v2 scopes automatically and never state that a patron can be granted a scope that reads another creator's campaign. The docs' own summary of the access model is "You may only fetch your own list of pledges or public pledges", and an app that manages many creators' campaigns must contact Patreon.
- The documented post resource has `embed_url` and `embed_data` but no attachment or file field, and no endpoint lists a post's attachments. The Media resource (`download_url`, valid 24 hours) is documented only as linked to tiers.
- Rate limits ([Rate limits](https://docs.patreon.com/#rate-limits)): 100 requests per 2 seconds per client, 100 per minute per access token, 429 on excess, and a 30 minute API block after more than 2,000 4xx responses in 10 minutes. A `User-Agent` header is required or calls may get a 403.
- Terms ([Patreon Terms of Use](https://www.patreon.com/policy/legal)): no clause names scraping or bots, but the terms bar abusing Patreon "in a technical way" or "in an unintended manner". Patrons get a licence to view creations for "private, personal, non-promotional, non-commercial use", may not use creations "in any way not authorized by the creator", and may not share them "with others who have not purchased" them. Automated downloading of patron files by an app is therefore not an authorised use the terms describe.
- Conclusion for the shape below: nothing found supports a patron-side post or attachment API, so no Searcher and no Mortar-side download. The operator skipped the live probe (a patron OAuth token against `campaigns/{id}/posts`); the shape stands on the documentation alone, and a patron-side API turning up later reopens search only by a new decision.

Scraping Patreon's internal `/api/posts` or driving the user's browser session is out: it breaks without notice and acts as the user's login, which the standing rules refuse.

**Decided shape.** Patreon is a **link and handoff source**, with no search and no Mortar-side download.

- Driver `internal/source/patreon` (`ID` `patreon`, `Name` "Patreon", `Modes` `[]source.Acquire{source.Handoff}` as `internal/source/curseforge/curseforge.go:72` lists both), registered with `source.Register` (`internal/source/source.go:308`) and imported in `internal/source/all/all.go`. It implements `Hoster` (`Hosts` `patreon.com`; `source.go:239`) so a pasted post URL traces back to the source, and `PageLinker` (`ModPageURL(gameKey, id)` returns `https://www.patreon.com/posts/<id>`; `source.go:234`). It is not a `Searcher`, so `source.Searchable` (`source.go:350`) leaves it out of Browse and an `all` search; it is not `Gated`.
- **Entry shape.** A mod from Patreon is a profile entry of kind `patreon` whose name is the numeric post id (the number ending a `patreon.com/posts/<slug>-<id>` URL). Adding one is: paste a post URL in Add mod (the existing URL path finds the source through `source.NameOfHost`, `source.go:378`), Mortar opens the post in the browser, and the file the user saves from it is picked up by the existing Downloads watcher (`internal/dlwatch`, `internal/folderwatch`, [architecture.md](architecture.md#nexus-mods) "Manual downloads") and offered for install; installing it records the post id on the entry, so the profile menu's mod page link and a share link both carry the id. There is no update check (no API to ask); a mod page link is the only update path.
- **Gated posts and the no-re-hosting rule.** Mortar never fetches a patron-only file. It never holds a Patreon session cookie or password, never forwards the user's login, and never writes a Patreon file into a share, a bundle or the LAN transfer except as the user's own computers pair (the one exception in [AGENTS.md](../AGENTS.md) § Decided: a profile sent between the user's own paired computers carries its mod files). A share link to a profile holding a Patreon entry carries the post id and name only, so the receiver gets a link to the post and downloads it if they are a patron.
- No OAuth, no keyring entry, no account in settings in this version.

**Touches.**

- `internal/source/patreon/` (new): `patreon.go`, `link.go` (post URL parser, accepts `patreon.com` and `www.patreon.com` posts of the two forms `/posts/<slug>-<id>` and `/posts/<id>`, numeric id only), `fuzz_test.go`.
- `internal/source/all/all.go:4`: one blank import.
- `internal/profile`: a `KindPatreon` beside `KindNexus` and its source struct, and the exhaustive switches over entry kinds (`rg "KindNexus" internal` lists them; each needs a patreon case or the build's exhaustive lint fails). `profile.Source` carries kind and name already.
- `internal/browse/installed.go`: the in-profile matcher (`Hold`) needs no Patreon id because Browse never lists the source; a pasted URL for a post already in the profile is detected by entry name.
- `internal/archivesvc` `InstallDownload` records `profile.KindPatreon` with the post id when the add came from a Patreon link.
- Catalog: a game lists `{"id": "patreon"}` in `sources`; `TestEveryCatalogReferenceResolves` then requires the driver. `GameInfo.Validate` already requires a unique id (`components.go:420`).
- `docs/security.md`: a row under Network and links for the post-URL parser (threat: a crafted link spoofs another site or injects a path; guard: the exact host and shape check; test: `FuzzParsePostURL`), and the "Links handed to the system handler" row's allow-list (the post URL opens in the browser).
- Frontend: a source chip is not needed (Browse never lists it); Add mod's URL box recognises the host through the existing source lookup. A Patreon icon for the mod page link and the entry row.

**Traps.**

- The pasted URL is untrusted input and goes to the system URL handler: accept `https` only, host exactly `patreon.com` or `www.patreon.com`, no userinfo, and rebuild the opened URL from the parsed id rather than opening the pasted text.
- `source.Searchable` and `ForGame` both read the catalog; a source with no `Searcher` must not appear as a greyed chip with an empty reason (check `browse.Service.SearchableSources`).
- A file saved from a post has no stable name; do not infer identity from it. The post id comes from the link the user pasted, so installing a download without that link records no Patreon id (it stays a plain archive entry).
- Patreon's terms bar automated access to patron content; this design makes none, and that is the reason it has no search. Do not add a "check for updates" that fetches a post page.

**Acceptance.**

- `go test ./internal/source/... ./internal/profile ./internal/archivesvc ./internal/components` pass, including fuzz seeds for the URL parser and `TestEveryCatalogReferenceResolves`.
- Add mod with a Patreon post URL opens the post in the default browser (a recorded fake opener in the test), then a file placed in a fake Downloads folder is offered and installs as an entry that names the post id; the profile's share link holds the id and no file.
- Patreon never appears among Browse's source chips or in an `all` search.

**Decisions (NOMAD, 2026-10-09).**

1. Link and handoff only: no search, no update checks, no Patreon OAuth.
2. Add mod accepts post URLs only; creator pages are not accepted.
3. Which games list `patreon` in their catalog `sources` is decided per game when a modder is known to ship there; it does not block the driver.

## Later

- **UI translations** beyond English, as Stardrop (17+), MO2 and r2modman ship: every string already goes through Lingui and the catalogs are extracted; needs chosen languages and translators. Parked 2026-10-02 (not v1).
- **macOS build**: Stardew runs on macOS, and Stardrop ships for x64 and arm64, but Mortar has no macOS CI or test machine; it needs an Apple developer account for signing and notarization, Mac Steam paths and nxm registration, and a Mac to test on. Parked 2026-10-02 (not v1).
- **EA App** as a game store, beside the drivers in `internal/gamestore/drivers.go` (Steam, Steam (Flatpak), GOG, Heroic, Lutris, Minigalaxy, Bottles): spec: [EA App](#ea-app-game-store), decided 2026-10-09. Queued 2026-10-07.
- **Patreon** as a mod source, beside the drivers in `internal/source/`: spec: [Patreon](#patreon-mod-source). Queued 2026-10-07.

Not in the first release; re-weigh only when asked:

- Translucent window, desktop showing through; on Windows it needs Acrylic measured with the frameless window. The see-through window looked wrong, so v1 is solid. The `Rethunk-AI/wails` fork's GTK4 `setTransparent()` fix (upstream wailsapp/wails#6197) makes it possible: stock GTK4 leaves `setTransparent()` empty (`wailsapp/wails` `v3/pkg/application/linux_cgo.go:1418`), and the fix registers a display-wide CSS provider that clears the window background. Any fading or `backdrop-filter` full-window layer turns WebKitGTK's translucent window opaque; a static tint does not. Stacked alphas compound toward opaque, so images and overlays each need their own alpha. Frosted glass needs `ext-background-effect-v1`, below.
- Frosted glass on Linux, built when the desktop runs GNOME 51. CSS cannot do it: `backdrop-filter` sees only the webview's pixels, and a full-window filter layer turns WebKitGTK's translucent window opaque. The compositor blurs behind the window through the Wayland protocol `ext-background-effect-v1` (Mutter from GNOME 51, KWin from Plasma 6.7; GTK 4.23.3 speaks it). Shape: in the `Rethunk-AI/wails` fork, beside `setTransparent()` in `v3/pkg/application/linux_cgo.go`, bind `ext_background_effect_manager_v1` on Wayland, and when it advertises blur, set the toplevel `wl_surface`'s blur region to the whole window, updated on resize; a no-op elsewhere, and only while the translucent window is on. Accept when the desktop behind the window shows blurred on GNOME 51 and is unchanged on GNOME 50. Offer it upstream with the GTK4 transparency PR. Windows already blurs through Acrylic.
- A hosted share service with short codes and share versioning (running costs).
- The Xbox app version (WindowsApps folders are locked down).
- Windows code signing: SignPath Foundation declined while Mortar has little public history; re-apply once it has more.
- Needs Nexus's approval through app registration first (below), since it starts downloads outside Nexus's own Mod Manager Download button: an "Add to Mortar" button on Nexus listing tiles.
- Decided against: ModDrop as a source. It has no public listing or download API (the site is a single-page app and `/api/v1/*` answers "does not exist"), so Mortar could only build a page link. SMAPI `ModDrop:N` update keys show as an unknown source.
- Decided against: deduplicating identical files across store items. Measured, not worth it (2026-10-05, a 1.6 GB store of 724 items, 50,257 files): files identical across items hold 76 MB, 4.7% of the store, under the 5% bar, and writable files barely change that; counting copies inside one item too it is 99 MB, 6.1%.
- Decided against: speeding up Play. Measured, not worth it (2026-10-05, the 811-mod Stardew profile, direct launch in a sandbox copy): Play to the game process takes 93 to 120 ms wall and 60 to 130 ms of Mortar CPU (deploy and journal included), and the warm pre-Play checks take 74 to 260 ms. A run changes the problems fingerprint, and the recheck that follows runs when the game exits, so the next Play finds it done.
- Decided against: trimming idle memory. Measured, not worth it (2026-10-05, the same profile after a full problem check): the parsed Content Patcher packs (about 80 MB) are already released two minutes after the last check, and after 30 minutes idle the heap holds 45 MB in use (RSS 192 MB). The largest retained part is the store listing for drift checks, at 13 MB, under the 50 MB bar.
- Decided against: cutting the details panel's open further. Measured, not worth it (2026-10-06, the 811-mod Stardew profile in a headless Chromium selftest, long main-thread tasks in the 400 ms after the click): the panel narrowed the card grid and remounted all 36 visible cards into new lanes, 103 to 109 ms warm. The cards now sit in one grid keyed by mod id (`ModCards.tsx` `GridWindow`), so a new column count moves them by CSS and mounts none: 79 to 81 ms warm, 0 cards added. What is left is the panel's own mount and the relayout of the narrowed grid.
- Decided against: cutting a Problems check after a change further. Measured, not worth it (2026-10-05, the 811-mod Stardew profile in a sandbox copy): an unchanged profile is served from the findings memo (`internal/framework/contentpatcher/memo.go`), 0.03 s of Content Patcher, and the window's request after a restart takes 0.3 s wall, the rest being requirements 0.1 s, the broken-mod lookup 0.04 s and the drift walk 0.06 s. After a change only the conflict targets and packs the change touches are recomputed (`internal/framework/contentpatcher/parts.go`): Content Patcher re-checks a one-pack edit in 0.15 s in a running Mortar and 0.44 s after a restart, and a pack enabled or removed in about 0.2 s. What is left is spread thin: statting every pack's files so an edit in place shows (0.045 s CPU, on every core), saving the memos and caches (0.04 s), and the targets the change really touches. Reusing the pack stamps for the tilesheet pass's map stats would save under 0.01 s.
- Decided against: collapsing the deploy, installer, metadata-provider and framework registries into direct calls. Measured, not taken (2026-10-06): it would remove about 210 lines, and `AGENTS.md` keeps them as registries so a game normally needs no code.
- Settings considered and not taken (2026-10-02): new profiles starting as a copy of the open profile or from a bundle; an offline mode that never contacts the network.
- Registering Mortar with Nexus (Collections still to ask about): SSO and OAuth PKCE code is ready behind a build flag (`nexussso.Slug` / `ClientID`, off while empty), waiting for Nexus approval and a slug or client id; see docs/nexus-application.md.
- Decided against: caching decoded PNG alpha masks for a first Problems check. Measured, not worth it (2026-10-08, the 785-mod Stardew profile): a first-ever check takes 2.1 s, about 45% of it decoding PNG alpha (`internal/framework/contentpatcher/footprint.go` `warmOverlayImages`), and every check after is about 30 ms from the existing memos, so the cache would save about 1 s once.
- Decided against: removing the links Vortex leaves in the game folder after a switch to Mortar. Play is unaffected, since Mortar starts SMAPI with the profile's own `--mods-path`, and the game page already offers to install Mortar's own SMAPI over a linked copy; re-weigh if testers report dangling links after uninstalling Vortex.
- Decided against: speeding up cold start, idle CPU and the frontend bundle. Measured, not worth it (2026-10-08, a copy of the 805-mod Stardew profile in server mode, headless Chromium): the game page with its mods shows 0.7 s after process start (0.65 s of Go CPU); idle costs 0.03% of a core with no window and 0.08% with one, the only periodic request being the 15 s network-state check; first load is 1.96 MB of JS in 56 files; reopening the Mods tab takes 120 to 175 ms with at most one 65 ms long task, and typing in the mod filter at most one 54 ms task. The frontend's repeated startup reads (settings, games, profile lists) total under 0.1 s, and the second bundled-mods sync after the components manifest arrives costs about 25 ms.
- Decided against: caching the decoded Thunderstore listing for the background mod-update check. Measured, not worth it (2026-10-08): it costs about 1 s of CPU once, five minutes after start, decoding the cached Lethal Company listing; nothing waits on it.
