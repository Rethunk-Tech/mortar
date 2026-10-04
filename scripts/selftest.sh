#!/usr/bin/env bash
# Runs Mortar in server mode against a sandboxed home, for self-testing in a browser at http://127.0.0.1:$PORT.
# The sandbox has its own HOME with a minimal Steam library holding a copy of the game, so nothing Mortar does
# reaches the real game folder, the real data folder or the real Steam config.
#
#   scripts/selftest.sh start [--copy-data]   build, set up the sandbox if missing, start the server
#   scripts/selftest.sh restart               rebuild from the working tree and restart
#   scripts/selftest.sh stop                  stop the server
#
# --copy-data copies the real Mortar profiles and settings into the sandbox once (downloads, cache, trash and
# backups are left out). The sandbox is never deleted by this script; remove $ROOT by hand to start over.
set -euo pipefail

ROOT=${MORTAR_SELFTEST_DIR:-/var/tmp/mortar-selftest}
PORT=${MORTAR_SELFTEST_PORT:-9455}
STEAM=${MORTAR_SELFTEST_STEAM:-$HOME/.local/share/Steam}
APP_ID=413150
GAME_FOLDER="Stardew Valley"
REPO=$(cd "$(dirname "$0")/.." && pwd)
SANDBOX_HOME=$ROOT/home
SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam

build() {
  echo "building frontend and server-mode binary"
  (cd "$REPO" && bun run --cwd frontend build >"$ROOT/frontend-build.log" 2>&1)
  (cd "$REPO" && GOTMPDIR=/var/tmp go build -tags server -o "$ROOT/mortar-server.new" .)
  mv "$ROOT/mortar-server.new" "$ROOT/mortar-server"
}

setup() {
  if [ -d "$SANDBOX_STEAM/steamapps/common/$GAME_FOLDER" ]; then
    return
  fi
  local game="$STEAM/steamapps/common/$GAME_FOLDER"
  [ -d "$game" ] || {
    echo "no game at $game (set MORTAR_SELFTEST_STEAM)" >&2
    exit 1
  }
  echo "copying the game into the sandbox (a copy, never a link)"
  mkdir -p "$SANDBOX_STEAM/steamapps/common" "$SANDBOX_STEAM/config"
  cp -a "$game" "$SANDBOX_STEAM/steamapps/common/"
  cp "$STEAM/steamapps/appmanifest_$APP_ID.acf" "$SANDBOX_STEAM/steamapps/"
  cp "$STEAM/config/loginusers.vdf" "$SANDBOX_STEAM/config/"
  cat >"$SANDBOX_STEAM/steamapps/libraryfolders.vdf" <<EOF
"libraryfolders"
{
	"0"
	{
		"path"		"$SANDBOX_STEAM"
		"apps"
		{
			"$APP_ID"		"1"
		}
	}
}
EOF
}

copy_data() {
  local real=$HOME/.local/share/mortar dest=$SANDBOX_HOME/.local/share/mortar
  if [ -d "$dest/profiles" ]; then
    echo "sandbox already has profile data; leaving it"
    return
  fi
  echo "copying real profiles and settings into the sandbox"
  mkdir -p "$dest"
  local item
  for item in profiles store storage overlay tools settings.json; do
    if [ -e "$real/$item" ]; then
      cp -a "$real/$item" "$dest/"
    fi
  done
}

# The server re-executes itself, so the process to stop is the one holding the port, checked against our binary.
listener() {
  ss -ltnp "sport = :$PORT" | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2
}

stop() {
  local pid
  pid=$(listener || true)
  # A rebuild replaces the binary first, so the running server's exe then reads "<path> (deleted)".
  if [ -n "$pid" ] && [ "$(readlink "/proc/$pid/exe" | sed 's/ (deleted)$//')" = "$ROOT/mortar-server" ]; then
    kill "$pid"
    # A restart must not start the next server before this one has shut down cleanly, or the next one reads
    # the unfinished shutdown as a crash.
    for _ in $(seq 1 50); do
      kill -0 "$pid" 2>/dev/null || break
      sleep 0.2
    done
    echo "stopped $pid"
  elif [ -n "$pid" ]; then
    echo "port $PORT is held by pid $pid, which is not the self-test server; leaving it" >&2
    exit 1
  fi
}

start() {
  (cd "$ROOT" && env -u XDG_DATA_HOME -u XDG_CONFIG_HOME -u XDG_CACHE_HOME HOME="$SANDBOX_HOME" \
    WAILS_SERVER_HOST=127.0.0.1 WAILS_SERVER_PORT="$PORT" nohup ./mortar-server >"$ROOT/server.log" 2>&1 &)
  for _ in $(seq 1 30); do
    if [ -n "$(listener || true)" ]; then
      echo "self-test server on http://127.0.0.1:$PORT (pid $(listener), log $ROOT/server.log)"
      return
    fi
    sleep 1
  done
  echo "server did not start; see $ROOT/server.log" >&2
  exit 1
}

mkdir -p "$ROOT"
case "${1:-}" in
  start)
    setup
    if [ "${2:-}" = "--copy-data" ]; then copy_data; fi
    build
    stop
    start
    ;;
  restart)
    build
    stop
    sleep 1
    start
    ;;
  stop) stop ;;
  *)
    sed -n '2,11p' "$0"
    exit 2
    ;;
esac
