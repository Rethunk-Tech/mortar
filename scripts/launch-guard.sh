#!/usr/bin/env bash
# Starts one game for a self-test sandbox (scripts/selftest.sh names it in MORTAR_LAUNCH_WRAPPER, and a server build
# runs every direct launch through it): it refuses a launch that could reach the desktop session, counts the launch
# against the session's cap, records the game's pid and starts it without the network.
#
#   launch-guard.sh COMMAND [ARG...]
#
# MORTAR_LAUNCH_CAP        launches allowed per session (default 3)
# MORTAR_LAUNCH_COUNT      the session's counter file
# MORTAR_LAUNCH_PIDS       the sandbox's record of started games: "pid starttime command" per line
# MORTAR_HIDDEN_RUNTIME    the hidden display's runtime dir, which XDG_RUNTIME_DIR must be
set -euo pipefail

if [ "${1:-}" = --record ]; then
  shift
  start=$(sed 's/.*) //' "/proc/$$/stat" | cut -d' ' -f20)
  printf '%s %s %s\n' "$$" "$start" "$*" >>"${MORTAR_LAUNCH_PIDS:?}"
  exec "$@"
fi

cap=${MORTAR_LAUNCH_CAP:-3}
count=${MORTAR_LAUNCH_COUNT:?MORTAR_LAUNCH_COUNT is not set}
: "${MORTAR_LAUNCH_PIDS:?MORTAR_LAUNCH_PIDS is not set}"
hidden=${MORTAR_HIDDEN_RUNTIME:?MORTAR_HIDDEN_RUNTIME is not set}

refuse() {
  echo "launch-guard: launch refused: $*" >&2
  exit 3
}

# The hidden display's Wayland socket lives in its own runtime dir, so a launch that kept the session's
# XDG_RUNTIME_DIR, or names another socket, would draw on the desktop.
[ "${XDG_RUNTIME_DIR:-}" = "$hidden" ] || refuse "XDG_RUNTIME_DIR is ${XDG_RUNTIME_DIR:-unset}, not the hidden display's $hidden"
[ -n "${WAYLAND_DISPLAY:-}" ] && [ -S "$hidden/$WAYLAND_DISPLAY" ] || refuse "WAYLAND_DISPLAY ${WAYLAND_DISPLAY:-unset} is not a socket in $hidden"
[ -n "${DISPLAY:-}" ] && [ -f "$hidden/display.x11" ] && [ "$DISPLAY" = "$(cat "$hidden/display.x11")" ] ||
  refuse "DISPLAY ${DISPLAY:-unset} is not the hidden display's X server"
[ $# -gt 0 ] || refuse "no command"

mkdir -p "$(dirname "$count")"
exec 9>>"$count.lock"
flock 9
used=$(cat "$count" 2>/dev/null || echo 0)
[ "$used" -lt "$cap" ] || refuse "this session has used all $cap sandbox game launches ($count)"
echo $((used + 1)) >"$count"
exec 9>&-

# Inside the sandbox the guard runs again with --record, which notes the game's own pid and becomes the game. A network
# namespace of its own keeps the game from any Steam client listening on loopback.
# The Mortar bridge skips Lethal Company's intro for a launch that sets this; only a sandbox launch passes through
# here, so the user's own play never sees it.
export MORTAR_SKIP_INTRO=1
exec bwrap --dev-bind / / --unshare-net -- "$0" --record "$@"
