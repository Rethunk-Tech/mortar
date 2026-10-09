#!/usr/bin/env bash
# Runs Mortar in server mode against a sandboxed home, for self-testing in a browser at http://127.0.0.1:$PORT.
# The sandbox has its own HOME with a minimal Steam library holding copies of the games (Stardew Valley and, when installed, Lethal Company and Valheim), so nothing Mortar does
# reaches the real game folder, the real data folder or the real Steam config.
#
#   scripts/selftest.sh start [--copy-data]   build, set up the sandbox if missing, start the server
#   scripts/selftest.sh setup                 only create or top up the sandbox's Steam library; no build, no server
#   scripts/selftest.sh restart               rebuild from the working tree and restart
#   scripts/selftest.sh stop                  stop the server
#   scripts/selftest.sh destroy               stop the server and everything running from the sandbox, then delete it
#   scripts/selftest.sh reap [--hours N] [--yes]  list (and with --yes delete) idle sandboxes under the base dir
#   scripts/selftest.sh seed                  fill the running sandbox with fixture data (once; skipped when present)
#   scripts/selftest.sh ea-fixture FOLDER MARKER  add a fake Bottles bottle holding an EA App install of FOLDER (discovery only)
#   scripts/selftest.sh regress --fake GAME_ID  launch regress for a game not installed here: a fake install and a dummy window (see regress_fake)
#   scripts/selftest.sh regress [--game lethal-company]  one-shot regression run (Stardew, or Lethal Company through Proton) in its own throwaway sandbox (see below)
#   scripts/selftest.sh curseforge            CurseForge end to end in its own throwaway sandbox; needs MORTAR_CURSEFORGE_KEY, else skips
#   scripts/selftest.sh harness-check         prove the launch harness with a dummy window instead of a game
#
# Every game a sandbox starts, from the browser or from regress, runs on the sandbox's own headless display (mutter
# with Xwayland, its own D-Bus and runtime dir; the desktop's DISPLAY, WAYLAND_DISPLAY and bus are unset) through
# scripts/launch-guard.sh, without the network, and each session gets at most 3 launches across all its sandboxes.
#
# --copy-data copies the real Mortar profiles and settings (or those in MORTAR_SELFTEST_DATA, a Mortar data folder) into
# the sandbox once (downloads, cache, trash and backups are left out). Every sandbox root carries a .mortar-selftest marker, and only a marked folder directly under
# $MORTAR_SELFTEST_BASE (default /var/tmp) is ever deleted: by destroy, or by reap once nothing runs from it and nothing
# in it changed for N hours (default 6). reap only lists unless given --yes.
#
# regress is the exception: it makes /var/tmp/mortar-regress-XXXXXX on a free port, copies the real data, launches the
# first Stardew profile directly, waits for SMAPI to load every enabled mod, then stops the game and requires the game
# folder to hash exactly as before. It deletes only its own directory, after a pass; a failure keeps it for the logs.
# MORTAR_REGRESS_PROFILE picks another profile and MORTAR_REGRESS_TIMEOUT (seconds, default 180) bounds the load wait.
# regress --game lethal-company builds its own sandbox, bootstraps a Proton prefix, installs BepInEx and three plugins
# from Thunderstore (cached in /var/tmp/mortar-regress-cache; MORTAR_REGRESS_OFFLINE=1 proves a rerun needs no network),
# launches directly under Proton, and requires the plugins to load and the purge to leave the game folder identical.
# MORTAR_REGRESS_MATRIX=1 also runs the BepInEx test matrix (scripts/regress-bepinex.sh, docs/bepinex-test-matrix.md)
# within three launches, the base one and two more, which leaves none for MORTAR_REGRESS_R2_CODE (refused with it); it needs the network and the .NET SDK, and MORTAR_REGRESS_R2_EXPORT=1
# also publishes an r2modman code to thunderstore.io.
set -euo pipefail

BASE=${MORTAR_SELFTEST_BASE:-/var/tmp}
ROOT=${MORTAR_SELFTEST_DIR:-$BASE/mortar-selftest}
MARKER=.mortar-selftest
PORT=${MORTAR_SELFTEST_PORT:-9455}
STEAM=${MORTAR_SELFTEST_STEAM:-$HOME/.local/share/Steam}
APP_ID=413150
GAME_FOLDER="Stardew Valley"
LC_APP_ID=1966720
LC_FOLDER="Lethal Company"
VH_APP_ID=892970
VH_FOLDER=Valheim
REPO=$(cd "$(dirname "$0")/.." && pwd)
SANDBOX_HOME=$ROOT/home
SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
# Games the shipped catalog has not enabled yet are switched on in the sandbox only (comma separated catalog ids).
export MORTAR_ENABLE_GAMES=${MORTAR_SELFTEST_ENABLE:-lethal-company}
# shellcheck source=scripts/regress-bepinex.sh
. "$REPO/scripts/regress-bepinex.sh"

# Every game a sandbox starts goes through scripts/launch-guard.sh onto a display of the sandbox's own, and a session
# gets at most LAUNCH_CAP launches across all its sandboxes. The session is MORTAR_LAUNCH_SESSION, else the agent
# session, else the login session; its counter lives beside the sandboxes, since a regress sandbox lasts one run.
LAUNCH_CAP=3
SESSION_KEY=$(printf '%s' "${MORTAR_LAUNCH_SESSION:-${CLAUDE_CODE_SESSION_ID:-${XDG_SESSION_ID:-default}}}" | tr -c 'A-Za-z0-9_-' _)
LAUNCH_COUNT=$BASE/mortar-launch-cap/$SESSION_KEY

# The last server binary built, kept under the hash of the tree it came from so an unchanged tree skips the build.
BUILD_CACHE=/var/tmp/mortar-selftest-build

# Every file the build reads that git sees; specs and docs change no binary.
tree_key() {
  (cd "$REPO" && git ls-files -co --exclude-standard -z -- . ':!frontend/e2e' ':!docs' ':!*.md' |
    xargs -0 sha256sum 2>/dev/null | sha256sum | cut -d' ' -f1)
}

build() {
  sandbox_tmp
  # The server embeds frontend/dist, which every sandbox's build rewrites, so concurrent sandboxes build one at a time.
  exec 9>/var/tmp/mortar-selftest-build.lock
  flock 9
  local key
  key=$(tree_key)
  if [ -f "$BUILD_CACHE/$key" ]; then
    echo "server-mode binary unchanged since the last build; reusing it"
    cp "$BUILD_CACHE/$key" "$ROOT/mortar-server.new"
  else
    echo "building frontend and server-mode binary"
    # A fresh clone has no generated bindings, which the frontend build imports.
    if [ ! -d "$REPO/frontend/bindings" ]; then
      (cd "$REPO" && wails3 generate bindings -clean=true -ts -i >"$ROOT/bindings.log" 2>&1)
    fi
    (cd "$REPO" && bun run --cwd frontend build >"$ROOT/frontend-build.log" 2>&1)
    (cd "$REPO" && go build -tags server -o "$ROOT/mortar-server.new" .)
    # Filed under the key taken before the build: the tree can change while it runs (another worker's edit), and a
    # key taken after would file an older binary under the newer tree.
    rm -rf "$BUILD_CACHE"
    mkdir -p "$BUILD_CACHE"
    cp "$ROOT/mortar-server.new" "$BUILD_CACHE/$key"
  fi
  exec 9>&-
  mv "$ROOT/mortar-server.new" "$ROOT/mortar-server"
}

# Each game is copied once; a sandbox that already holds one is topped up with the other.
copy_game() {
  local app=$1 folder=$2 game="$STEAM/steamapps/common/$2"
  [ -d "$SANDBOX_STEAM/steamapps/common/$folder" ] && return
  [ -d "$game" ] || {
    echo "no game at $game (set MORTAR_SELFTEST_STEAM)" >&2
    return 1
  }
  echo "copying $folder into the sandbox (a copy, never a link)"
  mkdir -p "$SANDBOX_STEAM/steamapps/common"
  cp -a "$game" "$SANDBOX_STEAM/steamapps/common/"
  cp "$STEAM/steamapps/appmanifest_$app.acf" "$SANDBOX_STEAM/steamapps/"
}

# Lists every copied game as installed, so Steam discovery finds exactly what the sandbox holds.
write_library() {
  local apps="" manifest app
  for manifest in "$SANDBOX_STEAM"/steamapps/appmanifest_*.acf; do
    [ -f "$manifest" ] || continue
    app=${manifest##*appmanifest_}
    app=${app%.acf}
    apps+=$'\t\t\t"'$app$'"\t\t"1"\n'
  done
  printf '"libraryfolders"\n{\n\t"0"\n\t{\n\t\t"path"\t\t"%s"\n\t\t"apps"\n\t\t{\n%s\t\t}\n\t}\n}\n' "$SANDBOX_STEAM" "$apps" >"$SANDBOX_STEAM/steamapps/libraryfolders.vdf"
}

# ea_fixture FOLDER MARKER adds a fake Bottles bottle to the sandbox home holding an EA App install of FOLDER (a stub
# MARKER file under drive_c/Program Files/EA Games), which is how an EA game reaches Mortar on Linux. Discovery
# only: Bottles itself is not run, so a launch of it needs bottles-cli.
ea_fixture() {
  local folder=${1:?usage: selftest.sh ea-fixture FOLDER MARKER} marker=${2:?usage: selftest.sh ea-fixture FOLDER MARKER}
  local bottle="$SANDBOX_HOME/.local/share/bottles/bottles/EAFixture"
  mkdir -p "$bottle/drive_c/Program Files/EA Games/$folder"
  : >"$bottle/bottle.yml"
  printf '#!/bin/sh\nexit 0\n' >"$bottle/drive_c/Program Files/EA Games/$folder/$marker"
  chmod +x "$bottle/drive_c/Program Files/EA Games/$folder/$marker"
  echo "EA fixture at $bottle"
}

# owned PID succeeds when PID still runs and its executable or working directory is inside $ROOT.
owned() {
  local exe cwd
  exe=$(readlink "/proc/$1/exe" 2>/dev/null) || return 1
  cwd=$(readlink "/proc/$1/cwd" 2>/dev/null) || return 1
  case "$exe/ $cwd/" in "$ROOT"/* | *" $ROOT"/*) return 0 ;; esac
  return 1
}

# display_up starts the sandbox's hidden display once: a headless mutter with a virtual monitor, its own Xwayland, a
# D-Bus session of its own and a private runtime dir (no desktop socket, audio or portal in it). The desktop session's
# DISPLAY, WAYLAND_DISPLAY and bus are unset first, so nothing started on it can reach the desktop. mutter runs a
# sleep that holds it up; display.hold records that sleep, whose exit ends mutter and its bus.
display_up() {
  local run=$ROOT/run
  if [ -s "$run/display.env" ] && owned "$(cat "$ROOT/display.hold" 2>/dev/null || echo 0)"; then
    return
  fi
  if ! command -v mutter >/dev/null || ! command -v dbus-run-session >/dev/null; then
    echo "the hidden display needs mutter and dbus-run-session" >&2
    exit 1
  fi
  mkdir -p "$run"
  chmod 700 "$run"
  rm -f "$run/display.env" "$run/display.x11"
  # mutter runs this inside its session, where DISPLAY names the Xwayland it started.
  cat >"$ROOT/display-hold.sh" <<'HOLD'
echo $$ >display.hold
printf '%s\n' "$DISPLAY" >"$XDG_RUNTIME_DIR/display.x11"
printf 'DISPLAY=%s\nWAYLAND_DISPLAY=%s\nDBUS_SESSION_BUS_ADDRESS=%s\nXAUTHORITY=%s\n' "$DISPLAY" "$WAYLAND_DISPLAY" "$DBUS_SESSION_BUS_ADDRESS" "${XAUTHORITY:-}" >"$XDG_RUNTIME_DIR/display.env.new"
mv "$XDG_RUNTIME_DIR/display.env.new" "$XDG_RUNTIME_DIR/display.env"
exec sleep infinity
HOLD
  (cd "$ROOT" && env -u DISPLAY -u WAYLAND_DISPLAY -u XAUTHORITY -u DBUS_SESSION_BUS_ADDRESS -u XDG_SESSION_TYPE -u PULSE_SERVER \
    XDG_RUNTIME_DIR="$run" nohup dbus-run-session -- mutter --wayland --headless --virtual-monitor 1280x720 \
    --wayland-display mortar-hidden -- sh "$ROOT/display-hold.sh" >"$ROOT/display.log" 2>&1 &)
  for _ in $(seq 1 100); do
    [ -s "$run/display.env" ] && return
    sleep 0.1
  done
  echo "the hidden display did not start; see $ROOT/display.log" >&2
  exit 1
}

# display_down ends the hidden display through the sleep it recorded, verified as running from the sandbox.
display_down() {
  local hold
  hold=$(cat "$ROOT/display.hold" 2>/dev/null || true)
  [ -n "$hold" ] && owned "$hold" || return 0
  kill "$hold" 2>/dev/null || true
  for _ in $(seq 1 50); do
    [ -e "$ROOT/run/$(sed -n 's/^WAYLAND_DISPLAY=//p' "$ROOT/run/display.env" 2>/dev/null)" ] || break
    sleep 0.1
  done
  rm -f "$ROOT/display.hold"
}

# hidden runs a command on the hidden display, with the launch guard's settings for any game it starts.
hidden() {
  local vars=()
  mapfile -t vars <"$ROOT/run/display.env"
  env -u DISPLAY -u WAYLAND_DISPLAY -u XAUTHORITY -u DBUS_SESSION_BUS_ADDRESS -u XDG_SESSION_TYPE -u PULSE_SERVER \
    XDG_RUNTIME_DIR="$ROOT/run" "${vars[@]}" MORTAR_HIDDEN_RUNTIME="$ROOT/run" \
    MORTAR_LAUNCH_WRAPPER="$REPO/scripts/launch-guard.sh" MORTAR_LAUNCH_CAP="$LAUNCH_CAP" \
    MORTAR_LAUNCH_COUNT="$LAUNCH_COUNT" MORTAR_LAUNCH_PIDS="$ROOT/launches.pids" "$@"
}

launches_used() { cat "$LAUNCH_COUNT" 2>/dev/null || echo 0; }

# need_launches N refuses a run that would start more games than the session has left, before it does anything.
need_launches() {
  local left=$((LAUNCH_CAP - $(launches_used)))
  if [ "$1" -gt "$left" ]; then
    echo "refused: this run starts $1 games and the session ($SESSION_KEY) has $left of $LAUNCH_CAP launches left ($LAUNCH_COUNT)" >&2
    exit 3
  fi
}

# launch_pid PID START prints the host pid of the recorded launch PID: the guard records the pid it sees, which inside
# the server's pid namespace is not the host's, so the host pid is the process with the same start time whose pid in
# its innermost namespace is PID. Nothing is printed when that process is gone.
launch_pid() {
  local d
  for d in /proc/[0-9]*; do
    [ "$(sed 's/.*) //' "$d/stat" 2>/dev/null | cut -d' ' -f20)" = "$2" ] || continue
    [ "$(awk '/^NSpid:/ {print $NF}' "$d/status" 2>/dev/null)" = "$1" ] && echo "${d#/proc/}" && return
  done
}

# stop_launches stops every game the guard recorded in this sandbox, each found again by its start time so a reused
# pid is never hit, then SIGKILLs what ignored SIGTERM.
stop_launches() {
  local pid start _ host live=()
  [ -f "$ROOT/launches.pids" ] || return 0
  while read -r pid start _; do
    host=$(launch_pid "$pid" "$start")
    [ -n "$host" ] && live+=("$host:$start")
  done <"$ROOT/launches.pids"
  [ ${#live[@]} -gt 0 ] || return 0
  local p
  for p in "${live[@]}"; do kill "${p%%:*}" 2>/dev/null || true; done
  sleep 2
  for p in "${live[@]}"; do
    [ "$(sed 's/.*) //' "/proc/${p%%:*}/stat" 2>/dev/null | cut -d' ' -f20)" = "${p#*:}" ] && kill -KILL "${p%%:*}" 2>/dev/null
  done
  echo "stopped recorded launches: ${live[*]%%:*}"
}

# Everything the sandbox starts (server, games, Chromium, Go builds) writes its scratch under the sandbox, so a run
# killed mid-way leaves nothing behind that destroy does not delete. On /var/tmp, like the sandbox, not tmpfs.
sandbox_tmp() {
  mkdir -p "$ROOT/tmp"
  export TMPDIR=$ROOT/tmp GOTMPDIR=$ROOT/tmp
}

mark() {
  mkdir -p "$ROOT"
  touch "$ROOT/$MARKER"
  sandbox_tmp
}

setup() {
  mark
  mkdir -p "$SANDBOX_STEAM/config"
  copy_game "$APP_ID" "$GAME_FOLDER" || exit 1
  [ -f "$SANDBOX_STEAM/config/loginusers.vdf" ] || cp "$STEAM/config/loginusers.vdf" "$SANDBOX_STEAM/config/"
  # Lethal Company and Valheim are optional: a machine without them still gets the Stardew sandbox.
  copy_game "$LC_APP_ID" "$LC_FOLDER" || true
  copy_game "$VH_APP_ID" "$VH_FOLDER" || true
  # An empty prefix is enough for runtime path resolution; the real one is never copied.
  local app folder
  for app in "$LC_APP_ID:$LC_FOLDER" "$VH_APP_ID:$VH_FOLDER"; do
    folder=${app#*:}
    if [ -d "$SANDBOX_STEAM/steamapps/common/$folder" ]; then
      mkdir -p "$SANDBOX_STEAM/steamapps/compatdata/${app%%:*}/pfx/drive_c/users/steamuser/AppData/LocalLow"
    fi
  done
  write_library
}

copy_data() {
  local real=${MORTAR_SELFTEST_DATA:-$HOME/.local/share/mortar} dest=$SANDBOX_HOME/.local/share/mortar
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
    # the unfinished shutdown as a crash, so wait for the process to exit and for the port to be free again.
    for _ in $(seq 1 75); do
      if ! kill -0 "$pid" 2>/dev/null && [ -z "$(listener || true)" ]; then
        echo "stopped $pid"
        stop_launches
        return
      fi
      sleep 0.2
    done
    echo "pid $pid did not exit and free port $PORT within 15s; not killing it (see $ROOT/server.log)" >&2
    exit 1
  elif [ -n "$pid" ]; then
    echo "port $PORT is held by pid $pid, which is not the self-test server; leaving it" >&2
    exit 1
  fi
  # The hidden display stays up for the next start; destroy takes it down with the sandbox.
  stop_launches
}

start() {
  local holder
  holder=$(listener || true)
  if [ -n "$holder" ]; then
    echo "port $PORT is already held by pid $holder; stop it or set MORTAR_SELFTEST_PORT" >&2
    exit 1
  fi
  mark
  # The host's steam run with the sandbox HOME brings up a second, signed-out Steam; the sandbox gets a steam that
  # refuses, so a Steam launch fails here and only a direct launch can start the copied game.
  mkdir -p "$ROOT/bin"
  printf '#!/bin/sh\necho "self-test sandbox: Steam is never started from here" >&2\nexit 1\n' >"$ROOT/bin/steam"
  chmod +x "$ROOT/bin/steam"
  # Opening a mod page or folder from the sandbox would land in the maintainer's own browser or file manager.
  local opener
  for opener in xdg-open x-www-browser www-browser gio; do
    printf '#!/bin/sh\necho "self-test sandbox: not opening $*" >&2\nexit 0\n' >"$ROOT/bin/$opener"
    chmod +x "$ROOT/bin/$opener"
  done
  # Mortar counts any Stardew Valley process it cannot place in another install as its own game running, which holds
  # back scheduled backups; one launched anywhere else on the machine (another sandbox, a QA copy, the real game)
  # would then stall the seed. A PID namespace with its own /proc shows the server only the sandbox's processes.
  # The namespace ends, taking every process in it, when its PID 1 exits, and the server restarts by starting itself
  # again and exiting (a data move, an update). So PID 1 is a reaper that runs the server and leaves only once no
  # process is left in the namespace.
  local isolate=()
  local reaper='import os, sys
if os.fork() == 0:
    os.execv(sys.argv[1], sys.argv[1:])
while True:
    try:
        pid, status = os.wait()
    except ChildProcessError:
        break
    # One line per reaped process, "<pid> <exit code>", negative for a signal: the evidence a sandbox that lost its
    # server leaves (frontend/e2e/evidence.ts).
    with open("server.exit", "a") as f:
        f.write(f"{pid} {os.waitstatus_to_exitcode(status)}\n")'
  if unshare --user --map-current-user --pid --fork --mount --mount-proc true 2>/dev/null; then
    isolate=(unshare --user --map-current-user --pid --fork --mount --mount-proc python3 -c "$reaper")
  fi
  # The server, and every game it starts, sees only the hidden display.
  display_up
  (cd "$ROOT" && hidden env -u XDG_DATA_HOME -u XDG_CONFIG_HOME -u XDG_CACHE_HOME HOME="$SANDBOX_HOME" PATH="$ROOT/bin:$PATH" \
    WAILS_SERVER_HOST=127.0.0.1 WAILS_SERVER_PORT="$PORT" nohup "${isolate[@]}" ./mortar-server >"$ROOT/server.log" 2>&1 &)
  for _ in $(seq 1 300); do
    if [ -n "$(listener || true)" ]; then
      # Recorded so a test run that dies can have its server stopped by pid later (frontend/e2e/sandbox.ts).
      listener >"$ROOT/server.pid"
      echo "self-test server on http://127.0.0.1:$PORT (pid $(listener), log $ROOT/server.log)"
      return
    fi
    sleep 0.1
  done
  echo "server did not start; see $ROOT/server.log" >&2
  exit 1
}

# mortar runs the sandbox's own command line, so the fixtures go through the same commands an agent would use.
cli() {
  env -u XDG_DATA_HOME -u XDG_CONFIG_HOME -u XDG_CACHE_HOME HOME="$SANDBOX_HOME" "$ROOT/mortar-server" "$@"
}

# make_mod DIR ID NAME writes a tiny content-pack mod whose manifest is valid and needs no network.
make_mod() {
  mkdir -p "$1"
  printf '{"Name":"%s","Author":"Self-test","Version":"1.0.0","Description":"Fixture mod","UniqueID":"%s","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}\n' "$3" "$2" >"$1/manifest.json"
}

# seed_lc fills a Lethal Company profile with two Thunderstore-shaped packages (manifest.json, a placeholder plugin
# dll and a BepInEx config file) when the sandbox holds the game; it is skipped when the profile exists.
seed_lc() {
  cli games --json | grep -q '"id": "lethal-company"' || return 0
  if cli profiles lethal-company --json | grep -q '"Seed Lobby"'; then
    return
  fi
  local fx=$ROOT/seed-lc name
  rm -rf "$fx"
  for name in SeedAlpha SeedBeta; do
    mkdir -p "$fx/$name/plugins" "$fx/$name/config" "$fx/zips"
    printf '{"name":"%s","version_number":"1.0.0","author":"Self-test","website_url":"","description":"Fixture package","dependencies":[]}\n' "$name" >"$fx/$name/manifest.json"
    printf 'placeholder %s\n' "$name" >"$fx/$name/plugins/$name.dll"
    if [ "$name" = SeedAlpha ]; then
      cp "$REPO/internal/dotnet/testdata/e2e/Alpha.dll" "$fx/$name/plugins/$name.dll"
    fi
    printf '[General]\nEnabled = true\n' >"$fx/$name/config/Self-test.$name.cfg"
    (cd "$fx/$name" && python3 -m zipfile -c "$fx/zips/$name.zip" ./*)
  done
  cli profile create lethal-company "Seed Lobby"
  for name in SeedAlpha SeedBeta; do
    cli install lethal-company "Seed Lobby" "$fx/zips/$name.zip"
  done
  seed_lc_configs "$fx"
}

# seed_lc_configs gives the Lethal Company profile what the Config page lists: SeedAlpha's plugin (the repo's
# declared-plugin test assembly, com.e2e.alpha, copied in at seed time) owns a .cfg that differs from its default, and
# BepInEx.cfg belongs to no mod.
seed_lc_configs() {
  local fx=$1 pid cfg
  pid=$(cli profiles lethal-company --json | python3 -c 'import json,sys; print(next(p["id"] for p in json.load(sys.stdin) if p["name"] == "Seed Lobby"))')
  cfg=$SANDBOX_HOME/.local/share/mortar/profiles/lethal-company/$pid/BepInEx/config
  mkdir -p "$cfg"
  printf '## Settings file was created by plugin E2E Alpha v1.0.0\n## Plugin GUID: com.e2e.alpha\n\n[General]\n\n## Turns the feature on.\n# Setting type: Boolean\n# Default value: true\nEnabled = false\n' >"$cfg/com.e2e.alpha.cfg"
  printf '[Logging.Console]\n\n## Enables showing a console for log output.\n# Setting type: Boolean\n# Default value: false\nEnabled = true\n' >"$cfg/BepInEx.cfg"
}

# seed_gmcm writes what the bridge leaves for Seed.Beta in the profile folder dir: an in-game menu capture, one edit
# waiting for the next start and the result of an earlier apply (shapes: internal/gmcm).
seed_gmcm() {
  python3 - "$1" <<'PY'
import json, os, sys
root = sys.argv[1]
os.makedirs(os.path.join(root, "gmcm"), exist_ok=True)
os.makedirs(os.path.join(root, "gmcm-pending"), exist_ok=True)
option = lambda i, kind, field, name, value, **more: {"index": i, "kind": kind, "fieldId": field, "name": name, "tooltip": "", "value": value, "editable": True, "titleScreenOnly": False, **more}
capture = {"schema": 1, "mod": {"id": "Seed.Beta", "name": "Seed Beta", "version": "1.0.0"}, "gmcmVersion": "1.12.0",
           "capturedAt": "2026-01-02T03:04:05.0000000+00:00", "titleScreenOnlyDefault": False,
           "pages": [{"id": "", "options": [option(0, "bool", "enabled", "Enabled", True),
                                            option(1, "int", "count", "Count", 3, min=1, max=9, interval=1)]}]}
edit = {"page": "", "index": 1, "kind": "int", "fieldId": "count", "name": "Count", "value": 5}
json.dump(capture, open(os.path.join(root, "gmcm", "Seed.Beta.json"), "w"), indent=2)
json.dump({"schema": 1, "edits": [edit]}, open(os.path.join(root, "gmcm-pending", "Seed.Beta.json"), "w"), indent=2)
json.dump({"schema": 1, "applied": [{**edit, "value": 4}], "skipped": []}, open(os.path.join(root, "gmcm-pending", "Seed.Beta.result.json"), "w"), indent=2)
PY
}

seed() {
  [ -n "$(listener || true)" ] || {
    echo "start the server first" >&2
    exit 1
  }
  if cli profiles stardew --json | grep -q '"Seed Farm"'; then
    seed_lc
    echo "sandbox already seeded; leaving it"
    return
  fi
  local fx=$ROOT/seed game="$SANDBOX_STEAM/steamapps/common/$GAME_FOLDER"
  rm -rf "$fx"
  mkdir -p "$fx/zips" "$ROOT/extra"
  make_mod "$fx/alpha/Seed.Alpha" Seed.Alpha "Seed Alpha"
  printf '{"Speed":1,"Enabled":true}\n' >"$fx/alpha/Seed.Alpha/config.json"
  make_mod "$fx/beta/Seed.Beta" Seed.Beta "Seed Beta"
  # A pack folder with no manifest of its own is what lets a dotted sibling count as hidden rather than as the mod.
  make_mod "$fx/gamma/Seed.Pack/Seed.Gamma" Seed.Gamma "Seed Gamma"
  make_mod "$fx/gamma/Seed.Pack/.Seed.Hidden" Seed.Hidden "Seed Hidden"
  local m
  for m in alpha beta gamma; do
    (cd "$fx/$m" && python3 -m zipfile -c "$fx/zips/$m.zip" ./*)
  done
  make_mod "$ROOT/extra/ExtraOne" Seed.ExtraOne "Seed Extra One"
  make_mod "$ROOT/extra/ExtraTwo" Seed.ExtraTwo "Seed Extra Two"
  make_mod "$game/Mods/SeedStray" Seed.Stray "Seed Stray"
  mkdir -p "$SANDBOX_HOME/.config/StardewValley/Saves/Seed_123456"
  echo '<SaveGame/>' | tee "$SANDBOX_HOME/.config/StardewValley/Saves/Seed_123456/"{Seed_123456,SaveGameInfo} >/dev/null

  cli settings set --game stardew showDotHiddenMods true
  cli settings set --game stardew extraModsFolder "$ROOT/extra"
  cli profile create stardew "Seed Farm"
  for m in alpha beta gamma; do
    cli install stardew "Seed Farm" "$fx/zips/$m.zip"
  done
  cli mods config stardew "Seed Farm" Seed.Alpha Speed 5
  cli mods disable stardew "Seed Farm" Seed.Beta
  cli mods enable stardew "Seed Farm" Seed.Beta
  cli templates save stardew "Seed Farm" "Seed Template"
  cli templates new stardew "Seed Template" "Seed From Template"
  cli backups create --game stardew Seed_123456

  # A failed download has no command that makes one, so it is written where the queue keeps its history.
  local pid data=$SANDBOX_HOME/.local/share/mortar
  pid=$(cli profiles stardew --json | python3 -c 'import json,sys; print(next(p["id"] for p in json.load(sys.stdin) if p["name"] == "Seed Farm"))')
  python3 - "$data/download-history.json" "$pid" <<'PY'
import json, sys, time
path, profile = sys.argv[1:]
now = int(time.time())
entry = {"name": "Seed Failed Download", "version": "1.0.0", "source": "nexus", "profileId": profile, "game": "stardew",
         "modId": 1, "fileId": 1, "kind": "mod", "size": 2048, "started": now - 5, "finished": now,
         "outcome": "failed", "error": "[network] connection reset"}
json.dump([entry], open(path, "w"))
PY

  seed_gmcm "$data/profiles/stardew/$pid"

  # The scheduled backup runs on the server's first check after start, so enable it and restart without rebuilding.
  cli settings set --game stardew saveBackupHours 1
  stop
  start
  # stop waits for the old server to exit; the new one's first check runs at start and logs its pass.
  for _ in $(seq 1 300); do
    if cli backups list --game stardew --json | grep -q scheduled; then
      seed_lc
      echo "sandbox seeded"
      return
    fi
    sleep 0.1
  done
  if ! grep -q "scheduled save backup:" "$ROOT/server.log"; then
    echo "the scheduled backup pass never ran: Mortar took the game for running (cli status stardew)" >&2
  fi
  echo "the scheduled backup never appeared; see $ROOT/server.log" >&2
  exit 1
}

# tree_hash prints one line per entry of DIR, sorted by path: type, mode, path, and the sha256 of a file or the target of a link.
tree_hash() {
  python3 - "$1" <<'PY'
import hashlib, os, stat, sys
root = sys.argv[1]
rows = []
for dirpath, dirnames, filenames in os.walk(root):
    for name in dirnames + filenames:
        path = os.path.join(dirpath, name)
        st = os.lstat(path)
        rel = "./" + os.path.relpath(path, root)
        mode = format(stat.S_IMODE(st.st_mode), "o")
        if stat.S_ISLNK(st.st_mode):
            rows.append(("l", mode, rel, os.readlink(path)))
        elif stat.S_ISDIR(st.st_mode):
            rows.append(("d", mode, rel, ""))
        else:
            h = hashlib.sha256()
            with open(path, "rb") as f:
                for chunk in iter(lambda: f.read(1 << 20), b""):
                    h.update(chunk)
            rows.append(("f", mode, rel, h.hexdigest()))
for row in sorted(rows, key=lambda r: r[2]):
    print(" ".join(row))
PY
}

# game_pids prints the pids whose executable lives in the sandbox's game folder, verified through /proc/PID/exe.
game_pids() {
  local dir=$1 p exe
  for p in /proc/[0-9]*; do
    exe=$(readlink "$p/exe" 2>/dev/null || true)
    case "$exe" in "$dir"/*) echo "${p#/proc/}" ;; esac
  done
}

# sandbox_pids prints the pids of the processes running from $ROOT: an executable or working directory inside it (the
# server, its namespace reaper, the copied game and its Wine processes). This script's own pid is never one of them.
# One find instead of a readlink per process: reap runs this once per sandbox over every process on the machine.
sandbox_pids() {
  find /proc -mindepth 2 -maxdepth 2 \( -name exe -o -name cwd \) \
    \( -lname "$ROOT" -o -lname "$ROOT/*" -o -lname "$ROOT (deleted)" \) 2>/dev/null |
    cut -d/ -f3 | grep -vx "$$" | sort -u || true
}

# release_sandbox stops the server, then everything else still running from $ROOT (SIGTERM, then SIGKILL, each pid
# looked up again first), and deletes $ROOT only after a pass and only once nothing runs from it: a server whose folder
# is gone keeps running unseen. Anything left running keeps the folder and is named.
release_sandbox() {
  (stop) >/dev/null 2>&1 || true
  # stop leaves early when another program holds the port; the games and the display still go.
  stop_launches >/dev/null 2>&1 || true
  display_down
  local left=() i
  for i in $(seq 1 30); do
    mapfile -t left < <(sandbox_pids)
    [ ${#left[@]} -eq 0 ] && break
    [ "$i" = 10 ] && kill "${left[@]}" 2>/dev/null
    [ "$i" = 20 ] && kill -KILL "${left[@]}" 2>/dev/null
    sleep 1
  done
  if [ ${#left[@]} -gt 0 ]; then
    echo "sandbox kept, still running from it: pids ${left[*]} ($ROOT)" >&3
  elif [ "$verdict" = PASS ]; then
    # A gvfsd-fuse killed with the sandbox leaves its mount dead, and rm cannot cross it.
    fusermount3 -u -z "$ROOT/run/gvfs" 2>/dev/null || true
    rm -rf "$ROOT"
  else
    echo "sandbox kept for inspection: $ROOT" >&3
  fi
}

# regress_traps makes an interrupt or kill run the EXIT trap, which bash skips on a fatal signal; a failing command under
# set -e already exits through it. A trap can fire inside a command's redirection, so the trap's messages go to fd 3, the
# script's own stderr.
regress_traps() {
  exec 3>&2
  trap finish EXIT
  trap 'exit 129' HUP
  trap 'exit 130' INT
  trap 'exit 143' TERM
}

regress() {
  ROOT=$(mktemp -d /var/tmp/mortar-regress-XXXXXX)
  case "$ROOT" in /var/tmp/mortar-regress-??????) ;; *)
    echo "unexpected sandbox dir $ROOT" >&2
    exit 1
    ;;
  esac
  # A failed run keeps its folder for the logs; the marker lets reap collect it later.
  mark
  PORT=$((9600 + RANDOM % 300))
  while [ -n "$(ss -ltn "sport = :$PORT" | tail -n +2)" ]; do PORT=$((9600 + RANDOM % 300)); done
  SANDBOX_HOME=$ROOT/home
  SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
  # game and verdict are read by the EXIT trap, after this function's locals are gone.
  game="$SANDBOX_STEAM/steamapps/common/$GAME_FOLDER"
  gpids=""
  verdict=FAIL
  local timeout=${MORTAR_REGRESS_TIMEOUT:-180}
  local log="$SANDBOX_HOME/.config/StardewValley/ErrorLogs/SMAPI-latest.txt" t0=$SECONDS
  local failures=() mods_n="" packs_n="" skips=0 enabled="" profile="" diff_lines=0

  finish() { release_sandbox; }
  regress_traps

  need_launches 1
  setup
  copy_data
  build
  start
  cli settings set --game stardew defaultLaunchMethod direct >/dev/null
  profile=${MORTAR_REGRESS_PROFILE:-$(cli profiles stardew --json | python3 -c 'import json,sys; print(json.load(sys.stdin)[0]["id"])')}
  enabled=$(cli mods stardew "$profile" --json | python3 -c 'import json,sys; print(sum(1 for m in json.load(sys.stdin) if m["enabled"]))')
  tree_hash "$game" >"$ROOT/game-before.txt"

  echo "launching profile $profile ($enabled enabled)"
  cli launch stardew "$profile" >"$ROOT/launch.txt" 2>&1 || failures+=("launch failed: $(head -c 300 "$ROOT/launch.txt")")
  local deadline=$((SECONDS + timeout))
  while [ ${#failures[@]} -eq 0 ] && [ "$SECONDS" -lt "$deadline" ]; do
    if [ -f "$log" ]; then
      mods_n=$(sed -n 's/.*SMAPI\] *Loaded \([0-9]*\) mods:.*/\1/p' "$log" | head -1)
      packs_n=$(sed -n 's/.*SMAPI\] *Loaded \([0-9]*\) content packs:.*/\1/p' "$log" | head -1)
      [ -n "$mods_n" ] && [ -n "$packs_n" ] && break
    fi
    sleep 2
  done
  gpids=$(game_pids "$game" | tr '\n' ' ')
  if [ -z "$mods_n" ] || [ -z "$packs_n" ]; then
    failures+=("SMAPI did not log its mod and content pack counts within ${timeout}s")
  else
    # A mod SMAPI names as skipped ("- Name because reason") never loads, so it is not owed to the count.
    skips=$(grep -cE 'SMAPI\] +- .+ because ' "$log" || true)
    if [ $((mods_n + packs_n + skips)) -ne "$enabled" ]; then
      failures+=("loaded $mods_n mods + $packs_n content packs + $skips skipped != $enabled enabled")
    fi
  fi
  [ -n "$gpids" ] && echo "$gpids" >"$ROOT/game.pid"

  echo "stopping the game (pids: ${gpids:-none})"
  local pid
  for pid in $gpids; do
    case "$(readlink "/proc/$pid/exe" 2>/dev/null)" in "$game"/*) kill "$pid" 2>/dev/null || true ;; esac
  done
  for _ in $(seq 1 60); do
    [ "$(cli status stardew --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])')" = idle ] && break
    sleep 1
  done
  for pid in $gpids; do
    case "$(readlink "/proc/$pid/exe" 2>/dev/null)" in "$game"/*) kill -KILL "$pid" 2>/dev/null || true ;; esac
  done
  [ "$(cli status stardew --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])')" = idle ] || failures+=("Mortar never went idle after the game stopped")
  sleep 3
  tree_hash "$game" >"$ROOT/game-after.txt"
  diff_lines=$(diff "$ROOT/game-before.txt" "$ROOT/game-after.txt" | grep -c '^[<>]' || true)
  if [ "$diff_lines" -ne 0 ]; then
    failures+=("game folder differs after purge ($diff_lines lines); see $ROOT/game-before.txt vs game-after.txt")
  fi

  [ ${#failures[@]} -eq 0 ] && verdict=PASS
  echo "---- regress: $verdict ($((SECONDS - t0))s)"
  echo "profile        $profile"
  echo "enabled        $enabled"
  echo "SMAPI loaded   ${mods_n:-?} mods + ${packs_n:-?} content packs (+ $skips skipped)"
  echo "game entries   $(wc -l <"$ROOT/game-before.txt") hashed, $diff_lines differing after purge"
  local f
  for f in "${failures[@]}"; do echo "FAIL: $f"; done
  [ "$verdict" = PASS ]
}

# reap_prefix stops the processes that run in the Lethal Company prefix at $1 (a compatdata folder), each verified
# through /proc/PID/environ: a Wine process names the prefix in WINEPREFIX, Proton's own script in STEAM_COMPAT_DATA_PATH.
# The game's executable goes first so Mortar sees it exit while the wineserver still answers.
reap_prefix() {
  local compat=$1 d p pids=() first=()
  for d in /proc/[0-9]*; do
    p=${d#/proc/}
    if { tr '\0' '\n' <"$d/environ" | grep -qxE "(WINEPREFIX=${compat}/pfx/?|STEAM_COMPAT_DATA_PATH=${compat}/?)"; } 2>/dev/null; then
      if tr '\0' ' ' <"$d/cmdline" 2>/dev/null | grep -q 'Lethal Company.exe' && ! tr '\0' ' ' <"$d/cmdline" | grep -q 'steam.exe'; then
        first+=("$p")
      else
        pids+=("$p")
      fi
    fi
  done
  [ ${#first[@]} -gt 0 ] && kill "${first[@]}" 2>/dev/null
  sleep 2
  [ ${#pids[@]} -gt 0 ] && kill "${pids[@]}" 2>/dev/null
  sleep 2
  for p in "${first[@]}" "${pids[@]}"; do
    # A process that ignored SIGTERM is checked against its prefix again before it is killed.
    if { tr '\0' '\n' <"/proc/$p/environ" | grep -qxE "(WINEPREFIX=${compat}/pfx/?|STEAM_COMPAT_DATA_PATH=${compat}/?)"; } 2>/dev/null &&
      ! grep -q ') Z' "/proc/$p/stat" 2>/dev/null; then
      kill -KILL "$p" 2>/dev/null || true
    fi
  done
  echo "${first[*]} ${pids[*]}"
}

# regress_lc is regress for Lethal Company: a direct Proton launch of a BepInEx profile with three Thunderstore plugins,
# in its own sandbox. Steam is never started; Proton only needs Steam's client library and the .steam links, both
# copied into the sandbox home. Every Proton run has a network namespace of its own: steamclient.so reaches a running
# Steam over loopback TCP whatever HOME says, and the game then registers with it under its Wine pid, which on the host
# names a kernel thread that never exits, so Steam waits for it to shut down until Steam itself is killed.
# The Thunderstore index and downloads are cached in $REGRESS_CACHE after the first run.
# regress_launches prints how many games a Lethal Company regress with the current environment starts: the base run's,
# each of the matrix's and the r2 step's.
regress_launches() {
  local n=1
  [ -n "${MORTAR_REGRESS_R2_CODE:-}" ] && n=$((n + 1))
  [ -n "${MORTAR_REGRESS_MATRIX:-}" ] && n=$((n + $(mx_launches)))
  echo "$n"
}

regress_lc() {
  if [ -n "${MORTAR_REGRESS_R2_CODE:-}" ] && [ "$(regress_launches)" -gt "$LAUNCH_CAP" ]; then
    echo "refused: MORTAR_REGRESS_R2_CODE needs launch $(regress_launches), but a session allows $LAUNCH_CAP and the base run and the matrix use $(($(regress_launches) - 1)) of them; unset MORTAR_REGRESS_R2_CODE or MORTAR_REGRESS_MATRIX" >&2
    exit 3
  fi
  ROOT=$(mktemp -d /var/tmp/mortar-regress-lc-XXXXXX)
  case "$ROOT" in /var/tmp/mortar-regress-lc-??????) ;; *)
    echo "unexpected sandbox dir $ROOT" >&2
    exit 1
    ;;
  esac
  # A failed run keeps its folder for the logs; the marker lets reap collect it later.
  mark
  PORT=$((9600 + RANDOM % 300))
  while [ -n "$(ss -ltn "sport = :$PORT" | tail -n +2)" ]; do PORT=$((9600 + RANDOM % 300)); done
  SANDBOX_HOME=$ROOT/home
  SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
  verdict=FAIL
  game="$SANDBOX_STEAM/steamapps/common/$LC_FOLDER"
  compat="$SANDBOX_STEAM/steamapps/compatdata/$LC_APP_ID"
  local proton=${MORTAR_REGRESS_PROTON:-$STEAM/steamapps/common/Proton - Experimental}
  local cache=${MORTAR_REGRESS_CACHE:-/var/tmp/mortar-regress-cache}/lc-data
  local data=$SANDBOX_HOME/.local/share/mortar
  local timeout=${MORTAR_REGRESS_TIMEOUT:-240} t0=$SECONDS
  local bepinex=${MORTAR_REGRESS_BEPINEX:-5.4.2305}
  local plugins=(LethalConfig ShipLoot MoreCompany)
  # Pinned so the cached zips stay the ones the run was written against.
  local packages=(AinaVT/LethalConfig/1.4.6 tinyhoot/ShipLoot/1.1.0 notnotnotswipez/MoreCompany/1.14.0)
  local failures=() profile="" loaded=0 diff_lines=0 log="" killed=""
  local gdirs=(cache/thunderstore store)

  finish() {
    reap_prefix "$compat" >/dev/null 2>&1 || true
    release_sandbox
  }
  regress_traps

  need_launches "$(regress_launches)"
  [ -d "$proton" ] || {
    echo "no Proton at $proton (set MORTAR_REGRESS_PROTON)" >&2
    exit 1
  }
  [ -f "$STEAM/linux64/steamclient.so" ] || {
    echo "no steamclient.so under $STEAM" >&2
    exit 1
  }
  command -v bwrap >/dev/null || {
    echo "bwrap (bubblewrap) is needed to keep the game away from a running Steam" >&2
    exit 1
  }
  if [ -n "${MORTAR_REGRESS_OFFLINE:-}" ]; then
    export HTTPS_PROXY=http://127.0.0.1:9 HTTP_PROXY=http://127.0.0.1:9 NO_PROXY=127.0.0.1
  fi

  mkdir -p "$SANDBOX_STEAM/config"
  copy_game "$LC_APP_ID" "$LC_FOLDER" || exit 1
  [ -f "$STEAM/config/loginusers.vdf" ] && cp "$STEAM/config/loginusers.vdf" "$SANDBOX_STEAM/config/"
  write_library
  # Proton loads Steam's client library from the Steam folder it is told about, which is the sandbox's own.
  mkdir -p "$SANDBOX_STEAM/linux64" "$SANDBOX_STEAM/linux32" "$SANDBOX_HOME/.steam"
  cp "$STEAM/linux64/steamclient.so" "$SANDBOX_STEAM/linux64/"
  cp "$STEAM/linux32/steamclient.so" "$SANDBOX_STEAM/linux32/"
  ln -sfn "$SANDBOX_STEAM" "$SANDBOX_HOME/.steam/steam"
  ln -sfn "$SANDBOX_STEAM" "$SANDBOX_HOME/.steam/root"
  ln -sfn "$SANDBOX_STEAM/linux64" "$SANDBOX_HOME/.steam/sdk64"
  ln -sfn "$SANDBOX_STEAM/linux32" "$SANDBOX_HOME/.steam/sdk32"

  # Profiles get a local build of the bridge checkout beside this repo (MORTAR_REGRESS_BRIDGE_REPO names another), handed
  # to the server through MORTAR_LOCAL_BRIDGES, so a run tests the bridge as it is before its release; without a
  # checkout they get the catalog's published bridge, which an offline run cannot download.
  local bridge=expected bridge_repo=${MORTAR_REGRESS_BRIDGE_REPO:-$REPO/../mortar-bepinex-bridge}
  [ -n "${MORTAR_REGRESS_OFFLINE:-}" ] && bridge=skipped
  if [ -x "$bridge_repo/scripts/package.sh" ]; then
    local bridge_dir=$ROOT/bridge built
    mkdir -p "$bridge_dir"
    if built=$(DIST="$ROOT/bridge-dist" "$bridge_repo/scripts/package.sh" 2>"$ROOT/bridge-build.log" | tail -1) && [ -f "$built" ]; then
      cp "$built" "$bridge_dir/lethal-company.zip"
      export MORTAR_LOCAL_BRIDGES=$bridge_dir
      bridge=expected
    else
      failures+=("the bridge did not build; see $ROOT/bridge-build.log")
    fi
  elif [ -n "${MORTAR_REGRESS_BRIDGE_REPO:-}" ]; then
    failures+=("no bridge checkout at $bridge_repo")
  fi
  mkdir -p "$ROOT"
  cat >"$ROOT/run-proton.sh" <<EOF
#!/bin/bash
export STEAM_COMPAT_DATA_PATH='$compat'
export STEAM_COMPAT_CLIENT_INSTALL_PATH='$SANDBOX_STEAM'
export STEAM_COMPAT_APP_ID=$LC_APP_ID SteamAppId=$LC_APP_ID SteamGameId=$LC_APP_ID
exec bwrap --dev-bind / / --unshare-net -- '$proton/proton' run "\$@"
EOF
  chmod +x "$ROOT/run-proton.sh"
  # The first Proton run creates the prefix; Mortar can only add its winhttp override to a prefix that exists.
  echo "creating the Proton prefix"
  mkdir -p "$compat"
  display_up
  (cd "$ROOT" && hidden env HOME="$SANDBOX_HOME" timeout 300 "$ROOT/run-proton.sh" wineboot -u >"$ROOT/wineboot.log" 2>&1) ||
    failures+=("the Proton prefix could not be created; see $ROOT/wineboot.log")
  reap_prefix "$compat" >/dev/null

  build
  start
  if [ -d "$cache/store" ]; then
    local d
    for d in "${gdirs[@]}"; do [ -d "$cache/$d" ] && mkdir -p "$data/$(dirname "$d")" && cp -a "$cache/$d" "$data/$(dirname "$d")/"; done
  fi
  cli settings set --game lethal-company defaultLaunchMethod direct >/dev/null
  profile=$(cli profile create lethal-company "Regress LC" | cut -f1)
  # The server may already be installing the loader for the new game; a second install waits its turn.
  for _ in $(seq 1 30); do
    cli loader install lethal-company "$bepinex" >"$ROOT/loader.txt" 2>&1 && break
    grep -q 'already running' "$ROOT/loader.txt" || break
    sleep 2
  done
  grep -q "^Installed loader $bepinex" "$ROOT/loader.txt" || failures+=("BepInEx $bepinex did not install: $(head -c 300 "$ROOT/loader.txt")")
  local pkg
  # A pinned loader is not looked up again at launch.
  cli loader pin lethal-company "$bepinex" >/dev/null 2>&1 || failures+=("pinning BepInEx $bepinex failed")
  local zip
  mkdir -p "$cache/zips"
  for pkg in "${packages[@]}"; do
    zip=$cache/zips/${pkg//\//-}.zip
    [ -s "$zip" ] || curl -fsSL -o "$zip" "https://thunderstore.io/package/download/$pkg/" || failures+=("downloading $pkg failed")
    cli install lethal-company "$profile" "$zip" >>"$ROOT/install.txt" 2>&1 || failures+=("installing $pkg failed: $(tail -c 200 "$ROOT/install.txt")")
  done
  cli profile set lethal-company "$profile" launchPrefix "$ROOT/run-proton.sh" >/dev/null
  if [ ! -d "$cache/store" ] && [ ${#failures[@]} -eq 0 ]; then
    for d in "${gdirs[@]}"; do [ -d "$data/$d" ] && mkdir -p "$cache/$(dirname "$d")" && cp -a "$data/$d" "$cache/$(dirname "$d")/"; done
  fi
  tree_hash "$game" >"$ROOT/game-before.txt"

  # lc_launch launches profile $1 and waits for BepInEx to finish loading; the run's log path is left in $log.
  lc_launch() {
    log="$data/profiles/lethal-company/$1/BepInEx/LogOutput.log"
    if [ ${#failures[@]} -eq 0 ]; then
      cli launch lethal-company "$1" >"$ROOT/launch-$2.txt" 2>&1 || failures+=("$2 launch failed: $(head -c 300 "$ROOT/launch-$2.txt")")
    fi
    deadline=$((SECONDS + timeout))
    while [ ${#failures[@]} -eq 0 ] && [ "$SECONDS" -lt "$deadline" ]; do
      grep -q 'Chainloader startup complete' "$log" 2>/dev/null && break
      sleep 2
    done
    if [ ${#failures[@]} -eq 0 ] && ! grep -q 'Chainloader startup complete' "$log" 2>/dev/null; then
      failures+=("$2: BepInEx never logged \"Chainloader startup complete\" within ${timeout}s ($(cli status lethal-company 2>&1 | head -c 200))")
    fi
  }
  # lc_stop stops the game through Mortar, then reaps whatever is left of the prefix by verified pid, waits for Mortar
  # to go idle (which purges the deploy) and requires the game folder to hash as it did before the first launch.
  lc_stop() {
    cli stop lethal-company >"$ROOT/stop-$1.txt" 2>&1 || failures+=("$1: mortar stop failed: $(head -c 300 "$ROOT/stop-$1.txt")")
    killed="$killed $(reap_prefix "$compat")"
    local state=""
    for _ in $(seq 1 60); do
      state=$(cli status lethal-company --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])')
      [ "$state" = idle ] && break
      sleep 1
    done
    [ "$state" = idle ] || failures+=("$1: Mortar never went idle after the game stopped")
    sleep 3
    tree_hash "$game" >"$ROOT/game-after-$1.txt"
    local n
    n=$(diff "$ROOT/game-before.txt" "$ROOT/game-after-$1.txt" | grep -c '^[<>]' || true)
    diff_lines=$((diff_lines + n))
    [ "$n" -eq 0 ] || failures+=("$1: game folder differs after purge ($n lines); see $ROOT/game-before.txt vs game-after-$1.txt")
    cp "$log" "$ROOT/LogOutput-$1.log" 2>/dev/null || true
  }

  # The matrix fills the base profile first, so the base launch is also its launch (a).
  local matrix=skipped
  if [ -n "${MORTAR_REGRESS_MATRIX:-}" ]; then
    if [ -n "${MORTAR_REGRESS_OFFLINE:-}" ]; then
      failures+=("the matrix needs the network; unset MORTAR_REGRESS_OFFLINE")
    else
      matrix=run
      regress_bepinex_prepare "$data" "$compat" "$game" "$timeout" "$profile" "$bepinex"
    fi
  fi

  echo "launching profile $profile (${#plugins[@]} plugins)"
  lc_launch "$profile" base
  local name
  for name in "${plugins[@]}"; do
    if [ ${#failures[@]} -gt 0 ]; then break; fi
    if grep -qE "BepInEx\] Loading \[$name " "$log" 2>/dev/null; then loaded=$((loaded + 1)); else failures+=("plugin $name did not load"); fi
  done

  if [ "$bridge" = expected ] && [ ${#failures[@]} -eq 0 ]; then
    if grep -q 'BepInEx\] Loading \[Mortar BepInEx Bridge ' "$log" && grep -q 'Listening on 127.0.0.1:' "$log"; then
      bridge=loaded
    else
      failures+=("Mortar bridge did not load")
    fi
  fi

  echo "stopping the game"
  if [ "$matrix" = run ]; then
    regress_bepinex_running "$log"
    mx_stop a
  else
    lc_stop base
  fi

  # MORTAR_REGRESS_R2_CODE imports an r2modman code into a new profile through `mortar profile import`, waits for its
  # downloads, launches it and requires BepInEx to load every plugin it counted. MORTAR_REGRESS_R2_DISABLE lists
  # package ids (Namespace-Name) to switch off before the launch.
  local code=${MORTAR_REGRESS_R2_CODE:-} r2=skipped r2_profile="" r2_listed=0 r2_mods=0 r2_plugins="" r2_loading=0 r2_errors=0
  if [ -n "$code" ] && [ ${#failures[@]} -eq 0 ]; then
    r2=run
    cli profile import "$code" --preview --json >"$ROOT/r2-preview.json" 2>&1 || failures+=("r2 preview failed: $(head -c 300 "$ROOT/r2-preview.json")")
    r2_listed=$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))["packages"]))' "$ROOT/r2-preview.json" 2>/dev/null || echo 0)
    if cli profile import "$code" --game lethal-company --name "Regress R2" --json >"$ROOT/r2-import.json" 2>&1; then
      r2_profile=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["profile"])' "$ROOT/r2-import.json")
    else
      failures+=("r2 import failed: $(head -c 300 "$ROOT/r2-import.json")")
    fi
    echo "imported $code into $r2_profile; waiting for its downloads"
    local pending=1
    deadline=$((SECONDS + 600))
    while [ -n "$r2_profile" ] && [ "$SECONDS" -lt "$deadline" ]; do
      cli queue --json >"$ROOT/r2-queue.json"
      pending=$(
        python3 - "$ROOT/r2-queue.json" "$r2_profile" 2>"$ROOT/r2-queue-open.txt" <<'PY'
import json, sys
items = [i for i in json.load(open(sys.argv[1]))["items"] if i["profileId"] == sys.argv[2]]
print(sum(1 for i in items if i["state"] not in ("done", "failed", "skipped", "cancelled")))
for i in items:
    if i["state"] != "done":
        print(f'{i["name"]} {i["version"]}: {i["state"]} {i["error"]}', file=sys.stderr)
PY
      )
      [ "$pending" = 0 ] && break
      sleep 2
    done
    [ "$pending" = 0 ] || failures+=("r2 downloads never finished: $(head -c 300 "$ROOT/r2-queue-open.txt")")
    [ -s "$ROOT/r2-queue-open.txt" ] && failures+=("r2 downloads failed: $(head -c 600 "$ROOT/r2-queue-open.txt")")
    if [ -n "$r2_profile" ]; then
      local off
      for off in ${MORTAR_REGRESS_R2_DISABLE:-}; do
        cli mods disable lethal-company "$r2_profile" "thunderstore:$off" >>"$ROOT/r2-disable.txt" 2>&1 || failures+=("disabling $off failed: $(tail -c 200 "$ROOT/r2-disable.txt")")
      done
      cli mods lethal-company "$r2_profile" --json >"$ROOT/r2-mods.json"
      r2_mods=$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))))' "$ROOT/r2-mods.json")
      cli profile set lethal-company "$r2_profile" launchPrefix "$ROOT/run-proton.sh" >/dev/null
      echo "launching profile $r2_profile ($r2_mods mods)"
      lc_launch "$r2_profile" r2
      r2_plugins=$(grep -oE 'BepInEx\] [0-9]+ plugins? to load' "$log" 2>/dev/null | grep -oE '[0-9]+' | head -1)
      r2_loading=$(grep -cE 'BepInEx\] Loading \[' "$log" 2>/dev/null || true)
      # A plugin's own errors (a transpiler that does not match the game) are the mod's, reported but not failed.
      r2_errors=$(grep -cE '^\[(Error|Fatal) ' "$log" 2>/dev/null || true)
      if [ ${#failures[@]} -eq 0 ] && { [ -z "$r2_plugins" ] || [ "$r2_loading" -ne "$r2_plugins" ]; }; then
        failures+=("r2: BepInEx counted ${r2_plugins:-no} plugins but logged $r2_loading Loading lines")
      fi
      if grep -qE '\[(Error|Fatal) *: *BepInEx\]' "$log" 2>/dev/null; then
        failures+=("r2: BepInEx logged errors: $(grep -E '\[(Error|Fatal) *: *BepInEx\]' "$log" | head -5 | tr '\n' ' ')")
      fi
      echo "stopping the game"
      lc_stop r2
    fi
  fi

  if [ "$matrix" = run ]; then
    regress_bepinex_after "Regress LC"
    local failed
    failed=$(grep -c "$(printf '\tFAIL\t')" "$ROOT/matrix.tsv" || true)
    matrix="$(grep -c "$(printf '\tPASS\t')" "$ROOT/matrix.tsv" || true) PASS, $failed FAIL"
    [ "$failed" -eq 0 ] || failures+=("matrix: $failed rows failed; see $ROOT/matrix.tsv")
  fi

  [ ${#failures[@]} -eq 0 ] && verdict=PASS
  echo "---- regress lethal-company: $verdict ($((SECONDS - t0))s)"
  echo "profile        $profile"
  echo "BepInEx        $bepinex"
  echo "plugins loaded $loaded of ${#plugins[@]} (${plugins[*]})"
  echo "bridge         $bridge"
  echo "game entries   $(wc -l <"$ROOT/game-before.txt") hashed, $diff_lines differing after purge"
  echo "r2 code        ${code:-none}: $r2 into ${r2_profile:-none}, $r2_listed listed, $r2_mods mods installed, BepInEx counted ${r2_plugins:-?} plugins and logged $r2_loading Loading lines, $r2_errors error lines"
  echo "stopped pids   ${killed:-none}"
  echo "matrix         $matrix"
  local f
  for f in "${failures[@]}"; do echo "FAIL: $f"; done
  [ "$verdict" = PASS ]
}

# fake_game ID prints "APP_ID|FOLDER|MARKER|BEPINEX" for a catalog game that has no install on this machine, so a
# regress can run against a stand-in folder (see regress_fake).
fake_game() {
  case "$1" in
    repo) echo "3241660|REPO|REPO.exe|5.4.2305" ;;
    peak) echo "3527290|PEAK|PEAK.exe|5.4.75301" ;;
    riskofrain2) echo "632360|Risk of Rain 2|Risk of Rain 2.exe|5.4.2122" ;;
    *) return 1 ;;
  esac
}

# regress_fake ID is regress for a game with no install here: the sandbox holds a fake install (the marker file and a
# Steam manifest, nothing a game would load) and the launch is a dummy window started through the guard, so what it
# proves is Mortar's side: discovery, the BepInEx pack install, the profile, the Doorstop pair placed beside the
# executable for the launch, the hidden display, and a game folder that hashes as before once the launch is purged.
# That BepInEx loads inside the real game is not covered; the game stays disabled in the catalog until a real run.
regress_fake() {
  local id=${1:?usage: selftest.sh regress --fake GAME_ID} spec app folder marker bepinex
  spec=$(fake_game "$id") || {
    echo "no fake install defined for $id" >&2
    exit 2
  }
  IFS="|" read -r app folder marker bepinex <<<"$spec"
  command -v zenity >/dev/null || {
    echo "regress --fake needs zenity as its dummy window" >&2
    exit 1
  }
  ROOT=$(mktemp -d /var/tmp/mortar-regress-fake-XXXXXX)
  case "$ROOT" in /var/tmp/mortar-regress-fake-??????) ;; *)
    echo "unexpected sandbox dir $ROOT" >&2
    exit 1
    ;;
  esac
  mark
  PORT=$((9600 + RANDOM % 300))
  while [ -n "$(ss -ltn "sport = :$PORT" | tail -n +2)" ]; do PORT=$((9600 + RANDOM % 300)); done
  SANDBOX_HOME=$ROOT/home
  SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
  export MORTAR_ENABLE_GAMES=$id
  verdict=FAIL
  local game="$SANDBOX_STEAM/steamapps/common/$folder" compat="$SANDBOX_STEAM/steamapps/compatdata/$app"
  local data=$SANDBOX_HOME/.local/share/mortar failures=() profile="" t0=$SECONDS diff_lines=0 placed=no
  finish() { release_sandbox; }
  regress_traps
  need_launches 1

  mkdir -p "$SANDBOX_STEAM/config" "$SANDBOX_STEAM/steamapps/common" "$game" "$compat/pfx/drive_c/users/steamuser/AppData/LocalLow"
  printf 'MZ fake executable for a Mortar regress; it is never run\n' >"$game/$marker"
  mkdir -p "$game/${marker%.exe}_Data"
  printf '"AppState"\n{\n\t"appid"\t\t"%s"\n\t"installdir"\t\t"%s"\n\t"StateFlags"\t\t"4"\n}\n' "$app" "$folder" >"$SANDBOX_STEAM/steamapps/appmanifest_$app.acf"
  [ -f "$STEAM/config/loginusers.vdf" ] && cp "$STEAM/config/loginusers.vdf" "$SANDBOX_STEAM/config/"
  printf 'WINE REGISTRY Version 2\n;; All keys relative to \\\\User\\\\S-1-5-21-0-0-0-1000\n\n#arch=win64\n' >"$compat/pfx/user.reg"
  write_library

  # The launch prefix stands in for Proton: it records whether the Doorstop pair sits beside the executable, then
  # keeps a dummy window up until Mortar stops it.
  cat >"$ROOT/run-fake.sh" <<EOF
#!/bin/bash
dir=\$(dirname "\$1")
{ for f in winhttp.dll doorstop_config.ini; do [ -f "\$dir/\$f" ] && echo "\$f"; done; } >'$ROOT/seen.txt'
printf '%s\n' "\$@" >'$ROOT/args.txt'
exec -a "\$1" zenity --info --text 'fake $id game'
EOF
  chmod +x "$ROOT/run-fake.sh"

  build
  start
  cli settings set --game "$id" defaultLaunchMethod direct >/dev/null
  profile=$(cli profile create "$id" "Regress fake" | cut -f1)
  cli games --json >"$ROOT/games.json"
  python3 - "$ROOT/games.json" "$id" <<'PY' || failures+=("discovery: $id is not listed as installed with the fake folder")
import json, sys
g = next((g for g in json.load(open(sys.argv[1])) if g["id"] == sys.argv[2]), None)
sys.exit(0 if g and g.get("installed") else 1)
PY
  # A game whose catalog offers graphics APIs gets its recommended one chosen, so the launch must carry its arguments.
  local graphics_args choice=""
  graphics_args=$(python3 - "$REPO/internal/components/components.json" "$id" <<'PY'
import json, sys
g = next(g for g in json.load(open(sys.argv[1]))["games"] if g["id"] == sys.argv[2])
gr = g.get("graphics")
if gr:
    c = next(c for c in gr["choices"] if c["id"] == gr["recommended"])
    print(c["id"], *(c.get("args") or []))
PY
  )
  if [ -n "$graphics_args" ]; then
    choice=${graphics_args%% *}
    cli settings set --game "$id" graphicsApi "$choice" >/dev/null || failures+=("setting graphicsApi=$choice failed")
  fi
  for _ in $(seq 1 30); do
    cli loader install "$id" "$bepinex" >"$ROOT/loader.txt" 2>&1 && break
    grep -q 'already running' "$ROOT/loader.txt" || break
    sleep 2
  done
  grep -q "^Installed loader $bepinex" "$ROOT/loader.txt" || failures+=("BepInEx $bepinex did not install: $(head -c 300 "$ROOT/loader.txt")")
  cli loader pin "$id" "$bepinex" >/dev/null 2>&1 || failures+=("pinning BepInEx $bepinex failed")
  cli profile set "$id" "$profile" launchPrefix "$ROOT/run-fake.sh" >/dev/null
  tree_hash "$game" >"$ROOT/game-before.txt"

  if [ ${#failures[@]} -eq 0 ]; then
    cli launch "$id" "$profile" >"$ROOT/launch.txt" 2>&1 || failures+=("launch failed: $(head -c 300 "$ROOT/launch.txt")")
    for _ in $(seq 1 60); do
      [ -s "$ROOT/seen.txt" ] && break
      sleep 1
    done
    if [ "$(sort "$ROOT/seen.txt" 2>/dev/null | tr '\n' ' ')" = "doorstop_config.ini winhttp.dll " ]; then
      placed=yes
    else
      failures+=("the Doorstop pair was not beside the executable at launch (saw: $(tr '\n' ' ' <"$ROOT/seen.txt" 2>/dev/null))")
    fi
    grep -q -- '--doorstop' "$ROOT/args.txt" 2>/dev/null || failures+=("the launch carried no --doorstop arguments")
    local arg
    for arg in ${graphics_args#"$choice"}; do
      grep -qxF -- "$arg" "$ROOT/args.txt" 2>/dev/null || failures+=("the launch did not carry the graphics argument $arg")
    done
    cli stop "$id" >"$ROOT/stop.txt" 2>&1 || failures+=("mortar stop failed: $(head -c 300 "$ROOT/stop.txt")")
    local state=""
    for _ in $(seq 1 60); do
      state=$(cli status "$id" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])')
      [ "$state" = idle ] && break
      sleep 1
    done
    [ "$state" = idle ] || failures+=("Mortar never went idle after the launch was stopped")
    sleep 3
    tree_hash "$game" >"$ROOT/game-after.txt"
    diff_lines=$(diff "$ROOT/game-before.txt" "$ROOT/game-after.txt" | grep -c '^[<>]' || true)
    [ "$diff_lines" -eq 0 ] || failures+=("game folder differs after purge ($diff_lines lines)")
  fi

  [ ${#failures[@]} -eq 0 ] && verdict=PASS
  echo "---- regress --fake $id: $verdict ($((SECONDS - t0))s)"
  echo "profile        $profile"
  echo "BepInEx        $bepinex"
  echo "Doorstop pair  $placed beside $marker at launch"
  [ -z "$graphics_args" ] || echo "graphics API   $choice, launch args:${graphics_args#"$choice"}"
  echo "game entries   $(wc -l <"$ROOT/game-before.txt") hashed, $diff_lines differing after purge"
  local f
  for f in "${failures[@]}"; do echo "FAIL: $f"; done
  [ "$verdict" = PASS ]
}

# display_peers PID prints the path of every server PID's network namespace holds a connection to. A launch has a
# namespace of its own, and the server side of a unix connection lives in the client's namespace, so each connected
# socket there with a path is a display, bus or other service the game reached.
display_peers() {
  python3 - "$1" <<'PY'
import sys
for line in open(f"/proc/{sys.argv[1]}/net/unix").read().splitlines()[1:]:
    cols = line.split(None, 7)
    if len(cols) == 8 and cols[5] == "03":
        print(cols[7])
PY
}

# harness_check proves the launch harness without starting a game: in a throwaway sandbox it starts a dummy GTK
# window (zenity) through the guard three times, on Wayland and on X11, and requires each to run on the hidden display
# and to reach no socket of the desktop session's; the fourth launch must be refused, and stopping must take exactly
# the recorded pids. It runs under a session key of its own, so it spends none of the real session's launches.
harness_check() {
  ROOT=$(mktemp -d "$BASE/mortar-harness-XXXXXX")
  case "$ROOT" in "$BASE"/mortar-harness-??????) ;; *)
    echo "unexpected sandbox dir $ROOT" >&2
    exit 1
    ;;
  esac
  mark
  SESSION_KEY=harness-check-$$
  LAUNCH_COUNT=$BASE/mortar-launch-cap/$SESSION_KEY
  verdict=FAIL
  local desk_x=${DISPLAY:-} desk_run=${XDG_RUNTIME_DIR:-} failures=() i backend out rc=0
  finish() {
    rm -f "$LAUNCH_COUNT" "$LAUNCH_COUNT.lock"
    release_sandbox
  }
  regress_traps
  command -v zenity >/dev/null || {
    echo "harness-check needs zenity as its dummy window" >&2
    exit 1
  }
  display_up
  local hidden_x
  hidden_x=$(cat "$ROOT/run/display.x11")
  for i in 1 2 3; do
    backend=x11
    [ "$i" = 1 ] && backend=wayland
    (cd "$ROOT" && hidden env GDK_BACKEND="$backend" nohup "$REPO/scripts/launch-guard.sh" \
      zenity --info --text "launch harness check $i" >"$ROOT/dummy-$i.log" 2>&1 &)
  done
  for _ in $(seq 1 100); do
    [ -f "$ROOT/launches.pids" ] && [ "$(wc -l <"$ROOT/launches.pids")" -ge 3 ] && break
    sleep 0.1
  done
  # A window is mapped once the client has talked to its display; give each a moment to connect.
  sleep 3
  local pid start _ peers env_of
  while read -r pid start _; do
    env_of=$(tr '\0' '\n' <"/proc/$pid/environ" 2>/dev/null || true)
    [ -n "$env_of" ] || {
      failures+=("dummy $pid exited early; see $ROOT/dummy-*.log")
      continue
    }
    grep -qx "DISPLAY=$hidden_x" <<<"$env_of" || failures+=("dummy $pid has $(grep '^DISPLAY=' <<<"$env_of"), not $hidden_x")
    grep -qx "XDG_RUNTIME_DIR=$ROOT/run" <<<"$env_of" || failures+=("dummy $pid has the desktop's runtime dir")
    grep -qx "MORTAR_SKIP_INTRO=1" <<<"$env_of" || failures+=("dummy $pid lacks MORTAR_SKIP_INTRO=1")
    peers=$(display_peers "$pid")
    echo "dummy $pid ($(grep '^GDK_BACKEND=' <<<"$env_of")) talks to: $(tr '\n' ' ' <<<"$peers")"
    if ! grep -qE "^($ROOT/run/|@?/tmp/.X11-unix/X${hidden_x#:}$)" <<<"$peers"; then
      failures+=("dummy $pid is connected to no socket of the hidden display")
    fi
    if [ -n "$desk_x" ] && grep -qE "/tmp/.X11-unix/X${desk_x#:}$" <<<"$peers"; then
      failures+=("dummy $pid reached the desktop's X server $desk_x")
    fi
    # The desktop's Wayland socket, session bus and audio all live in its runtime dir.
    if [ -n "$desk_run" ] && grep -q "^$desk_run/" <<<"$peers"; then
      failures+=("dummy $pid reached the desktop session: $(grep "^$desk_run/" <<<"$peers" | tr '\n' ' ')")
    fi
  done <"$ROOT/launches.pids"
  [ "$(wc -l <"$ROOT/launches.pids")" = 3 ] || failures+=("expected 3 recorded launches, found $(wc -l <"$ROOT/launches.pids")")

  out=$(cd "$ROOT" && hidden "$REPO/scripts/launch-guard.sh" zenity --info --text "fourth" 2>&1) || rc=$?
  echo "fourth launch: exit $rc: $out"
  [ "$rc" = 3 ] && grep -q 'used all 3' <<<"$out" || failures+=("the fourth launch was not refused (exit $rc)")
  rc=0
  out=$(env MORTAR_LAUNCH_COUNT="$ROOT/desktop-count" MORTAR_LAUNCH_PIDS="$ROOT/desktop.pids" \
    MORTAR_HIDDEN_RUNTIME="$ROOT/run" "$REPO/scripts/launch-guard.sh" zenity --info 2>&1) || rc=$?
  echo "launch from the desktop session: exit $rc: $out"
  [ "$rc" = 3 ] && [ ! -e "$ROOT/desktop.pids" ] || failures+=("a launch with the desktop's display was not refused")

  local recorded=()
  mapfile -t recorded < <(cut -d' ' -f1 "$ROOT/launches.pids")
  stop_launches
  for pid in "${recorded[@]}"; do
    [ -e "/proc/$pid" ] && ! grep -q ') Z' "/proc/$pid/stat" 2>/dev/null && failures+=("recorded pid $pid survived stop_launches")
  done

  [ ${#failures[@]} -eq 0 ] && verdict=PASS
  echo "---- harness-check: $verdict"
  local f
  for f in "${failures[@]}"; do echo "FAIL: $f"; done
  [ "$verdict" = PASS ]
}

# curseforge_scenario proves CurseForge end to end in its own throwaway sandbox, with no game launched: a Stardew
# profile, a search for Content Patcher, the top hit installed through the queue, an update check on it, and a mod whose
# author disallows third-party distribution, which must not be downloaded. MORTAR_CURSEFORGE_KEY is the app key; it is
# built into this sandbox's server only, and sent to the API through a config file descriptor, never on a command line
# or in output. Unset, the scenario skips.
curseforge_scenario() {
  if [ -z "${MORTAR_CURSEFORGE_KEY:-}" ]; then
    echo "SKIP curseforge: MORTAR_CURSEFORGE_KEY is not set"
    return 0
  fi
  ROOT=$(mktemp -d /var/tmp/mortar-regress-cf-XXXXXX)
  case "$ROOT" in /var/tmp/mortar-regress-cf-??????) ;; *)
    echo "unexpected sandbox dir $ROOT" >&2
    exit 1
    ;;
  esac
  mark
  PORT=$((9600 + RANDOM % 300))
  while [ -n "$(ss -ltn "sport = :$PORT" | tail -n +2)" ]; do PORT=$((9600 + RANDOM % 300)); done
  SANDBOX_HOME=$ROOT/home
  SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
  verdict=FAIL
  local failures=() t0=$SECONDS profile="" top="" blocked="" blocked_name="" mods="" fileid=""
  local data=$SANDBOX_HOME/.local/share/mortar
  finish() { release_sandbox; }
  regress_traps

  setup
  build
  start

  profile=$(cli profile create stardew "CurseForge Test" | cut -f1)
  cli browse stardew "Content Patcher" --source curseforge --json >"$ROOT/cf-browse.json" 2>&1 ||
    failures+=("browse failed: $(head -c 300 "$ROOT/cf-browse.json")")
  top=$(python3 -c 'import json,sys; i=json.load(open(sys.argv[1]))["items"]; print(i[0]["id"] if i and i[0]["source"]=="curseforge" else "")' "$ROOT/cf-browse.json" 2>/dev/null || true)
  [ -n "$top" ] || failures+=("the search returned no CurseForge hit for Content Patcher")

  if [ -n "$top" ]; then
    cli queue add stardew "$profile" "$top" --source curseforge >"$ROOT/cf-add.txt" 2>&1 ||
      failures+=("queue add failed: $(head -c 300 "$ROOT/cf-add.txt")")
    cf_wait_queue "$profile" >"$ROOT/cf-wait.txt"
    [ -s "$ROOT/cf-wait.txt" ] && failures+=("download did not finish: $(head -c 300 "$ROOT/cf-wait.txt")")
    cli mods stardew "$profile" --json >"$ROOT/cf-mods.json"
    mods=$(python3 -c 'import json,sys; print(sum(1 for m in json.load(open(sys.argv[1])) if m["source"]=="curseforge" and m["version"]))' "$ROOT/cf-mods.json")
    [ "$mods" -ge 1 ] || failures+=("no mod with source curseforge and a version landed in the profile")
    fileid=$(python3 - "$data/profiles/stardew" "$profile" <<'PY'
import json, os, sys
root = os.path.join(sys.argv[1], sys.argv[2])
for d, _, files in os.walk(root):
    for f in files:
        if not f.endswith(".json"):
            continue
        try:
            doc = json.load(open(os.path.join(d, f)))
        except (OSError, ValueError):
            continue
        for e in doc.get("entries", []) if isinstance(doc, dict) else []:
            src = e.get("source", {})
            if src.get("kind") == "curseforge" and src.get("fileId"):
                print(src["fileId"])
                sys.exit()
PY
)
    [ -n "$fileid" ] || failures+=("the profile records no CurseForge file id")
    cli updates stardew "$profile" --json >"$ROOT/cf-updates.json" 2>&1 ||
      failures+=("update check failed: $(head -c 300 "$ROOT/cf-updates.json")")
  fi

  # Mods whose authors disallow third-party distribution have allowModDistribution false in the API.
  local page
  for page in 0 50 100 150 200; do
    curl -fsS --config <(printf 'header = "x-api-key: %s"\n' "$MORTAR_CURSEFORGE_KEY") \
      "https://api.curseforge.com/v1/mods/search?gameId=669&sortField=2&sortOrder=desc&pageSize=50&index=$page" \
      >"$ROOT/cf-search.json" 2>/dev/null || break
    blocked=$(python3 - "$ROOT/cf-search.json" <<'PY'
import json, sys
for m in json.load(open(sys.argv[1]))["data"]:
    if m.get("allowModDistribution") is False:
        print(f'{m["id"]}\t{m["name"]}')
        break
PY
)
    [ -n "$blocked" ] && break
  done
  if [ -z "$blocked" ]; then
    echo "note: no mod with allowModDistribution=false in the first 250 Stardew hits; handoff check skipped"
  else
    blocked_name=${blocked#*$'\t'}
    # Refused with a pointer to the CurseForge page, which the sandbox's xdg-open stub logs instead of opening.
    if cli queue add stardew "$profile" "${blocked%%$'\t'*}" --source curseforge >"$ROOT/cf-blocked.txt" 2>&1; then
      failures+=("queue add accepted a non-distributable mod")
    elif ! grep -q "CurseForge" "$ROOT/cf-blocked.txt"; then
      failures+=("refusal does not point to CurseForge: $(head -c 300 "$ROOT/cf-blocked.txt")")
    fi
    grep -qF "(the page is open)" "$ROOT/cf-blocked.txt" ||
      failures+=("the mod's CurseForge page was not handed off: $(head -c 300 "$ROOT/cf-blocked.txt")")
    cli queue --json >"$ROOT/cf-queue.json"
    cli mods stardew "$profile" --json >"$ROOT/cf-mods2.json"
    python3 - "$ROOT/cf-queue.json" "$ROOT/cf-mods2.json" "$blocked_name" "$mods" >"$ROOT/cf-handoff.txt" <<'PY' || failures+=("non-distributable mod ${blocked_name}: $(head -c 300 "$ROOT/cf-handoff.txt")")
import json, sys
queue, mods, name, before = sys.argv[1:]
bad = [i for i in json.load(open(queue))["items"] if i["name"] == name and i["state"] in ("downloading", "installing", "done")]
landed = sum(1 for m in json.load(open(mods)) if m["source"] == "curseforge" and m["version"])
if bad:
    print("downloaded: " + ", ".join(i["state"] for i in bad)); sys.exit(1)
if landed != int(before or 0):
    print(f"curseforge mods in the profile went from {before} to {landed}"); sys.exit(1)
PY
  fi

  # The key must appear in no output the scenario or the server wrote.
  if grep -rqF -- "$MORTAR_CURSEFORGE_KEY" "$ROOT" --include='*.json' --include='*.txt' --include='*.log' 2>/dev/null; then
    failures+=("the CurseForge key appears in a log or output file under the sandbox")
  fi

  [ ${#failures[@]} -eq 0 ] && verdict=PASS
  echo "---- curseforge: $verdict ($((SECONDS - t0))s)"
  echo "top hit        ${top:-none}"
  echo "installed      ${mods:-0} curseforge mods, file id ${fileid:-none}"
  echo "non-distrib.   ${blocked_name:-none found}"
  local f
  for f in "${failures[@]}"; do echo "FAIL: $f"; done
  [ "$verdict" = PASS ]
}

# cf_wait_queue PROFILE waits for the profile's downloads and prints the ones that did not finish.
cf_wait_queue() {
  local deadline=$((SECONDS + 180)) out=""
  while [ "$SECONDS" -lt "$deadline" ]; do
    cli queue --json >"$ROOT/cf-queue.json"
    out=$(python3 - "$ROOT/cf-queue.json" "$1" <<'PY'
import json, sys
items = [i for i in json.load(open(sys.argv[1]))["items"] if i["profileId"] == sys.argv[2]]
for i in items:
    if i["state"] != "done":
        print(f'{i["name"]}: {i["state"]} {i["error"]}')
PY
)
    [ -z "$out" ] && return 0
    case "$out" in *failed* | *skipped* | *cancelled*) break ;; esac
    sleep 2
  done
  echo "$out"
}

# deletable DIR succeeds only for a folder that carries the marker and whose parent is exactly the base dir, both
# resolved, so neither a typo nor a symlink can point a delete anywhere else.
deletable() {
  local dir base
  dir=$(realpath -e -- "$1" 2>/dev/null) || return 1
  base=$(realpath -e -- "$BASE" 2>/dev/null) || return 1
  [ "$(dirname -- "$dir")" = "$base" ] && [ -f "$dir/$MARKER" ]
}

destroy() {
  deletable "$ROOT" || {
    echo "refusing to delete $ROOT: not a marked sandbox directly under $BASE" >&2
    exit 1
  }
  ROOT=$(realpath -e -- "$ROOT")
  exec 3>&2
  verdict=PASS
  release_sandbox
  [ -e "$ROOT" ] && exit 1
  echo "deleted $ROOT"
}

reap() {
  local hours=6 yes="" dir
  while [ $# -gt 0 ]; do
    case "$1" in
      --hours)
        hours=${2:?--hours takes a number}
        shift 2
        ;;
      --yes)
        yes=1
        shift
        ;;
      *)
        echo "reap takes --hours N and --yes" >&2
        exit 2
        ;;
    esac
  done
  [[ $hours =~ ^[0-9]+$ ]] || {
    echo "--hours takes a whole number" >&2
    exit 2
  }
  local found=0
  for dir in "$BASE"/*/; do
    dir=${dir%/}
    [ -f "$dir/$MARKER" ] || continue
    if ! deletable "$dir"; then
      continue
    fi
    ROOT=$dir
    if [ -n "$(sandbox_pids)" ]; then
      echo "live     $dir (processes running from it)"
      continue
    fi
    if [ -n "$(find "$dir" -mmin "-$((hours * 60))" -print -quit 2>/dev/null)" ]; then
      echo "recent   $dir (changed within ${hours}h)"
      continue
    fi
    found=1
    echo "idle     $(du -sh -- "$dir" | cut -f1)	$dir"
    if [ -n "$yes" ]; then
      # Checked again right before the delete: the folder may have been reused since the scan above.
      if deletable "$dir" && [ -z "$(sandbox_pids)" ]; then
        rm -rf -- "$dir"
        echo "deleted  $dir"
      fi
    fi
  done
  if [ "$found" = 1 ] && [ -z "$yes" ]; then
    echo "dry run; rerun with --yes to delete the idle sandboxes"
  fi
}

case "${1:-}" in
  destroy | reap | harness-check) ;;
  regress)
    case "${2:-}${3:-}" in
      "") regress ;;
      --gamelethal-company) regress_lc ;;
      --gamestardew) regress ;;
      --fake*) regress_fake "${3:-}" ;;
      *)
        echo "regress takes --game stardew, --game lethal-company or --fake GAME_ID" >&2
        exit 2
        ;;
    esac
    ;;
  curseforge) curseforge_scenario ;;
  *) mkdir -p "$ROOT" ;;
esac
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
    start
    ;;
  setup) setup ;;
  ea-fixture) ea_fixture "${2:-}" "${3:-}" ;;
  seed) seed ;;
  stop) stop ;;
  destroy) destroy ;;
  reap)
    shift
    reap "$@"
    ;;
  regress | curseforge) ;;
  harness-check) harness_check ;;
  *)
    sed -n '2,24p' "$0"
    exit 2
    ;;
esac
