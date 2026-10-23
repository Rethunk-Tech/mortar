# Releasing Mortar

How the packages are built and a release is cut. The gate and test commands are in [HUMANS.md](../HUMANS.md#gate).

## Building the packages

```sh
wails3 task linux:create:appimage:jammy  # bin/mortar-linux-x86_64.AppImage and the portable bin/mortar-linux-amd64 (Docker)
wails3 task linux:nfpm                   # .deb, .rpm, Arch package, bin/mortar-aur-linux-amd64
wails3 task linux:flatpak                # bin/mortar-linux-x86_64.flatpak (needs flatpak-builder)
wails3 task linux:flatpak:arm64          # bin/mortar-linux-aarch64.flatpak, on x86_64
wails3 task linux:build:arm64            # bin/mortar-aur-linux-arm64 and its .deb, .rpm, Arch package
wails3 build GOOS=windows                # bin/mortar.exe (ARCH=arm64 for Windows on ARM; wails3 task windows:package ARCH=arm64 also writes bin/mortar-arm64-installer.exe)
MORTAR_UPDATE_KEY=/path/to/updater.key wails3 task release:manifest VERSION=1.2.3
```

- **AppImage:** linuxdeploy and the type2 runtime are pinned with SHA-256 in `build/linux/appimage/pins.txt`, kept in `$XDG_CACHE_HOME/mortar-build` (`MORTAR_BUILD_CACHE` overrides; CI caches `~/.cache/mortar-build`) and checked on every build.
- **`linux:build:arm64`** cross-compiles on x86_64 with no emulator registered. It needs `zig`, `docker` (fetches the arm64 Ubuntu packages it links against, extracted under `tmp/`), `nfpm` and `qemu-aarch64` (checks that every shared library resolves). The arm64 AppImage is built only in CI.
- **`linux:flatpak:arm64`** (`build/linux/flatpak/build-aarch64.sh`) runs that cross build with `linux:flatpak ARCH=arm64`. `flatpak-builder` runs the manifest's `build-commands` through the aarch64 SDK's `/bin/sh`, so the script registers `qemu-aarch64-static` in a user namespace's own `binfmt_misc` (Linux 6.7 or newer; no root). It needs the build:arm64 tools plus `flatpak`, and unpacks any missing host tool from Fedora's packages into `tmp/rpm/`. CI builds the release bundle natively on the arm64 leg.
- **`scripts/package-matrix.sh [tag]`** downloads a release's Linux assets and, in throwaway containers (podman, else docker), installs and smoke-runs the `.deb` (Debian, Ubuntu), `.rpm` (Fedora), Arch package and AppImage (extract-and-run), then the Flatpak bundle on the host in a temporary `FLATPAK_USER_DIR`. Each case runs `mortar version`, checks the binary, `.desktop`, metainfo, icon and MIME files land, uninstalls and checks nothing is left; it prints a pass/fail table and exits non-zero on failure.
- **`scripts/windows-installer-smoke.ps1 -Installer <exe>`** does the same for the Windows installer (silent install, `mortar version`, Start Menu shortcut, autostart Run key, silent uninstall, leftovers); it needs a Windows machine or the test VM and an x64 installer.
- **`release:manifest`** refuses a `VERSION` other than `build/config.yml`'s `info.version`, needs `bin/mortar-windows-amd64.exe` (copy `bin/mortar.exe` to it; `release.yml` does), adds `mortar-windows-arm64.exe` and the arm64 AppImage when present, writes `bin/manifest.json` signed with the private key `MORTAR_UPDATE_KEY` names, then verifies it against `build/updater/public.key`. The app reads the manifest from the latest release, or the latest pre-release when Settings › Updates includes betas. The app verifies it with `updateKey` in `main.go`, the public half of that key; the private half never enters the repository.

## Cutting a release in CI

`.github/workflows/release.yml` runs on a `v*` tag:

1. The gate on the amd64 leg (skipped when `ci.yml` already passed on that commit).
2. On x86_64: the AppImage, nfpm packages, Flatpak bundle (SDK cached between runs), and the Windows exes and per-user NSIS installers for amd64 and arm64 (`bin/mortar-amd64-installer.exe`, `bin/mortar-arm64-installer.exe`, both cross-built; the arm64 installer is arm64-native). On an `ubuntu-24.04-arm` runner: the AppImage and nfpm packages again.
3. The signed `manifest.json`, the rendered AUR `PKGBUILD` and `SRCINFO` (the release asset for `.SRCINFO`), the Flathub manifest and `SUBMISSION.md`, and the GitHub release with all of them.

- **Notes** are user-facing: `build/release/notes.sh` keeps `feat`, `fix` and `perf` commits since the previous `v*` tag outside internal scopes, in sentence case, grouped New and Fixed by the areas the app's What's new shows, capped per group, with a changelog link.
- **Immutable releases:** the release is created as a draft, every draft asset is downloaded and checked against the staged files and the manifest's digests (`build/release/verify.sh`), and only then published; a rerun finishes the existing draft. After publishing, `build/release/verify.sh --public` fetches the manifest, its assets, the pinned component assets and the newest components manifest without a token: an unreachable URL fails the check when its repo is public and only warns when it is private.
- **Dry run:** a manual dispatch (Actions › Release › Run workflow) runs both legs, the signed manifest, package-manager sources, the notes (in the run summary) and `build/release/verify.sh --staged` for `build/config.yml`'s version, with no GitHub release and no attestation. A tag run reads the caches a dispatch on `main` saved but cannot share its own, so the dry run warms them for the tag.
- **Secret `MORTAR_UPDATE_KEY`** holds the private key file's PEM contents, not its path:

  ```sh
  gh secret set MORTAR_UPDATE_KEY --repo Rethunk-Tech/mortar < ~/.config/mortar-release/updater.key
  ```

- **Optional org secret** `MORTAR_REPO_GPG_KEY` (`gh secret set MORTAR_REPO_GPG_KEY --org Rethunk-Tech --repos Rethunk-Tech/mortar`), the armored private key of "Mortar packages <opensource@rethunk.tech>" (fingerprint `D8B1C4C205C5FB3633CB8B9900B75959B637DCB5`); the output is skipped with a run notice while unset. After publishing, the `package-repo` job runs `scripts/package-repo.sh` on the release's `.deb`, `.rpm` and Arch packages, uploads the signed tree as the `package-repo` artifact and force-pushes it, under `packages/` beside the tag's `site/`, as the single commit of the orphan `packages` branch. App Platform deploys both components from that branch on push (site at <https://mortar.rethunk.tech>, repository at /packages), so the site changes only with a release. `package-repo.yml` (Actions › Package repository) rebuilds both for an existing tag.
  - The tree holds `deb/`, `rpm/<arch>/`, `arch/<arch>/` and `flatpak/` (an OSTree repo with signed commits and summary), the client snippets `mortar.list`, `mortar.repo` and `pacman.conf`, and the public key. The rpm packages are signed copies of the release assets, so `mortar.repo` sets both `repo_gpgcheck=1` and `gpgcheck=1`.
  - The script refuses a key whose fingerprint differs from the committed `build/linux/repo/mortar-archive-keyring.asc` and holds only the latest release. Locally (needs podman or docker, and gpg): `MORTAR_REPO_KEY=<private key file> MORTAR_REPO_URL=<base url> scripts/package-repo.sh <asset dir> <out dir>`.

## Each release

1. **Check it.** Gate-green is not QA: walk every screen on a `--copy-data` sandbox at the commit to tag ([AGENTS.md](../AGENTS.md) § Testing), and after a layout change run `bun run renders`, which checks every screen in Chromium and WebKitGTK.
2. **Player notes** go in `build/release/notes/v1.2.3.md` ("## New", "## Changed" and "## Fixed" lists, written for players); `build/release/notes.sh` prefers them to the commit list, for the GitHub release, the metainfo and the Discord post.
3. **Version.** Set `info.version` in `build/config.yml` (the only place), then `go run ./cmd/version` copies it into the Windows resources, the Linux metainfo and the AUR `PKGBUILD` (`main.go` reads `build/config.yml` itself). Run `build/release/notes.sh --metainfo build/linux/tech.rethunk.Mortar.metainfo.xml v1.2.3 HEAD` to write the release's user-facing notes into that version's `<release><description>` (software centres and Flathub show it). Gate (it fails on any differing copy, and `appstreamcli validate --strict --no-net` checks the metainfo), commit and push `main`.
4. **News post and pictures,** in the same commit as the version: `web/src/content/news/mortar-1-2-3.md` with two or three highlights, each with a picture from `bun frontend/e2e/screens.ts --set site --only <scenes>` against a `--copy-data` sandbox (it writes `web/public/img/<scene>.webp`, and the AppStream copies in `build/linux/screenshots/`). The site check refuses a version with no post. An earlier post keeps the picture it shipped with: before retaking a picture an older post shows, copy it to a versioned name and point that post at it.
5. **Tag:** gate, commit, push `main`, then `git tag v1.2.3 && git push origin v1.2.3`. The tag must equal that version with a leading `v`, or the release stops before building. `release.yml` publishes the site with the packages, and `release-discord.yml` posts the notes.
6. **Between releases,** a site fix publishes through Actions › Package repository with the latest release's tag and `site` set to the ref to build; that ref's app frontend is the site's demo.

## Flathub

CI attaches a single-file `.flatpak` for sideloading, built by `build/linux/flatpak/tech.rethunk.Mortar.yml` from the CI binary; users get the Flatpak from Mortar's own repo at mortar.rethunk.tech/packages. The Flathub listing is a separate submission that builds the tagged source.

- `build/linux/flathub/render.sh OUT_DIR` writes it. `tech.rethunk.Mortar.yml` is the sideload manifest's header and finish-args (the one list of permissions; `build/linux/flathub/render.test.ts` keeps the Flathub template free of its own) above `build/linux/flathub/tech.rethunk.Mortar.yml`'s modules, which build the wails3 CLI, the TypeScript bindings, the frontend (`vite build`; catalogs, credits and notices are committed) and Mortar with `-X main.packaged=flatpak`, inside the GNOME SDK with the Go and Node 24 SDK extensions and no network.
- `go-sources.json` lists every proxy.golang.org file a clean amd64 and arm64 build reads, served by `GOPROXY=file://`. `node-sources.json` comes from [flatpak-node-generator](https://github.com/flatpak/flatpak-builder-tools/tree/master/node) over `yarn.lock` (bun's lock in yarn v1 format; the generator reads no bun lock).
- Rendering needs network, Go, bun, jq and `uvx`. `--local` builds this checkout's HEAD instead of the tag, for `flatpak run org.flatpak.Builder` tests; the builder sees only home and `/var/tmp`, so test from a clone there.
- Each release attaches the four files with a `flathub-` prefix and `flathub-SUBMISSION.md` (the PR text). To submit: meet the [Flathub app requirements](https://docs.flathub.org/docs/for-app-authors/requirements) (AppStream metainfo `tech.rethunk.Mortar.metainfo.xml`, 128×128 and 256×256 icons, screenshots, AGPL-3.0 license text; builds from source with no network, every dependency a manifest source with a public URL), fork [flathub/flathub](https://github.com/flathub/flathub), open a PR adding `tech.rethunk.Mortar`, then maintain the app repo Flathub creates.
- finish-args (listed in `build/linux/flatpak/tech.rethunk.Mortar.yml`) cover the game libraries and launcher data Mortar manages, plus `org.freedesktop.Flatpak` (`flatpak-spawn --host` starts Steam and the game outside the sandbox). Launch at login and profile shortcuts use the Background and DynamicLauncher portals. The game-library paths and `flatpak-spawn` are the linter exceptions `SUBMISSION.md` requests.

## winget

`build/windows/winget/` holds the last submitted manifests for `RethunkTech.Mortar`. After a release publishes, `build/windows/winget/bump.sh 1.2.3 <winget-pkgs checkout>` verifies the release's `SHA256SUMS` signature against the repo key and writes `manifests/r/RethunkTech/Mortar/1.2.3/` with both installers' hashes. Run `winget validate`, send the update to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) as a PR, and copy the new manifests back over `build/windows/winget/` once it merges. Scoop and the Homebrew tap update themselves from the release ([scoop-bucket](https://github.com/Rethunk-Tech/scoop-bucket), [homebrew-tap](https://github.com/Rethunk-Tech/homebrew-tap)).

## AUR mortar-bin

`build/linux/aur/PKGBUILD` installs `mortar-aur-linux-amd64` or `mortar-aur-linux-arm64` (the packaged build, updater off) from the GitHub release, plus the tagged desktop entry and icon. Publishing:

1. `git clone ssh://aur@aur.archlinux.org/mortar-bin.git`
2. Copy the release's `PKGBUILD` and `SRCINFO` (saved as `.SRCINFO`) in; both carry real sums. The committed `PKGBUILD` keeps `SKIP` sums and its `pkgver` follows `build/config.yml` through `go run ./cmd/version`. If the source archive was not publicly downloadable when the release rendered them, its sum is `SKIP`: run `updpkgsums` and `makepkg --printsrcinfo > .SRCINFO`.
3. `git add PKGBUILD .SRCINFO && git commit -m "mortar-bin $pkgver" && git push`
