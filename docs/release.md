# Releasing Mortar

How the packages are built and a release is cut. The gate and test commands are in [HUMANS.md](../HUMANS.md#gate). Mortar hosts its own Flatpak repo (mortar.rethunk.tech/packages); the Flathub listing is submitted separately ([Flathub](#flathub)). The site (`web/`) deploys only with a release.

## Building the packages

The recipes are the `wails3 task` targets in `Taskfile.yml` (`linux:create:appimage:jammy`, `linux:nfpm`, `linux:flatpak`, `linux:flatpak:arm64`, `linux:build:arm64`, `windows:package`) and `wails3 build GOOS=windows`. Each task's header names its tools and outputs.

- **Signed manifest:** `MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3`. The app verifies the manifest with `updateKey` in `main.go`, the public half of that key; the private half never enters the repository. The app reads the manifest from the latest release, or the latest pre-release when Settings › Updates includes betas.
- **Package smoke tests:** `scripts/package-matrix.sh [tag]` (Linux packages in throwaway containers) and `scripts/windows-installer-smoke.ps1 -Installer <exe>` (needs a Windows machine or the test VM).

## Cutting a release in CI

`.github/workflows/release.yml` runs on a `v*` tag: gate, all packages for both architectures, the signed manifest, the AUR and Flathub sources, and the GitHub release. The release is created as a draft, checked by `build/release/verify.sh` and only then published; a rerun finishes the existing draft. A manual dispatch is a dry run with no GitHub release; it also warms the caches a tag run reads. The tag must equal `build/config.yml`'s `info.version` with a leading `v`, or the release stops before building.

- **Secret `MORTAR_UPDATE_KEY`** holds the private key file's PEM contents, not its path: `gh secret set MORTAR_UPDATE_KEY --repo Rethunk-Tech/mortar < ~/.config/mortar-release/updater.key`.
- **Optional org secret `MORTAR_REPO_GPG_KEY`** is the armored private key of "Mortar packages <opensource@rethunk.tech>"; its fingerprint must match the committed `build/linux/repo/mortar-archive-keyring.asc`. While unset the package repository step is skipped with a notice. Set it with `gh secret set MORTAR_REPO_GPG_KEY --org Rethunk-Tech --repos Rethunk-Tech/mortar`.
- **Package repository:** `scripts/package-repo.sh` builds the signed apt, rpm, pacman and Flatpak (OSTree) tree from the release's packages and the `package-repo` job force-pushes it, under `packages/` beside the tag's `site/`, as the single commit of the orphan `packages` branch; App Platform deploys both from it. Actions › Package repository rebuilds both for an existing tag. Locally (needs podman or docker, and gpg): `MORTAR_REPO_KEY=<private key file> MORTAR_REPO_URL=<base url> scripts/package-repo.sh <asset dir> <out dir>`.

## Each release

1. **Check it.** Gate-green is not QA: walk every screen on a `--copy-data` sandbox at the commit to tag ([AGENTS.md](../AGENTS.md) § Testing), and run `bun run renders:release`.
2. **Player notes** go in `build/release/notes/v1.2.3.md` ("## New", "## Changed" and "## Fixed" lists, written for players); `build/release/notes.sh` prefers them to the commit list, for the GitHub release, the metainfo and the Discord post.
3. **Version.** Set `info.version` in `build/config.yml` (the only place), then `go run ./cmd/version`. Run `build/release/notes.sh --metainfo build/linux/tech.rethunk.Mortar.metainfo.xml v1.2.3 HEAD` to write the notes into that version's `<release><description>`. Gate, commit and push `main`.
4. **News post and pictures,** in the same commit: `web/src/content/news/mortar-1-2-3.md` with two or three highlights, each with a picture from `bun frontend/e2e/screens.ts --set site --only <scenes>` against a `--copy-data` sandbox, after `scripts/selftest.sh prep-shots` (the maintainer's Nexus and itch.io keys, so signed-in checks run, and no update notices). The site check refuses a version with no post. An earlier post keeps the picture it shipped with: before retaking a picture an older post shows, copy it to a versioned name and point that post at it.
5. **Tag:** `git tag v1.2.3 && git push origin v1.2.3`. `release.yml` publishes the site with the packages, and `release-discord.yml` posts the notes.
6. **Between releases,** a site fix publishes through Actions › Package repository with the latest release's tag and `site` set to the ref to build; that ref's app frontend is the site's demo.

## Flathub

CI attaches a single-file `.flatpak` for sideloading; users get the Flatpak from Mortar's own repo. The Flathub listing is a separate submission that builds the tagged source with no network. `build/linux/flathub/render.sh OUT_DIR` writes its manifest and source lists; `--local` builds this checkout's HEAD instead of the tag (the builder sees only home and `/var/tmp`, so test from a clone there). `tech.rethunk.Mortar.yml` in `build/linux/flatpak/` is the one list of finish-args. `node-sources.json` comes from [flatpak-node-generator](https://github.com/flatpak/flatpak-builder-tools/tree/master/node) over `yarn.lock` (bun's lock in yarn v1 format).

Each release attaches those four files with a `flathub-` prefix. The maintainer submits by hand: Flathub refuses AI agents on submission PRs and AI-written PR text. Meet the [app requirements](https://docs.flathub.org/docs/for-app-authors/requirements), add the four files (unprefixed) on a branch of [flathub/flathub](https://github.com/flathub/flathub)'s `new-pr`, and open the PR against `new-pr`. The PR asks for linter exceptions for the game-library and launcher folders and `flatpak-spawn`; the finish-args comments give each grant's reason. After the merge, each release replaces the four files in the `flathub/tech.rethunk.Mortar` repo Flathub creates.

## winget

After a release publishes, `build/windows/winget/bump.sh 1.2.3 <winget-pkgs checkout>` writes `manifests/r/RethunkTech/Mortar/1.2.3/` and rewrites the templates in `build/windows/winget/` to match. Run `winget validate`, send the update to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) as a PR, and commit the templates. Scoop and the Homebrew tap update themselves from the release.

## AUR mortar-bin

`build/linux/aur/PKGBUILD` installs the packaged build (updater off) from the GitHub release. Each push needs the AUR account holder's permission. Publishing:

1. `git clone ssh://aur@aur.archlinux.org/mortar-bin.git`
2. Copy the release's `PKGBUILD` and `SRCINFO` (saved as `.SRCINFO`) in; both carry real sums. If the source archive was not publicly downloadable when the release rendered them, its sum is `SKIP`: run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`.
3. `git add PKGBUILD .SRCINFO && git commit -m "mortar-bin $pkgver" && git push`

## Facts that settle questions

- **Linux artifacts:** the AppImage is built in an ubuntu:22.04 container with the fork's `gtk3` tag because 22.04 has no WebKitGTK 6.0, so it bundles GTK 3 and WebKitGTK 4.1 and needs that container's glibc; AppRun sources `build/linux/appimage/glibc-check.sh`, so an older host gets a dialog pointing to the Flatpak. nfpm packages and the AUR ELF are linked with `-X main.packaged=...`. The AppImage's unpackaged binary ships as the portable `bin/mortar-linux-<arch>`, which updates itself and writes its own desktop entry and nxm handler. The package remove script (`build/linux/nfpm/preremove.sh`, skipped on upgrade) runs `mortar quit`, `--release-links` and `uninstall-cleanup` as the user who ran the package manager through sudo or pkexec, since per-user state is out of root's reach.
- **Flatpak sandbox:** paths outside the finish-args are invisible until `flatpak override --user --filesystem=<path> tech.rethunk.Mortar`. The sandbox cannot see the host's Steam or processes, so Mortar goes through `flatpak-spawn --host` (go-desktopenv's `flatpak` package) and the game never inherits the sandbox. Add to Steam is refused in the Flatpak (host Steam cannot start `/app/bin/mortar`).
- **Windows:** a per-user NSIS installer is the only scope (`WAILS_INSTALL_SCOPE` in `build/windows/nsis/wails_tools.nsh`), installing to `$LOCALAPPDATA\Programs\Mortar` without admin; a machine-wide build would register a second copy beside it. The installer and uninstaller close a running Mortar first with `mortar quit` (refused while downloads or the game run unless `--force`).
- **Self-update:** `internal/updatesvc` drives Wails' updater (`app.Updater`) with the `endpoint` provider reading the signed manifest; signing key `main.go` `updateKey`. Wails' GitHub provider is not used because it verifies no signature. A release is installed only when its manifest entry carries a valid signature; asset URLs name the release tag, so a release published mid-update cannot swap the file under it. Updating is off in a build without the `production` tag, a server build, a version containing `dev`, and a packaged build. The bare Linux binaries are listed in the manifest under platform `linux-portable` (`updatesvc.PortablePlatform`; `release:manifest` rewrites their inferred `linux`, which the per-file signatures do not cover).
- **What's new:** `<datadir>/last-run-version` records the previous run's version; when the running version is newer, Mortar fetches that release's GitHub notes once and shows them; offline leaves the file so the next start tries again.
- **Wails behaviour Mortar relies on** (upstream, except `OnUpdateApplied` and the other additions listed in [design](design.md), which only the `Rethunk-AI/wails` fork that `go.mod` pins carries): the self-contained AppImage patches the bundled WebKit library's compiled-in path to a relative one, with AppRun running from `$APPDIR/usr/lib` and the user's folder in `OWD` (`sharesvc.LaunchDir`). Updater staging goes beside the target and a cross-filesystem rename falls back to a copy (`/tmp` is tmpfs on Fedora); inside an AppImage it replaces and relaunches `$APPIMAGE`, not the read-only mount; `Config.OnUpdateApplied` lets Mortar rewrite `DisplayVersion` on Windows.
- **Linux desktop integration:** packaged installs (nfpm, AUR, Flatpak) own the system desktop entry, MIME XML and icons, and a packaged binary writes none at startup. Nothing installs an AppImage's embedded `.desktop` file, so an AppImage writes its own with `Exec=` set to `$APPIMAGE` (desktop files do not expand variables and the mount vanishes on exit), rewritten at startup when its target no longer exists (`Refresh` in `internal/nxm/handler_linux.go`). Application id and single-instance id are both `tech.rethunk.Mortar`; a portable copy appends a hash of its data folder.
