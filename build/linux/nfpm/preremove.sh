#!/bin/sh
# Runs as root before the package's files are removed (deb prerm, rpm %preun, pacman pre_remove). What Mortar wrote
# lives in each user's home: the nxm:// default and browser native-messaging hosts, the mortar:// and .mortar defaults,
# the autostart entry, profile desktop entries and the profiles added to Steam. Root cannot tell who ran Mortar, so
# this cleans only the user who ran the package manager through sudo or pkexec, as that user; other users, and a
# removal from a graphical software centre, run `mortar --release-links && mortar uninstall-cleanup` themselves first.
# The data folder (profiles, mods, settings) is never touched. A failure here never blocks the removal.

# deb passes "upgrade" and rpm a remaining-install count of 1 or more when this is an upgrade, not a removal; pacman
# runs pre_remove only for a removal and passes the old version.
case "${1:-}" in
  upgrade | failed-upgrade) exit 0 ;;
  '' | *[!0-9]*) ;;
  *) [ "$1" -gt 0 ] && exit 0 ;;
esac

user="${SUDO_USER:-}"
if [ -z "$user" ] && [ -n "${PKEXEC_UID:-}" ]; then
  user="$(getent passwd "$PKEXEC_UID" | cut -d: -f1)"
fi
if [ -z "$user" ] || [ "$user" = root ]; then
  exit 0
fi
home="$(getent passwd "$user" | cut -d: -f6)"
if [ -z "$home" ] || [ ! -d "$home" ] || ! command -v runuser >/dev/null 2>&1; then
  exit 0
fi

as_user() {
  timeout 30 runuser -u "$user" -- env -i HOME="$home" USER="$user" PATH=/usr/bin:/bin /usr/bin/mortar "$@" \
    >/dev/null 2>&1 || true
}
as_user quit
as_user --release-links
as_user uninstall-cleanup
exit 0
