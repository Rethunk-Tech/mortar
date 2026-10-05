#!/usr/bin/env bash
# Runs Mortar in server mode against a sandboxed home, for self-testing in a browser at http://127.0.0.1:$PORT.
# The sandbox has its own HOME with a minimal Steam library holding copies of the games (Stardew Valley and, when installed, Lethal Company), so nothing Mortar does
# reaches the real game folder, the real data folder or the real Steam config.
#
#   scripts/selftest.sh start [--copy-data]   build, set up the sandbox if missing, start the server
#   scripts/selftest.sh setup                 only create or top up the sandbox's Steam library; no build, no server
#   scripts/selftest.sh restart               rebuild from the working tree and restart
#   scripts/selftest.sh stop                  stop the server
#   scripts/selftest.sh seed                  fill the running sandbox with fixture data (once; skipped when present)
#   scripts/selftest.sh regress [--game lethal-company]  one-shot regression run (Stardew, or Lethal Company through Proton) in its own throwaway sandbox (see below)
#
# --copy-data copies the real Mortar profiles and settings into the sandbox once (downloads, cache, trash and
# backups are left out). The sandbox is never deleted by this script; remove $ROOT by hand to start over.
#
# regress is the exception: it makes /var/tmp/mortar-regress-XXXXXX on a free port, copies the real data, launches the
# first Stardew profile directly, waits for SMAPI to load every enabled mod, then stops the game and requires the game
# folder to hash exactly as before. It deletes only its own directory, after a pass; a failure keeps it for the logs.
# MORTAR_REGRESS_PROFILE picks another profile and MORTAR_REGRESS_TIMEOUT (seconds, default 180) bounds the load wait.
# regress --game lethal-company builds its own sandbox, bootstraps a Proton prefix, installs BepInEx and three plugins
# from Thunderstore (cached in /var/tmp/mortar-regress-cache; MORTAR_REGRESS_OFFLINE=1 proves a rerun needs no network),
# launches directly under Proton, and requires the plugins to load and the purge to leave the game folder identical.
set -euo pipefail

ROOT=${MORTAR_SELFTEST_DIR:-/var/tmp/mortar-selftest}
PORT=${MORTAR_SELFTEST_PORT:-9455}
STEAM=${MORTAR_SELFTEST_STEAM:-$HOME/.local/share/Steam}
APP_ID=413150
GAME_FOLDER="Stardew Valley"
LC_APP_ID=1966720
LC_FOLDER="Lethal Company"
REPO=$(cd "$(dirname "$0")/.." && pwd)
SANDBOX_HOME=$ROOT/home
SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
# Games the shipped catalog has not enabled yet are switched on in the sandbox only (comma separated catalog ids).
export MORTAR_ENABLE_GAMES=${MORTAR_SELFTEST_ENABLE:-lethal-company}

build() {
  echo "building frontend and server-mode binary"
  # The server embeds frontend/dist, which every sandbox's build rewrites, so concurrent sandboxes build one at a time.
  exec 9>/var/tmp/mortar-selftest-build.lock
  flock 9
  (cd "$REPO" && bun run --cwd frontend build >"$ROOT/frontend-build.log" 2>&1)
  (cd "$REPO" && GOTMPDIR=/var/tmp go build -tags server -o "$ROOT/mortar-server.new" .)
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
  local apps="" app
  for app in "$APP_ID" "$LC_APP_ID"; do
    if [ -f "$SANDBOX_STEAM/steamapps/appmanifest_$app.acf" ]; then
      apps+=$'\t\t\t"'$app$'"\t\t"1"\n'
    fi
  done
  printf '"libraryfolders"\n{\n\t"0"\n\t{\n\t\t"path"\t\t"%s"\n\t\t"apps"\n\t\t{\n%s\t\t}\n\t}\n}\n' "$SANDBOX_STEAM" "$apps" >"$SANDBOX_STEAM/steamapps/libraryfolders.vdf"
}

setup() {
  mkdir -p "$SANDBOX_STEAM/config"
  copy_game "$APP_ID" "$GAME_FOLDER" || exit 1
  [ -f "$SANDBOX_STEAM/config/loginusers.vdf" ] || cp "$STEAM/config/loginusers.vdf" "$SANDBOX_STEAM/config/"
  # Lethal Company is optional: a machine without it still gets the Stardew sandbox.
  copy_game "$LC_APP_ID" "$LC_FOLDER" || true
  # An empty prefix is enough for runtime path resolution; the real one is never copied.
  if [ -d "$SANDBOX_STEAM/steamapps/common/$LC_FOLDER" ]; then
    mkdir -p "$SANDBOX_STEAM/steamapps/compatdata/$LC_APP_ID/pfx/drive_c/users/steamuser/AppData/LocalLow"
  fi
  write_library
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
        os.wait()
    except ChildProcessError:
        break'
  if unshare --user --map-current-user --pid --fork --mount --mount-proc true 2>/dev/null; then
    isolate=(unshare --user --map-current-user --pid --fork --mount --mount-proc python3 -c "$reaper")
  fi
  (cd "$ROOT" && env -u XDG_DATA_HOME -u XDG_CONFIG_HOME -u XDG_CACHE_HOME HOME="$SANDBOX_HOME" PATH="$ROOT/bin:$PATH" \
    WAILS_SERVER_HOST=127.0.0.1 WAILS_SERVER_PORT="$PORT" nohup "${isolate[@]}" ./mortar-server >"$ROOT/server.log" 2>&1 &)
  for _ in $(seq 1 30); do
    if [ -n "$(listener || true)" ]; then
      # Recorded so a test run that dies can have its server stopped by pid later (frontend/e2e/sandbox.ts).
      listener >"$ROOT/server.pid"
      echo "self-test server on http://127.0.0.1:$PORT (pid $(listener), log $ROOT/server.log)"
      return
    fi
    sleep 1
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
    printf '[General]\nEnabled = true\n' >"$fx/$name/config/Self-test.$name.cfg"
    (cd "$fx/$name" && python3 -m zipfile -c "$fx/zips/$name.zip" ./*)
  done
  cli profile create lethal-company "Seed Lobby"
  for name in SeedAlpha SeedBeta; do
    cli install lethal-company "Seed Lobby" "$fx/zips/$name.zip"
  done
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

  # The scheduled backup runs on the server's first check after start, so enable it and restart without rebuilding.
  cli settings set --game stardew saveBackupHours 1
  stop
  start
  # stop waits for the old server to exit; the new one's first check runs at start and logs its pass.
  for _ in $(seq 1 30); do
    if cli backups list --game stardew --json | grep -q scheduled; then
      seed_lc
      echo "sandbox seeded"
      return
    fi
    sleep 1
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

regress() {
  ROOT=$(mktemp -d /var/tmp/mortar-regress-XXXXXX)
  case "$ROOT" in /var/tmp/mortar-regress-??????) ;; *) echo "unexpected sandbox dir $ROOT" >&2; exit 1 ;; esac
  PORT=$((9600 + RANDOM % 300))
  while [ -n "$(ss -ltn "sport = :$PORT" | tail -n +2)" ]; do PORT=$((9600 + RANDOM % 300)); done
  SANDBOX_HOME=$ROOT/home
  SANDBOX_STEAM=$SANDBOX_HOME/.local/share/Steam
  # game, gpids and verdict are read by the EXIT trap, after this function's locals are gone.
  game="$SANDBOX_STEAM/steamapps/common/$GAME_FOLDER"
  gpids=""
  verdict=FAIL
  local timeout=${MORTAR_REGRESS_TIMEOUT:-180}
  local log="$SANDBOX_HOME/.config/StardewValley/ErrorLogs/SMAPI-latest.txt" t0=$SECONDS
  local failures=() mods_n="" packs_n="" skips=0 enabled="" profile="" diff_lines=0

  finish() {
    local pid
    for pid in $gpids; do
      if [ "$(readlink "/proc/$pid/exe" 2>/dev/null)" ] && case "$(readlink "/proc/$pid/exe")" in "$game"/*) true ;; *) false ;; esac; then
        kill "$pid" 2>/dev/null || true
      fi
    done
    stop >/dev/null 2>&1 || true
    if [ "$verdict" = PASS ]; then
      rm -rf "$ROOT"
    else
      echo "sandbox kept for inspection: $ROOT" >&2
    fi
  }
  trap finish EXIT

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
    if { tr '\0' '\n' <"/proc/$p/environ" | grep -qxE "(WINEPREFIX=${compat}/pfx/?|STEAM_COMPAT_DATA_PATH=${compat}/?)"; } 2>/dev/null \
      && ! grep -q ') Z' "/proc/$p/stat" 2>/dev/null; then
      kill -KILL "$p" 2>/dev/null || true
    fi
  done
  echo "${first[*]} ${pids[*]}"
}

# regress_lc is regress for Lethal Company: a direct Proton launch of a BepInEx profile with three Thunderstore plugins,
# in its own sandbox. Steam is never started; Proton only needs Steam's client library and the .steam links, both
# copied into the sandbox home. The Thunderstore index and downloads are cached in $REGRESS_CACHE after the first run.
regress_lc() {
  ROOT=$(mktemp -d /var/tmp/mortar-regress-lc-XXXXXX)
  case "$ROOT" in /var/tmp/mortar-regress-lc-??????) ;; *) echo "unexpected sandbox dir $ROOT" >&2; exit 1 ;; esac
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
    stop >/dev/null 2>&1 || true
    if [ "$verdict" = PASS ]; then
      rm -rf "$ROOT"
    else
      echo "sandbox kept for inspection: $ROOT" >&2
    fi
  }
  trap finish EXIT

  [ -d "$proton" ] || { echo "no Proton at $proton (set MORTAR_REGRESS_PROTON)" >&2; exit 1; }
  [ -f "$STEAM/linux64/steamclient.so" ] || { echo "no steamclient.so under $STEAM" >&2; exit 1; }
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

  mkdir -p "$ROOT"
  cat >"$ROOT/run-proton.sh" <<EOF
#!/bin/bash
export STEAM_COMPAT_DATA_PATH='$compat'
export STEAM_COMPAT_CLIENT_INSTALL_PATH='$SANDBOX_STEAM'
export STEAM_COMPAT_APP_ID=$LC_APP_ID SteamAppId=$LC_APP_ID SteamGameId=$LC_APP_ID
exec '$proton/proton' run "\$@"
EOF
  chmod +x "$ROOT/run-proton.sh"
  # The first Proton run creates the prefix; Mortar can only add its winhttp override to a prefix that exists.
  echo "creating the Proton prefix"
  mkdir -p "$compat"
  (cd "$ROOT" && env HOME="$SANDBOX_HOME" timeout 300 "$ROOT/run-proton.sh" wineboot -u >"$ROOT/wineboot.log" 2>&1) \
    || failures+=("the Proton prefix could not be created; see $ROOT/wineboot.log")
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

  log="$data/profiles/lethal-company/$profile/BepInEx/LogOutput.log"
  echo "launching profile $profile (${#plugins[@]} plugins)"
  if [ ${#failures[@]} -eq 0 ]; then
    cli launch lethal-company "$profile" >"$ROOT/launch.txt" 2>&1 || failures+=("launch failed: $(head -c 300 "$ROOT/launch.txt")")
  fi
  deadline=$((SECONDS + timeout))
  while [ ${#failures[@]} -eq 0 ] && [ "$SECONDS" -lt "$deadline" ]; do
    grep -q 'Chainloader startup complete' "$log" 2>/dev/null && break
    sleep 2
  done
  if [ ${#failures[@]} -eq 0 ] && ! grep -q 'Chainloader startup complete' "$log" 2>/dev/null; then
    failures+=("BepInEx never logged \"Chainloader startup complete\" within ${timeout}s ($(cli status lethal-company 2>&1 | head -c 200))")
  fi
  local name
  for name in "${plugins[@]}"; do
    if [ ${#failures[@]} -gt 0 ]; then break; fi
    if grep -qE "BepInEx\] Loading \[$name " "$log" 2>/dev/null; then loaded=$((loaded + 1)); else failures+=("plugin $name did not load"); fi
  done

  echo "stopping the game"
  killed=$(reap_prefix "$compat")
  for _ in $(seq 1 60); do
    [ "$(cli status lethal-company --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])')" = idle ] && break
    sleep 1
  done
  [ "$(cli status lethal-company --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])')" = idle ] || failures+=("Mortar never went idle after the game stopped")
  sleep 3
  tree_hash "$game" >"$ROOT/game-after.txt"
  diff_lines=$(diff "$ROOT/game-before.txt" "$ROOT/game-after.txt" | grep -c '^[<>]' || true)
  if [ "$diff_lines" -ne 0 ]; then
    failures+=("game folder differs after purge ($diff_lines lines); see $ROOT/game-before.txt vs game-after.txt")
  fi
  cp "$log" "$ROOT/LogOutput.log" 2>/dev/null || true

  [ ${#failures[@]} -eq 0 ] && verdict=PASS
  echo "---- regress lethal-company: $verdict ($((SECONDS - t0))s)"
  echo "profile        $profile"
  echo "BepInEx        $bepinex"
  echo "plugins loaded $loaded of ${#plugins[@]} (${plugins[*]})"
  echo "game entries   $(wc -l <"$ROOT/game-before.txt") hashed, $diff_lines differing after purge"
  echo "stopped pids   ${killed:-none}"
  local f
  for f in "${failures[@]}"; do echo "FAIL: $f"; done
  [ "$verdict" = PASS ]
}

case "${1:-}" in
  regress)
    case "${2:-}${3:-}" in
      "") regress ;;
      --gamelethal-company) regress_lc ;;
      --gamestardew) regress ;;
      *) echo "regress takes --game stardew or --game lethal-company" >&2; exit 2 ;;
    esac
    ;;
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
    sleep 1
    start
    ;;
  setup) setup ;;
  seed) seed ;;
  stop) stop ;;
  regress) ;;
  *)
    sed -n '2,14p' "$0"
    exit 2
    ;;
esac
