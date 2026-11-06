#!/usr/bin/env bash
# Starts one game for a self-test sandbox (scripts/selftest.sh names it in MORTAR_LAUNCH_WRAPPER, and a server build
# runs every direct launch through it): it refuses a launch that could reach the desktop session, counts the launch
# against the session's cap, records the game's pid and starts it without the host's network, in a network namespace
# shared by every game this sandbox (one session) starts, so two copies can reach each other on loopback.
#
#   launch-guard.sh COMMAND [ARG...]
#
# MORTAR_LAUNCH_CAP        launches allowed per session (default 3)
# MORTAR_LAUNCH_COUNT      the session's counter file
# MORTAR_LAUNCH_PIDS       the sandbox's record of started games: "pid starttime command" per line
# MORTAR_HIDDEN_RUNTIME    the hidden display's runtime dir, which XDG_RUNTIME_DIR must be (a PULSE_SERVER must be a socket in it)
set -euo pipefail

if [ "${1:-}" = --hold ]; then
  # The namespace's holder: it lives as long as the sandbox, so later copies can join the namespace it keeps.
  start=$(sed 's/.*) //' "/proc/$$/stat" | cut -d' ' -f20)
  printf '%s %s\n' "$$" "$start" >"$2.new"
  mv "$2.new" "$2"
  exec sleep infinity
fi

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
# The sandbox's own audio server (scripts/selftest.sh) is the only one a game may name.
case "${PULSE_SERVER:-unix:$hidden/}" in "unix:$hidden/"*) ;; *) refuse "PULSE_SERVER $PULSE_SERVER is not a socket in $hidden" ;; esac
[ $# -gt 0 ] || refuse "no command"

mkdir -p "$(dirname "$count")"
exec 9>>"$count.lock"
flock 9
used=$(cat "$count" 2>/dev/null || echo 0)
[ "$used" -lt "$cap" ] || refuse "this session has used all $cap sandbox game launches ($count)"
echo $((used + 1)) >"$count"

# One network namespace per sandbox, which has one session key: a holder process keeps it, and a later copy joins it
# with nsenter (entering its user namespace too, since an unprivileged process may not join a network namespace alone).
# The holder is made under the launch lock so two copies started together share one.
netfile=$(dirname "$MORTAR_LAUNCH_PIDS")/launch-net.pid
holder_live() {
  local pid start
  read -r pid start 2>/dev/null <"$netfile" || return 1
  [ "$(sed 's/.*) //' "/proc/$pid/stat" 2>/dev/null | cut -d' ' -f20)" = "$start" ] && holder=$pid
}
holder=
if ! holder_live; then
  rm -f "$netfile"
  setsid bwrap --dev-bind / / --unshare-net -- "$0" --hold "$netfile" 9>&- >/dev/null 2>&1 </dev/null &
  for _ in $(seq 1 100); do
    holder_live && break
    sleep 0.1
  done
  [ -n "$holder" ] || refuse "the session's network namespace did not start"
fi
exec 9>&-

# Inside the namespace the guard runs again with --record, which notes the game's own pid and becomes the game. The
# namespace keeps the game from any Steam client listening on loopback.
# The Mortar bridge skips Lethal Company's intro for a launch that sets this; only a sandbox launch passes through
# here, so the user's own play never sees it.
export MORTAR_SKIP_INTRO=1
exec nsenter --user --net --preserve-credentials --target "$holder" -- "$0" --record "$@"
