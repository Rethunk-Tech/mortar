# Security policy

The licence is in [`LICENSE`](LICENSE).

## Reporting a vulnerability

Do not open an issue for a security report. Use GitHub's private vulnerability reporting on this repository, or email <security@rethunk.tech>, with a description, the affected paths or version, steps to reproduce, and the impact.

## Supported versions

Only the latest release is supported; fixes ship in a new release, not as patches to older ones.

## Verifying downloads

Packages, repositories and each release's `SHA256SUMS` are signed with the OpenPGP key "Mortar packages <security@rethunk.tech>":

```
3283 6046 06CA E229 5D47  6F98 83BC 8751 EE6F 773D
```

The key is in this repository at [`build/linux/repo/mortar-archive-keyring.asc`](build/linux/repo/mortar-archive-keyring.asc), at https://mortar.rethunk.tech/packages/mortar-archive-keyring.asc and on keyserver.ubuntu.com. To check a download:

```sh
curl -fsSL https://mortar.rethunk.tech/packages/mortar-archive-keyring.asc | gpg --import
gpg --verify SHA256SUMS.asc SHA256SUMS
sha256sum --check --ignore-missing SHA256SUMS
gh attestation verify <file> --repo Rethunk-Tech/mortar
```

The last line checks the GitHub build provenance: that the file was built by this repository's release workflow. Mortar's own updater checks every update against an ed25519 signature in the release's `manifest.json` before installing it.
