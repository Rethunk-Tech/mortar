# shellcheck shell=bash
# The BepInEx test matrix (docs/bepinex-test-matrix.md), run by `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress
# --game lethal-company`. selftest.sh sources this file; regress_lc calls three stages with its sandbox set up (ROOT,
# PORT, data, game, compat, profile, timeout) and its helpers defined (cli, reap_prefix, tree_hash):
#   regress_bepinex_prepare  before the base launch: every row that needs no game process, and the base profile
#                            filled with the modpack and the probe packages, so the base launch is launch (a)
#   regress_bepinex_running  during launch (a): what only a running game shows
#   regress_bepinex_after    after launch (a) is stopped: launches (c) and (b) and the rows that need no launch
# A session allows 3 game launches (scripts/launch-guard.sh), so the matrix adds launches (c) and (b) to the base run's.
# Each matrix item ends as one PASS or FAIL row in $ROOT/matrix.tsv; any FAIL fails the regress. The modpack
# and the update packages come from Thunderstore, so the matrix needs the network.

# The modpack: the 60 most-downloaded Lethal Company packages that are neither deprecated nor modpacks nor tools
# (2026-10-06), pinned so a rerun installs the same files. Their dependencies come from the queue.
MATRIX_MODPACK=(
  notnotnotswipez-MoreCompany/1.14.0 Evaisa-LethalLib/1.2.0 FlipMods-ReservedItemSlotCore/2.0.55
  Rune580-LethalCompany_InputUtils/0.7.13 FlipMods-TooManyEmotes/2.3.17 x753-More_Suits/1.5.4
  FlipMods-ReservedFlashlightSlot/2.0.10 x753-Mimics/2.7.4 IAmBatby-LethalLevelLoader/1.7.13
  FlipMods-ReservedWalkieSlot/2.0.7 malco-Lategame_Upgrades/3.14.1 Evaisa-HookGenPatcher/0.0.5
  BunyaPineTree-ModelReplacementAPI/2.4.20 loaforc-loaforcsSoundAPI/2.0.13 sunnobunno-YippeeMod/1.2.4
  Evaisa-LethalThings/0.10.13 AinaVT-LethalConfig/1.4.6 Gemumoddo-LethalEmotesAPI/1.18.0 Verity-TooManySuits/2.0.4
  Hamunii-DetourContext_Dispose_Fix/1.0.9 Hamunii-AutoHookGenPatcher/1.1.1 EliteMasterEric-Coroner/2.4.2
  Evaisa-FixPluginTypesSerialization/1.1.4 FlipMods-BetterStamina/1.5.7 NotAtomicBomb-TerminalApi/1.5.6
  xilophor-LethalNetworkAPI/3.4.2 AudioKnight-StarlancerAIFix/3.13.2 qwbarch-Mirage/1.29.1 MonoDetour-MonoDetour/0.7.16
  MonoDetour-MonoDetour_BepInEx_5/0.7.16 fumiko-CullFactory/2.0.11 Jordo-NeedyCats/1.2.5
  AllToasters-SpectateEnemies/2.9.1 FlipMods-HotbarPlus/1.8.6 Lordfirespeed-OdinSerializer/2024.2.2700
  TwinDimensionalProductions-CoilHeadStare/1.1.0 Piggy-LC_Office/2.3.4 sfDesat-Orion/2.1.7
  MegaPiggy-BuyableShotgunShells/1.3.0 Hardy-LCMaxSoundsFix/1.2.0 Renegades-FlashlightToggle/1.5.0 Sigurd-CSync/5.0.1
  MaxWasUnavailable-LethalModDataLib/1.2.2 loaforc-FacilityMeltdown/2.7.5 MegaPiggy-BuyableShotgun/1.3.0
  JacobG5-JLL/1.10.1 KoderTeh-Boombox_Controller/1.2.7 ButteryStancakes-EnemySoundFixes/1.9.21 sfDesat-Aquatis/2.2.8
  FlipMods-FasterItemDropship/1.3.1 PopleZoo-BetterItemScan/3.0.2 mrgrm7-LethalCasino/1.1.3
  LethalResonance-LETHALRESONANCE/4.7.8 KlutzyBubbles-BetterEmotes/1.5.8 Gemumoddo-BadAssCompany/1.3.1
  WhiteSpike-Interactive_Terminal_API/1.3.3 mrov-MrovLib/0.4.15 Hexnet111-SuitSaver/1.2.1 Renegades-WalkieUse/1.5.0
  scoopy-Scoopys_Variety_Mod/1.2.0
)

# A FAIL row fails the regress at its end (regress_lc reads matrix.tsv), not at once: a matrix row failing before
# launch (a) must not keep the base run from launching.
mx_result() {
  printf '%s\t%s\t%s\n' "$1" "$2" "$3" >>"$ROOT/matrix.tsv"
  echo "matrix $2 $1: $3"
}
mx_pass() { mx_result "$1" PASS "$2"; }
mx_fail() { mx_result "$1" FAIL "$2"; }
# mx_check ID EVIDENCE COMMAND... records PASS with EVIDENCE when COMMAND succeeds, else FAIL with it.
mx_check() {
  local id=$1 evidence=$2
  shift 2
  if "$@"; then mx_pass "$id" "$evidence"; else mx_fail "$id" "$evidence"; fi
}

# mx_json FILE EXPR prints a Python expression over the JSON in FILE, bound to d.
mx_json() { python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(eval(sys.argv[2]))' "$1" "$2"; }

# mx_wails METHOD ARG... calls a bound Go method the window uses (package path below internal/, then Service.Method),
# through the server's Wails runtime, as the frontend does. Arguments are JSON values.
mx_wails() {
  local method=$1
  shift
  local args
  args=$(
    IFS=,
    echo "[$*]"
  )
  curl -fsS -X POST "http://127.0.0.1:$PORT/wails/runtime" -H 'Content-Type: application/json' -H 'x-wails-client-id: matrix' \
    --data-binary "{\"object\":0,\"method\":0,\"args\":{\"call-id\":\"m$RANDOM\",\"methodName\":\"github.com/Rethunk-Tech/mortar/internal/$method\",\"args\":$args}}"
}
mx_q() { python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"; }

mx_profile() { cli profile create lethal-company "$1" | cut -f1; }
mx_dir() { echo "$mx_data/profiles/lethal-company/$1"; }

# mx_wait_queue PROFILE waits for the profile's downloads; it prints the ones that did not finish.
mx_wait_queue() {
  local deadline=$((SECONDS + 600)) pending=1
  while [ "$SECONDS" -lt "$deadline" ]; do
    cli queue --json >"$ROOT/queue.json"
    pending=$(
      python3 - "$ROOT/queue.json" "$1" 2>"$ROOT/queue-open.txt" <<'PY'
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
  [ "$pending" = 0 ] || echo "downloads still open after 600s"
  cat "$ROOT/queue-open.txt"
}

# mx_game_pids prints the pids of the game's own executable in the sandbox prefix, each checked through its environment.
mx_game_pids() {
  local d
  for d in /proc/[0-9]*; do
    { tr '\0' '\n' <"$d/environ" | grep -qxE "WINEPREFIX=${mx_compat}/pfx/?"; } 2>/dev/null || continue
    tr '\0' ' ' <"$d/cmdline" 2>/dev/null | grep -q 'Lethal Company.exe' || continue
    tr '\0' ' ' <"$d/cmdline" | grep -q 'steam.exe' && continue
    echo "${d#/proc/}"
  done
}

mx_state() { cli status lethal-company --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["state"])'; }

# mx_launches prints how many games the matrix starts on top of the base run's launch (a): launches (c) and (b), the
# two mx_launch calls below. The session's launch cap must have that many left before the run begins (need_launches).
mx_launches() { echo 2; }

# mx_bep_version VERSION prints the version BepInEx logs for a Thunderstore BepInExPack version (5.4.2305 -> 5.4.23.5).
mx_bep_version() {
  local v=$1 last=${1##*.}
  echo "${v%.*}.$((10#$last / 100)).$((10#$last % 100))"
}

# mx_launch PROFILE TAG starts the profile and waits for BepInEx to finish loading. The previous run's log is moved
# aside first, so its last line cannot stand in for this run's.
mx_launch() {
  mx_log="$(mx_dir "$1")/BepInEx/LogOutput.log"
  [ -f "$mx_log" ] && mv "$mx_log" "$mx_log.prev"
  if ! cli launch lethal-company "$1" >"$ROOT/launch-$2.txt" 2>&1; then
    mx_fail "launch.$2" "launch failed: $(head -c 300 "$ROOT/launch-$2.txt")"
    return 1
  fi
  local deadline=$((SECONDS + mx_timeout))
  while [ "$SECONDS" -lt "$deadline" ]; do
    grep -q 'Chainloader startup complete' "$mx_log" 2>/dev/null && return 0
    [ "$(mx_state)" = idle ] && break
    sleep 2
  done
  mx_fail "launch.$2" "BepInEx never logged \"Chainloader startup complete\" ($(mx_state))"
  return 1
}

# mx_idle waits up to $1 seconds for Mortar to report the game idle.
mx_idle() {
  for _ in $(seq 1 "$1"); do
    [ "$(mx_state)" = idle ] && return 0
    sleep 1
  done
  return 1
}

# mx_console_reads reads the Console twice into mx_lines1 and mx_lines2: the first once the first heartbeat is in, the
# second once the probe's error line (its third beat) is and a heartbeat has come since the first read. With a modpack
# this size the game's first frame comes well after BepInEx's "Chainloader startup complete", and a sound mod then
# holds the main thread for several seconds. mx_read1_at and mx_read2_at are the UTC wall-clock times of the two
# reads, to set against Player.log.
mx_console_reads() {
  local beats1
  for _ in $(seq 1 30); do
    mx_lines1=$(mx_wails launchsvc.Service.Lines '"lethal-company"' "$(mx_q "$mx_base")")
    mx_read1_at=$(date -u +%T.%3N)
    case $mx_lines1 in *'matrix heartbeat'*) break ;; esac
    sleep 2
  done
  beats1=$(grep -o 'matrix heartbeat' <<<"$mx_lines1" | wc -l)
  for _ in $(seq 1 30); do
    sleep 2
    mx_lines2=$(mx_wails launchsvc.Service.Lines '"lethal-company"' "$(mx_q "$mx_base")")
    mx_read2_at=$(date -u +%T.%3N)
    [[ $mx_lines2 == *'matrix error line'* ]] && [ "$(grep -o 'matrix heartbeat' <<<"$mx_lines2" | wc -l)" -gt "$beats1" ] && break
  done
}

# mx_keep_logs TAG copies launch TAG's LogOutput.log and Unity's Player.log (and Player-prev.log) into the sandbox as
# LogOutput-TAG.log, Player-TAG.log and Player-prev-TAG.log: the next launch overwrites Unity's.
mx_keep_logs() {
  cp "$mx_log" "$ROOT/LogOutput-$1.log" 2>/dev/null
  cp "$mx_saves/Player.log" "$ROOT/Player-$1.log" 2>/dev/null
  [ ! -e "$mx_saves/Player-prev.log" ] || cp "$mx_saves/Player-prev.log" "$ROOT/Player-prev-$1.log"
  return 0
}

# mx_bridge_survived TAG LOG notes, as an INFO row, whether the bridge's plugin component was alive at each scene load,
# from the "Bridge plugin alive after scene X: True|False" lines its intro runner writes into LogOutput.log at LOG.
# BepInEx's default HideManagerGameObject=false lets Lethal Company's first scene load destroy that component, so
# whether the bridge still answers is row bridge.reachable's to judge.
mx_bridge_survived() {
  local alive dead
  alive=$(grep -c 'Bridge plugin alive after scene .*: True' "$2" 2>/dev/null)
  dead=$(grep -c 'Bridge plugin alive after scene .*: False' "$2" 2>/dev/null)
  mx_result "bridge.survived.$1" INFO "launch ($1): the bridge's plugin was alive after ${alive:-0} scene load(s) and destroyed after ${dead:-0}; first load: $(grep -m1 -o 'Bridge plugin alive after scene .*' "$2" || echo 'no line')"
}

# mx_wait_scene SCENE SECONDS waits for the bridge's intro runner to log SCENE loading in $mx_log.
mx_wait_scene() {
  local deadline=$((SECONDS + $2))
  until grep -q "Bridge plugin alive after scene $1:" "$mx_log" 2>/dev/null; do
    [ "$SECONDS" -ge "$deadline" ] && return 1
    sleep 1
  done
}

# mx_game_net CMD [ARG...] runs CMD in the running game's network namespace: scripts/launch-guard.sh starts the game
# under bwrap --unshare-net, so the 127.0.0.1 the bridge listens on is the game's own loopback, not the host's.
mx_game_net() {
  local pid
  pid=$(mx_game_pids | head -1)
  if [ -z "$pid" ]; then
    echo "BAD no game process to reach the bridge from"
    return 1
  fi
  nsenter --target "$pid" --user --net --preserve-credentials "$@"
}

# mx_bridge_reachable TAG PROFILE records whether the bridge answers at the main menu: its state file in the profile
# names a port and token, and a status query there answers ok with scene MainMenu. The state file is kept in the sandbox
# as bridge-state-TAG.json: the bridge removes it when the game exits.
mx_bridge_reachable() {
  local out state
  if ! mx_wait_scene MainMenu 120; then
    mx_fail "bridge.reachable.$1" "launch ($1): the game never reached MainMenu within 120s, so the bridge was not asked"
    return
  fi
  state="$(mx_dir "$2")/BepInEx/config/mortar-bepinex-bridge.json"
  cp "$state" "$ROOT/bridge-state-$1.json" 2>/dev/null
  out=$(
    mx_game_net python3 - "$state" 2>&1 <<'PY'
import json, socket, sys
try:
    st = json.load(open(sys.argv[1]))
except (OSError, ValueError) as e:
    sys.exit(print(f"BAD no state file: {e}"))
try:
    with socket.create_connection(("127.0.0.1", st["port"]), timeout=5) as s:
        s.sendall(f'{st["token"]}\nstatus\n'.encode())
        reply = s.makefile().readline().strip()
except OSError as e:
    sys.exit(print(f"BAD nothing answers on port {st['port']}: {e}"))
ok = reply.startswith("ok ") and '"scene":"MainMenu"' in reply
print(("OK " if ok else "BAD ") + f"port {st['port']} answered {reply[:200]}")
PY
  )
  mx_check "bridge.reachable.$1" "launch ($1) at MainMenu: ${out#* }" test "${out%% *}" = OK
}

# mx_bridge_relay PROFILE serves the bridge's port on the host's loopback, relaying each command into the game's
# network namespace, so Mortar's own bridge client (its live poll, MeasureInGame) reaches the bridge as it would on a
# player's machine; the game alone has its loopback here (scripts/launch-guard.sh). mx_relay holds the relay's pid.
mx_bridge_relay() {
  local state pid bport
  state="$(mx_dir "$1")/BepInEx/config/mortar-bepinex-bridge.json"
  pid=$(mx_game_pids | head -1)
  bport=$(mx_json "$state" 'd["port"]' 2>/dev/null)
  [ -n "$pid" ] && [ -n "$bport" ] || return 1
  python3 - "$bport" "$pid" >"$ROOT/relay-$1.log" 2>&1 <<'PY' &
import socket, subprocess, sys
port, pid = int(sys.argv[1]), sys.argv[2]
one = ("import socket,sys\ns=socket.create_connection(('127.0.0.1',%d),timeout=10)\n"
       "s.sendall(sys.stdin.buffer.read())\nsys.stdout.buffer.write(s.makefile('rb').readline())") % port
srv = socket.socket()
srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
srv.bind(("127.0.0.1", port))
srv.listen()
while True:
    c, _ = srv.accept()
    with c:
        try:
            f = c.makefile("rb")
            req = f.readline() + f.readline()
            out = subprocess.run(["nsenter", "--target", pid, "--user", "--net", "--preserve-credentials", "python3", "-c", one],
                                 input=req, capture_output=True, timeout=15)
            c.sendall(out.stdout)
            command = req.splitlines()[-1:]
            print(f"{command!r} -> {out.stdout[:80]!r} {out.stderr[-200:]!r}", flush=True)
        except Exception as e:
            print(f"relay: {e}", flush=True)
PY
  mx_relay=$!
  sleep 0.5
}

# mx_events_start TAG records the server's window events (the Wails runtime's /wails/events socket, which the window
# itself listens on) into $ROOT/events-TAG.jsonl until mx_events_stop.
mx_events_start() {
  cat >"$ROOT/events.js" <<'JS'
const [port, out] = process.argv.slice(2)
const fs = require('fs')
const ws = new WebSocket(`ws://127.0.0.1:${port}/wails/events`, { headers: { Origin: `http://127.0.0.1:${port}` } })
ws.onmessage = (e) => fs.appendFileSync(out, `${e.data}\n`)
ws.onerror = (e) => fs.appendFileSync(out, `${JSON.stringify({ error: String(e.message ?? e) })}\n`)
JS
  bun "$ROOT/events.js" "$PORT" "$ROOT/events-$1.jsonl" >/dev/null 2>&1 &
  mx_events=$!
}
mx_events_stop() { [ -z "${mx_events:-}" ] || kill "$mx_events" 2>/dev/null; mx_events=''; }
mx_relay_stop() { [ -z "${mx_relay:-}" ] || kill "$mx_relay" 2>/dev/null; mx_relay=''; }

# mx_startup_rows PROFILE LOG checks the measured launch's startup timing: the bridge's patcher deployed and timing,
# its report written at MainMenu with every loaded plugin and monotonic phases, and Mortar naming its rows by package.
mx_startup_rows() {
  local dir patcher report out
  dir=$(mx_dir "$1")
  patcher=$(find "$dir/BepInEx/patchers" -name MortarBepInExBridge.Patcher.dll 2>/dev/null | head -1)
  mx_check startup.patcher "patcher ${patcher#"$dir/"}; LogOutput.log: $(grep -m1 -o "Timing this launch's plugins[^\r]*" "$2" || echo 'no timing line'); initializer failures: $(grep -c 'Failed to run Initializer of MortarBepInExBridge' "$2")" \
    test -n "$patcher" -a -n "$(grep -m1 "Timing this launch's plugins until scene MainMenu" "$2")" -a "$(grep -c 'Failed to run Initializer of MortarBepInExBridge' "$2")" = 0
  for _ in $(seq 1 30); do
    report=$(find "$dir/startup" -maxdepth 1 -name '*Z.json' -newer "$ROOT/measure-requested" 2>/dev/null | sort | tail -1)
    [ -n "$report" ] && break
    sleep 1
  done
  cp "$report" "$ROOT/startup-a.json" 2>/dev/null
  out=$(
    python3 - "$report" "$2" "$dir/startup" "$(date -u +%s)" 2>&1 <<'PY'
import datetime, json, os, re, sys
path, log, startup, now = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
if not path:
    sys.exit(print("BAD no startup report was written"))
r = json.load(open(path))
loaded = len(re.findall(r"BepInEx\] Loading \[", open(log, errors="replace").read()))
p = r["phases"]
marks = [p["bridgeEntry"], p["entryDone"], p["gameLaunched"], p["titleScreen"]]
start = datetime.datetime.fromisoformat(r["processStart"].replace("Z", "+00:00")).timestamp()
left = [n for n in (".measure-launch", ".measure-next-launch") if os.path.exists(os.path.join(startup, n))]
ok = (len(r["mods"]) == loaded and r["entryMissed"] == 0 and all(m > 0 for m in marks) and marks == sorted(marks)
      and p["gameLaunched"] <= (p["titleMenu"] or p["gameLaunched"]) <= p["titleScreen"] and now - 900 < start < now and not left)
print(("OK " if ok else "BAD ") + f"{os.path.basename(path)}: {len(r['mods'])} rows for {loaded} Loading lines, entryMissed {r['entryMissed']}, "
      f"phases {p}, processStart {r['processStart']} ({now - start:.0f}s ago), markers left {left}")
PY
  )
  mx_check startup.report "${out#* }" test "${out%% *}" = OK
  mx_wails launchsvc.Service.StartupReports '"lethal-company"' "$(mx_q "$1")" >"$ROOT/startup-mortar-a.json"
  out=$(mx_json "$ROOT/startup-mortar-a.json" '(len(d[0]["mods"]), sum(m["id"].startswith("thunderstore:") for m in d[0]["mods"]), [m["id"] for m in d[0]["mods"] if not m["id"].startswith("thunderstore:")])' 2>&1)
  mx_check startup.mortar "Mortar's newest startup report: (rows, rows named by a package, the rest) = $out" \
    python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); m=d[0]["mods"]; sys.exit(not (m and all(x["id"].startswith("thunderstore:") for x in m)))' "$ROOT/startup-mortar-a.json"
}

# mx_exceptions LOG FROM [TO] counts exception lines in LOG between line numbers FROM and TO (the end without TO).
mx_exceptions() { sed -n "$2,${3:-\$}p" "$1" 2>/dev/null | grep -c 'Exception'; }

# mx_perf_rows PROFILE LOG measures in game through Mortar (MeasureInGame, via the relay) on the measured launch:
# the bridge wraps every plugin's patches and updates, a summary 10 s later has frame stats and package-named rows and
# is listed by `perf reports`, and wrapping raised no exception the 10 s before it had not.
mx_perf_rows() {
  local n0 n1 out wrapped e_before e_after mortar
  n0=$(wc -l <"$2")
  sleep 10
  n1=$(wc -l <"$2")
  e_before=$(mx_exceptions "$2" "$n0" "$n1")
  out=$(mx_wails launchsvc.Service.MeasureInGame '"lethal-company"' "$(mx_q "$1")" true 2>&1)
  for _ in $(seq 1 30); do
    wrapped=$(grep -m1 -o 'Measuring [0-9]* plugins: [0-9]* methods timed, [0-9]* could not be wrapped' "$2")
    [ -n "$wrapped" ] && break
    sleep 1
  done
  mx_check perf.start "MeasureInGame(start) answered $out; LogOutput.log: ${wrapped:-no Measuring line}" \
    python3 -c 'import re,sys; m=re.search(r"(\d+) methods timed, (\d+) could not", sys.argv[2]); sys.exit(not ("\"measured\":true" in sys.argv[1] and m and int(m[2]) <= max(5, int(m[1]) // 20)))' "$out" "$wrapped"
  n1=$(grep -n 'Measuring [0-9]* plugins:' "$2" | head -1 | cut -d: -f1)
  sleep 10
  mx_wails launchsvc.Service.MeasureInGame '"lethal-company"' "$(mx_q "$1")" false >"$ROOT/perf-a.json" 2>&1
  cli perf reports lethal-company "$1" >"$ROOT/perf-reports-a.txt" 2>&1
  cli mods lethal-company "$1" --json >"$ROOT/mods-a.json" 2>/dev/null
  out=$(
    python3 - "$ROOT/perf-a.json" "$ROOT/mods-a.json" "$ROOT/perf-reports-a.txt" 2>&1 <<'PY'
import json, sys
try:
    r = json.load(open(sys.argv[1]))["report"]
except (ValueError, KeyError, TypeError) as e:
    sys.exit(print(f"BAD no saved report: {open(sys.argv[1]).read()[:200]}"))
names = {m["name"] for m in json.load(open(sys.argv[2]))}
f, rows = r["frame"], r["rows"]
unnamed = [x["name"] for x in rows if x["name"] not in names]
listed = r["id"] in open(sys.argv[3]).read() and "fps over" in open(sys.argv[3]).read()
ok = f["fps"] > 0 and 0 < f["p50Ms"] <= f["p95Ms"] <= f["p99Ms"] <= f["maxMs"] and f["avgMs"] < 1000 and rows and not unnamed and listed
print(("OK " if ok else "BAD ") + f"{f['fps']:.0f} fps over {f['seconds']:.0f}s, frame ms p50 {f['p50Ms']} p95 {f['p95Ms']} p99 {f['p99Ms']} max {f['maxMs']}, "
      f"Mono {f['monoUsed'] >> 20}/{f['monoHeap'] >> 20} MiB, {f['gcCollections']} GCs; {len(rows)} rows, top {[(x['name'], round(x['averageMs'], 2)) for x in rows[:3]]}; "
      f"not a package name: {unnamed[:5]}; listed by perf reports with its frame line: {listed}")
PY
  )
  mx_check perf.summary "${out#* }" test "${out%% *}" = OK
  e_after=$(mx_exceptions "$2" "${n1:-$(wc -l <"$2")}")
  mortar=$(sed -n "${n1:-1},\$p" "$2" | grep -c 'MortarBepInExBridge\.Perf\|PerfHost')
  mx_check perf.stable "exception lines in LogOutput.log: $e_before in the 10s before wrapping, $e_after in the ${n1:+10s }after; $mortar naming the bridge's measuring" \
    test "${e_after:-0}" -le "${e_before:-0}" -a "$mortar" = 0
}

# mx_live_rows PROFILE checks the window's launch:live events during launch (a), recorded since the relay let Mortar
# reach the bridge: one names scene MainMenu, and its badges cover every enabled package that ships plugins, with
# exactly one of the two duplicate-GUID probes (BepInEx loads one copy) read as loaded.
mx_live_rows() {
  local out
  for _ in $(seq 1 20); do
    grep -q '"launch:live".*"scene":"MainMenu"' "$ROOT/events-a.jsonl" 2>/dev/null && break
    sleep 1
  done
  out=$(
    python3 - "$ROOT/events-a.jsonl" "$ROOT/mods-a.json" 2>&1 <<'PY'
import json, sys
live = []
for line in open(sys.argv[1]):
    try:
        e = json.loads(line)
    except ValueError:
        continue
    if e.get("name") == "launch:live":
        d = e.get("data")
        live.append(d[0] if isinstance(d, list) else d)
if not live:
    sys.exit(print("BAD no launch:live event"))
last = live[-1]
print(("OK " if last.get("scene") == "MainMenu" else "BAD ") + f"{len(live)} launch:live events, last scene {last.get('scene')!r}")
mods = {m["id"]: m["loaded"] for m in last.get("mods") or []}
ids = {m["id"] for m in json.load(open(sys.argv[2])) if m.get("enabled")}
dups = [mods.get(f"thunderstore:MortarMatrix-ProbeDup{x}") for x in "AB"]
foreign = [i for i in mods if i not in ids]
ok = mods and not foreign and sorted(map(bool, dups)) == [False, True]
print(("OK " if ok else "BAD ") + f"{len(mods)} packages badged, {sum(mods.values())} loaded; not loaded: {[i for i, l in mods.items() if not l][:8]}; "
      f"DupA/DupB loaded {dups}; badged but not an enabled mod: {foreign[:5]}")
PY
  )
  local scene badges
  scene=$(head -1 <<<"$out")
  badges=$(sed -n 2p <<<"$out")
  mx_check live.scene "${scene#* }" test "${scene%% *}" = OK
  mx_check live.badges "${badges:-no live event to read badges from}" test "${badges%% *}" = OK
}

# mx_purged TAG requires every entry the game folder had before the first launch to hash the same and the Doorstop
# proxy files to be gone. Files a mod itself writes into the game folder at run time (BoomboxController keeps its
# settings and a yt-dlp.exe there) are Mortar's to leave alone; they are listed in $ROOT/game-added-TAG.txt.
mx_purged() {
  tree_hash "$mx_game" >"$ROOT/game-after-$1.txt"
  diff "$ROOT/game-before.txt" "$ROOT/game-after-$1.txt" | sed -n 's/^> //p' >"$ROOT/game-added-$1.txt"
  local changed proxy
  changed=$(diff "$ROOT/game-before.txt" "$ROOT/game-after-$1.txt" | grep -c '^<')
  proxy=$(grep -cE '^f [0-7]+ \./(winhttp\.dll|doorstop_config\.ini) ' "$ROOT/game-added-$1.txt")
  if [ "$changed" -ne 0 ] || [ "$proxy" -ne 0 ]; then
    echo "$changed entries changed or removed, $proxy proxy files left; see $ROOT/game-after-$1.txt"
  fi
}

# mx_doorstop_files PROFILE VERSION compares the profile's Doorstop files with BepInExPack VERSION's own.
mx_doorstop_files() {
  local dir zip=$ROOT/bep-$2.zip
  dir=$(mx_dir "$1")
  if ! cmp -s <(unzip -p "$zip" BepInExPack/doorstop_config.ini) "$dir/doorstop_config.ini"; then
    echo "doorstop_config.ini is not the pack's"
  elif [ "$(unzip -p "$zip" BepInExPack/.doorstop_version 2>/dev/null)" != "$(cat "$dir/.doorstop_version" 2>/dev/null)" ]; then
    echo ".doorstop_version is \"$(cat "$dir/.doorstop_version" 2>/dev/null)\", the pack's is \"$(unzip -p "$zip" BepInExPack/.doorstop_version 2>/dev/null)\""
  else
    echo "the pack's doorstop_config.ini and .doorstop_version"
  fi
}
mx_doorstop_ok() { echo "the pack's doorstop_config.ini and .doorstop_version; the game folder holds the same ini; the game started with the Doorstop flags for this pack"; }

# mx_order_problems ORDER names each row of a `profile load-order --json` file placed before a plugin it needs. Rows
# are plugins keyed bepinex:GUID, not packages; which plugins they cover is checked against launch (a)'s Loading lines.
mx_order_problems() {
  python3 - "$1" <<'PY'
import json, sys
order = json.load(open(sys.argv[1]))
pos = {r["id"]: r["position"] for r in order}
bad = [f'{r["id"]} is not a plugin row' for r in order if not r["id"].startswith("bepinex:")]
bad += [f'{r["id"]} before its dependent {d}' for r in order for d in r.get("dependents", []) if pos.get(d, 0) < r["position"]]
bad += [f'{r["id"]} before its requirement {d}' for r in order for d in r.get("required", []) if pos.get(d, 0) > r["position"]]
print("; ".join(bad))
PY
}

# mx_order_unlisted ORDER LOG names each plugin BepInEx loaded in LOG that ORDER does not list, leaving out the bridge
# (Mortar's own, not one of the profile's mods) and the Matrix probes, installed after the order was read.
mx_order_unlisted() {
  python3 - "$1" "$2" <<'PY'
import json, re, sys
names = {r["name"] for r in json.load(open(sys.argv[1]))}
logged = re.findall(r"BepInEx\] Loading \[(.+) [^ \]]+\]", open(sys.argv[2], errors="replace").read())
print("; ".join(sorted({n for n in logged if n not in names and not n.startswith("Matrix ") and n != "Mortar BepInEx Bridge"})))
PY
}

# mx_doorstop PROFILE VERSION checks, while the game runs, the profile's Doorstop files against the pack's, the ini
# copied into the game folder, and the game's own command line: Doorstop 4 takes --doorstop-target-assembly, 3 --doorstop-target.
mx_doorstop() {
  local files flag target pid args
  files=$(mx_doorstop_files "$1" "$2")
  case $(unzip -p "$ROOT/bep-$2.zip" BepInExPack/.doorstop_version 2>/dev/null) in 4*) flag=--doorstop-target-assembly ;; *) flag=--doorstop-target ;; esac
  target="Z:$(mx_dir "$1" | sed 's:/:\\:g')\\BepInEx\\core\\BepInEx.Preloader.dll"
  pid=$(mx_game_pids | head -1)
  args=$(tr '\0' ' ' <"/proc/$pid/cmdline" 2>/dev/null)
  if [ "$files" != "the pack's doorstop_config.ini and .doorstop_version" ]; then
    echo "$files"
  elif ! cmp -s "$(mx_dir "$1")/doorstop_config.ini" "$mx_game/doorstop_config.ini"; then
    echo "the game folder's doorstop_config.ini differs from the profile's"
  elif [[ "$args" != *"$flag $target"* ]]; then
    echo "the game's command line lacks $flag $target: $args"
  else
    mx_doorstop_ok
  fi
}

# mx_files_and_flags PROFILE VERSION checks, with no game running, the profile's Doorstop files against BepInExPack
# VERSION's and the command line Mortar would start the game with (launchsvc PreviewCommand): Doorstop 4 takes
# --doorstop-target-assembly, 3 --doorstop-target, each with the profile's preloader in Z: form.
mx_files_and_flags() {
  local files flag target argv
  files=$(mx_doorstop_files "$1" "$2")
  case $(unzip -p "$ROOT/bep-$2.zip" BepInExPack/.doorstop_version 2>/dev/null) in 4*) flag=--doorstop-target-assembly ;; *) flag=--doorstop-target ;; esac
  target="Z:$(mx_dir "$1" | sed 's:/:\\:g')\\BepInEx\\core\\BepInEx.Preloader.dll"
  argv=$(mx_wails launchsvc.Service.PreviewCommand '"lethal-company"' "$(mx_q "$1")" '""' "$(mx_q "$ROOT/run-proton.sh")" '""' |
    python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["error"] or " ".join(d["argv"]))')
  if [ "$files" != "the pack's doorstop_config.ini and .doorstop_version" ]; then
    echo "$files"
  elif [[ "$argv" != *"$flag $target"* ]]; then
    echo "the previewed command line lacks $flag $target: $argv"
  else
    echo "the pack's doorstop_config.ini and .doorstop_version; the previewed command line carries $flag with the profile's preloader"
  fi
}
mx_files_and_flags_ok() { [[ "$1" == "the pack's doorstop_config.ini and .doorstop_version; the previewed"* ]]; }

# mx_stop TAG stops the game from Mortar and records whether it was a polite stop: the game's own process gone before
# anything is reaped, Unity's clean-exit statistics in Player.log, Mortar idle, and the game folder purged.
mx_stop() {
  local before after killed
  before=$(mx_game_pids | tr '\n' ' ')
  cli stop lethal-company >"$ROOT/stop-$1.txt" 2>&1 || mx_fail "stop.$1" "mortar stop failed: $(head -c 300 "$ROOT/stop-$1.txt")"
  sleep 2
  after=$(mx_game_pids | tr '\n' ' ')
  killed=$(reap_prefix "$mx_compat")
  mx_idle 60 || mx_fail "stop.$1" "Mortar never went idle after Stop"
  sleep 3
  local purge
  purge=$(mx_purged "$1")
  if [ -z "${after// /}" ] && grep -q '^Memory Statistics:' "$mx_saves/Player.log" && [ -z "$purge" ]; then
    mx_pass "stop.$1" "game pids ${before:-none} gone after Stop, Unity logged its clean-exit statistics, Mortar idle, game folder unchanged"
  else
    mx_fail "stop.$1" "after Stop: game pids left '${after}', reaped '${killed}', clean exit $(grep -c '^Memory Statistics:' "$mx_saves/Player.log" 2>/dev/null), purge: ${purge:-clean}"
  fi
  mx_keep_logs "$1"
}

# mx_build_probes compiles scripts/bepinex-probe against the copied game's assemblies and the profile's BepInEx
# with the .NET SDK's Roslyn, and packages each variant as a Thunderstore zip in $ROOT/probes.
mx_build_probes() {
  local src=$REPO/scripts/bepinex-probe out=$ROOT/probes managed="$mx_game/Lethal Company_Data/Managed"
  local core
  core="$(mx_dir "$mx_base")/BepInEx/core"
  local csc=${MORTAR_REGRESS_CSC:-}
  if [ -z "$csc" ]; then
    csc=$(dotnet --list-sdks 2>/dev/null | sed -n 's/^\([^ ]*\) \[\(.*\)\]$/\2\/\1\/Roslyn\/bincore\/csc.dll/p' | tail -1)
  fi
  [ -f "$csc" ] || {
    echo "no Roslyn csc.dll (install the .NET SDK or set MORTAR_REGRESS_CSC)"
    return 1
  }
  mkdir -p "$out/build"
  local refs=(-r:"$managed/mscorlib.dll" -r:"$managed/netstandard.dll" -r:"$managed/UnityEngine.dll"
    -r:"$managed/UnityEngine.CoreModule.dll" -r:"$managed/Assembly-CSharp-firstpass.dll" -r:"$core/BepInEx.dll")
  local v def
  for v in base beat nested own caps flat deep dup throw quit save; do
    def=PROBE
    case $v in beat) def=BEAT ;; throw) def=THROW ;; quit) def=QUIT ;; save) def=SAVE ;; esac
    printf 'static class Variant { public const string Name = "%s"; }\n' "$v" >"$out/build/Variant-$v.cs"
    dotnet "$csc" -nologo -noconfig -nostdlib -t:library -define:"$def" -out:"$out/build/$v.dll" "${refs[@]}" \
      "$src/Probe.cs" "$out/build/Variant-$v.cs" >"$out/build/$v.log" 2>&1 || {
      cat "$out/build/$v.log"
      return 1
    }
  done
  dotnet "$csc" -nologo -noconfig -nostdlib -t:library -out:"$out/build/patcher.dll" -r:"$managed/mscorlib.dll" \
    -r:"$managed/netstandard.dll" -r:"$core/BepInEx.dll" -r:"$core/Mono.Cecil.dll" "$src/Patcher.cs" >"$out/build/patcher.log" 2>&1 ||
    {
      cat "$out/build/patcher.log"
      return 1
    }
  # A Windows path past 260 characters inside the profile, which Proton and BepInEx then have to load.
  local winprofile deep
  winprofile="Z:$(mx_dir "$mx_base" | sed 's:/:\\:g')"
  deep="plugins/$(printf 'very-long-folder-name-%02d/' $(seq 1 $(((300 - ${#winprofile}) / 23 + 1))) | sed 's:/$::')"
  # name|zip path=build file ... ; a value with no build file is written as text.
  local specs=(
    "Base|plugins/Base.dll=base.dll|config/tech.rethunk.mortar.matrix.base.cfg=[General]\nValue = shipped\n\n# Setting type: Single\n# Default value: 0.5\nRatio = 0.5\n"
    "Beat|Beat.dll=beat.dll"
    "Nested|plugins/Deep/Er/Nested.dll=nested.dll"
    "Own|BepInEx/plugins/Own/Own.dll=own.dll|BepInEx/config/tech.rethunk.mortar.matrix.own.cfg=[General]\nValue = own-layout\n"
    "Caps|BEPINEX/Plugins/Caps.dll=caps.dll|BEPINEX/Config/tech.rethunk.mortar.matrix.caps.cfg=[General]\nValue = caps\n"
    "Flat|Lib/Flat.dll=flat.dll|Data/matrix-data.txt=beside\n"
    "Deep|$deep/Deep.dll=deep.dll"
    "Patcher|patchers/MatrixPatcher.dll=patcher.dll"
    "DupA|plugins/Dup.dll=dup.dll"
    "DupB|plugins/Dup.dll=dup.dll"
    "Throw|plugins/Throw.dll=throw.dll"
    "Quit|plugins/Quit.dll=quit.dll"
    "Save|plugins/Save.dll=save.dll"
  )
  local spec
  for spec in "${specs[@]}"; do
    python3 - "$out" "$spec" <<'PY' || return 1
import base64, json, os, sys, zipfile
out, spec = sys.argv[1], sys.argv[2].split("|")
name = spec[0]
icon = base64.b64decode("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
manifest = {"name": "Probe" + name, "version_number": "1.0.0", "website_url": "", "description": "Mortar matrix probe",
            "dependencies": ["BepInEx-BepInExPack-5.4.2100"]}
with zipfile.ZipFile(os.path.join(out, f"MortarMatrix-Probe{name}-1.0.0.zip"), "w") as z:
    z.writestr("manifest.json", json.dumps(manifest))
    z.writestr("icon.png", icon)
    for entry in spec[1:]:
        path, value = entry.split("=", 1)
        built = os.path.join(out, "build", value)
        if os.path.isfile(built):
            z.write(built, path)
        else:
            z.writestr(path, value.replace("\\n", "\n"))
PY
  done
}

mx_install_probes() {
  local p=$1 name
  shift
  for name in "$@"; do
    cli install lethal-company "$p" "$ROOT/probes/MortarMatrix-Probe$name-1.0.0.zip" >>"$ROOT/probe-install.txt" 2>&1 ||
      {
        echo "installing Probe$name failed: $(tail -c 300 "$ROOT/probe-install.txt")"
        return 1
      }
  done
}

# mx_problems PROFILE TAG writes the profile's Problems as JSON to $ROOT/problems-TAG.json.
mx_problems() { cli problems lethal-company "$1" --json >"$ROOT/problems-$2.json"; }

# regress_bepinex_prepare DATA COMPAT GAME TIMEOUT PROFILE BEPINEX runs every row that needs no game process, before
# the base launch: Mortar's data folder, the Proton compatdata folder, the copied game, the launch wait in seconds,
# the base run's profile id and the BepInEx version it pinned. The loader rows check the profile's files and the
# command line Mortar would start; the base profile then gets the modpack, the probe packages and two config edits,
# which launch (a) loads.
regress_bepinex_prepare() {
  mx_data=$1 mx_compat=$2 mx_game=$3 mx_timeout=$4 mx_base=$5 mx_bepinex=$6 mx_log=/dev/null mx_edge_edited='' mx_ready=''
  # Every item runs and records its own verdict, so one failing command must not end the run.
  set +e +o pipefail
  mx_saves="$mx_compat/pfx/drive_c/users/steamuser/AppData/LocalLow/ZeekerssRBLX/Lethal Company"
  : >"$ROOT/matrix.tsv"
  echo "---- BepInEx matrix: before launch (a)"
  local loader_dir
  loader_dir=$(mx_dir "$mx_base")

  # 1. Loader installs, checked on the profile's files and the previewed command line. The base run installed and
  # pinned $mx_bepinex; launch (a) proves it loads, launches (c) and (b) prove the newest pack with a throwing plugin and the older pinned pack load.
  local v out
  for v in 5.4.2100 "$mx_bepinex"; do
    curl -fsSL -o "$ROOT/bep-$v.zip" "https://thunderstore.io/package/download/BepInEx/BepInExPack/$v/" || mx_fail loader.download "BepInExPack $v did not download"
  done
  mx_check loader.fresh "BepInExPack $mx_bepinex in the profile: marker $(cat "$loader_dir/.mortar-bepinex.json"), $(mx_doorstop_files "$mx_base" "$mx_bepinex")" \
    test "$(mx_doorstop_files "$mx_base" "$mx_bepinex")" = "the pack's doorstop_config.ini and .doorstop_version"
  if cli loader install lethal-company 5.4.2100 >"$ROOT/loader-2100.txt" 2>&1 && cli loader pin lethal-company 5.4.2100 >/dev/null; then
    out=$(mx_files_and_flags "$mx_base" 5.4.2100)
    mx_check loader.pin "pinned 5.4.2100 over $mx_bepinex: $out" mx_files_and_flags_ok "$out"
  else
    mx_fail loader.pin "installing or pinning 5.4.2100 failed: $(head -c 300 "$ROOT/loader-2100.txt")"
  fi
  if cli loader pin lethal-company latest >/dev/null && cli loader install lethal-company "$mx_bepinex" >"$ROOT/loader-latest.txt" 2>&1; then
    out=$(mx_files_and_flags "$mx_base" "$mx_bepinex")
    mx_check loader.unpin "unpinned and installed $mx_bepinex: $out" mx_files_and_flags_ok "$out"
  else
    mx_fail loader.unpin "unpin or install $mx_bepinex failed: $(head -c 300 "$ROOT/loader-latest.txt")"
  fi
  if cli loader install lethal-company "$mx_bepinex" >"$ROOT/loader-again.txt" 2>&1; then
    out=$(mx_files_and_flags "$mx_base" "$mx_bepinex")
    mx_check loader.reinstall "reinstalled $mx_bepinex over itself: $out" mx_files_and_flags_ok "$out"
  else
    mx_fail loader.reinstall "$(head -c 300 "$ROOT/loader-again.txt")"
  fi
  # Launch (a) runs the pinned version the base run expects.
  cli loader pin lethal-company "$mx_bepinex" >/dev/null || mx_fail loader.repin "pinning $mx_bepinex again failed"

  # The profiles of launches (c) and (b) and of the rows that never launch, created now so each install reaches all of them.
  local p
  mx_crash=$(mx_profile "Matrix Crash")
  mx_pin=$(mx_profile "Matrix Pinned")
  mx_upd=$(mx_profile "Matrix Update")
  mx_upd2=$(mx_profile "Matrix Update Two")
  mx_dep=$(mx_profile "Matrix Deprecated")
  for p in "$mx_crash" "$mx_pin" "$mx_upd" "$mx_upd2" "$mx_dep"; do
    cli profile set lethal-company "$p" launchPrefix "$ROOT/run-proton.sh" >/dev/null
  done

  local probe_err
  if probe_err=$(mx_build_probes 2>&1); then
    mx_pass probes.build "built $(find "$ROOT/probes" -name '*.zip' | wc -l) probe packages with Roslyn from the .NET SDK"
  else
    mx_fail probes.build "$probe_err"
    set -e -o pipefail
    return
  fi

  # 2. The modpack, through the queue as Browse adds it, into the base profile.
  local pkg open
  for pkg in "${MATRIX_MODPACK[@]}"; do
    cli queue add lethal-company "$mx_base" "${pkg%/*}" --source thunderstore --version "${pkg##*/}" >>"$ROOT/queue-add.txt" 2>&1 ||
      mx_fail modpack.queue "queue add $pkg failed"
  done
  open=$(mx_wait_queue "$mx_base")
  cli mods lethal-company "$mx_base" --json >"$ROOT/pack-mods.json"
  local npack
  npack=$(mx_json "$ROOT/pack-mods.json" 'len(d)')
  mx_check modpack.install "${#MATRIX_MODPACK[@]} packages queued; profile holds $npack entries (dependencies, the base run's plugins and the bridge included); open: ${open:-none}" \
    test -z "$open" -a "$npack" -gt "${#MATRIX_MODPACK[@]}"
  cli profile load-order lethal-company "$mx_base" --json >"$ROOT/pack-order.json"
  out=$(mx_order_problems "$ROOT/pack-order.json")
  mx_check modpack.loadorder "load order lists $(mx_json "$ROOT/pack-order.json" 'len(d)') plugins, each after what it needs${out:+: $out}" test -z "$out"

  # 3. The probe packages beside the modpack: layout edge cases, the Console's heartbeat and a duplicated GUID.
  if ! probe_err=$(mx_install_probes "$mx_base" Base Beat Nested Own Caps Flat Deep Patcher DupA DupB); then
    mx_fail edge.install "$probe_err"
  else
    mx_ready=1
    mx_problems "$mx_base" before
    out=$(mx_json "$ROOT/problems-before.json" '[sorted(x["version"] for x in c["copies"]) for c in d.get("pluginClashes") or [] if c["guid"] == "tech.rethunk.mortar.matrix.dup"]')
    mx_check mods.dupguid "Problems flags tech.rethunk.mortar.matrix.dup with copies at versions $out" test "$out" = "[['1.0.0', '1.0.0']]"
    # The typed editor writes a string and a float before the launch; BepInEx reads every .cfg at startup and rewrites
    # it, so launch (a) logging both values and the file keeping them proves the edit is in BepInEx's own format.
    # Only a launch lays a package's plugins out in the profile; its shipped .cfg is seeded on the editor's first read.
    local set_err
    if set_err=$(mx_wails configsvc.Service.Set '"lethal-company"' "$(mx_q "$mx_base")" '"thunderstore:MortarMatrix-ProbeBase"' '"tech.rethunk.mortar.matrix.base.cfg"' '"General"' '"Value"' '"edited by mortar"' 2>&1) &&
      set_err=$(mx_wails configsvc.Service.Set '"lethal-company"' "$(mx_q "$mx_base")" '"thunderstore:MortarMatrix-ProbeBase"' '"tech.rethunk.mortar.matrix.base.cfg"' '"General"' '"Ratio"' '"0.75"' 2>&1); then
      mx_edge_edited=1
    else
      mx_fail mods.config "configsvc Set failed: $set_err"
    fi
    # Launch (a) is measured, as Measure next launch asks: the bridge's patcher times startup and its plugin measures
    # in game. The marker file dates the request, so only a report written after it counts.
    touch "$ROOT/measure-requested"
    out=$(mx_wails launchsvc.Service.MeasureNextLaunch '"lethal-company"' "$(mx_q "$mx_base")" 2>&1) ||
      mx_fail startup.patcher "MeasureNextLaunch failed: $out"
  fi
  set -e -o pipefail
}

# regress_bepinex_running LOG checks launch (a) while the game runs, from BepInEx's LogOutput.log at LOG: the loader
# version and Doorstop, the Console read live, the plugin count and every probe and modpack edge case loading.
regress_bepinex_running() {
  mx_log=$1
  set +e +o pipefail
  echo "---- BepInEx matrix: launch (a)"
  local out want
  out=$(grep -m1 -o 'BepInEx 5\.[0-9.]* - Lethal Company' "$mx_log")
  mx_check loader.reinstall-run "launch (a) on the reinstalled $mx_bepinex: log opens with \"$out\"; $(mx_doorstop "$mx_base" "$mx_bepinex")" \
    test "$out|$(mx_doorstop "$mx_base" "$mx_bepinex")" = "BepInEx $(mx_bep_version "$mx_bepinex") - Lethal Company|$(mx_doorstop_ok)"
  if [ -n "$mx_ready" ]; then
    mx_bridge_reachable a "$mx_base"
    mx_events_start a
    mx_bridge_relay "$mx_base" || mx_fail perf.start "no game process or bridge state file to relay Mortar's bridge client to"
    mx_startup_rows "$mx_base" "$mx_log"
    mx_perf_rows "$mx_base" "$mx_log"
    mx_live_rows "$mx_base"
    mx_relay_stop
    mx_events_stop
    mx_console_reads
    # The Console of a modpack this size outgrows the kernel's 128 KiB limit on one argument, so the reads go by file.
    printf '%s' "$mx_lines1" >"$ROOT/console-a1.json"
    printf '%s' "$mx_lines2" >"$ROOT/console-a2.json"
    out=$(
      python3 - "$ROOT/console-a1.json" "$ROOT/console-a2.json" 2>&1 <<'PY'
import json, sys
a, b = (json.load(open(f)) for f in sys.argv[1:3])
beats = lambda es: [e for e in es if e["message"].startswith("matrix heartbeat")]
levels = {e["level"] for e in b}
sources = {e["mod"] for e in b}
ok = (beats(a) and len(beats(b)) > len(beats(a)) and max(e["seq"] for e in b) > max(e["seq"] for e in a)
      and all(e["level"] == "WARN" and e["mod"] == "Matrix beat" for e in beats(b))
      and any(e["level"] == "ERROR" and e["message"] == "matrix error line" for e in b) and "BepInEx" in sources)
print(("OK " if ok else "BAD ") + f"heartbeats {len(beats(a))} then {len(beats(b))}, levels {sorted(levels)}, {len(sources)} sources")
PY
    )
    mx_check launch.console "Console lines read twice during launch (a), at $mx_read1_at and $mx_read2_at UTC: ${out#* }" test "${out%% *}" = OK
    for want in "base cfg=edited by mortar:mods.config" "base ratio=0.75:mods.config-float" "own cfg=own-layout:edge.own-layout" "caps cfg=caps:edge.case" "flat beside=True:edge.flatten-probe"; do
      mx_check "${want#*:}" "LogOutput.log: $(grep -m1 -o "matrix ${want%%:*}" "$mx_log" || echo "no \"matrix ${want%%:*}\"")" grep -q "matrix ${want%%:*}" "$mx_log"
    done
    mx_bridge_survived a "$mx_log"
    mx_check edge.config-placement "config placed flat: $(grep -m1 -o 'matrix base cfg=[^\r]*' "$mx_log" || echo 'no "matrix base cfg="')" grep -q 'matrix base cfg=' "$mx_log"
    mx_check edge.nested "plugins/Deep/Er/Nested.dll: $(grep -m1 -o 'Loading \[Matrix nested[^]]*\]' "$mx_log" || echo not loaded)" grep -q 'Loading \[Matrix nested' "$mx_log"
    mx_check edge.longpath "a plugin $(find "$(mx_dir "$mx_base")/BepInEx/plugins" -name Deep.dll | awk '{print length("Z:" $0)}') characters deep as Wine sees it: $(grep -m1 -o 'Loading \[Matrix deep[^]]*\]' "$mx_log" || echo not loaded)" grep -q 'Loading \[Matrix deep' "$mx_log"
    mx_check edge.patcher "patchers/MatrixPatcher.dll: $(grep -m1 -o 'matrix patcher initialized' "$mx_log" || echo not initialized)" grep -q 'matrix patcher initialized' "$mx_log"
    mx_check mods.dupguid-runtime "BepInEx loaded the duplicated GUID $(grep -c 'Loading \[Matrix dup' "$mx_log") time(s)" test "$(grep -c 'Loading \[Matrix dup' "$mx_log")" = 1
  fi
  local planned loading skipped
  planned=$(grep -oE 'BepInEx\] [0-9]+ plugins? to load' "$mx_log" | grep -oE '[0-9]+' | head -1)
  loading=$(grep -cE 'BepInEx\] Loading \[' "$mx_log")
  skipped=$(grep -cE 'BepInEx\] (Skipping|Could not load) \[' "$mx_log")
  mx_check launch.count "BepInEx counted ${planned:-no} plugins to load, logged $loading Loading lines and skipped $skipped: $(grep -oE 'Skipping \[[^]]*\] because [^(]*' "$mx_log" | head -3 | tr '\n' ';')" \
    test "${planned:-x}" = "$((loading + skipped))"
  out=$(mx_order_unlisted "$ROOT/pack-order.json" "$mx_log")
  mx_check modpack.loadorder-run "the load order lists every modpack plugin launch (a) loaded${out:+; not listed: $out}" \
    test "$loading" -gt 0 -a -z "$out"
  mx_check edge.flatten "MirageCore ships FSharp.Core/FSharp.Core.dll and Mirage loads it from beside its plugin: $(grep -c 'FSharp.Core.dll.*or one of its dependencies' "$mx_saves/Player.log") load failures" \
    test "$(grep -c 'FSharp.Core.dll.*or one of its dependencies' "$mx_saves/Player.log")" = 0
  mx_check edge.patchers "the modpack's preloader patchers loaded: $(grep -c 'Loaded 1 patcher method from' "$mx_log") patcher lines" \
    grep -q 'Loaded 1 patcher method from \[BepInEx.MonoMod.AutoHookGenPatcher' "$mx_log"
  set -e -o pipefail
}

# regress_bepinex_after NAME runs once launch (a) is stopped (mx_stop a): the rows that read what (a) left behind,
# launch (b), and the rows that need no game. NAME is the base profile's name, which the LAN receiver looks up.
regress_bepinex_after() {
  local name=$1 out cfgdir
  set +e +o pipefail
  echo "---- BepInEx matrix: after launch (a)"
  cfgdir="$(mx_dir "$mx_base")/BepInEx/config"
  if [ -n "$mx_edge_edited" ]; then
    mx_check mods.config-file "after BepInEx rewrote the probe's .cfg: \"$(grep -m1 '^Value =' "$cfgdir/tech.rethunk.mortar.matrix.base.cfg")\", \"$(grep -m1 '^Ratio =' "$cfgdir/tech.rethunk.mortar.matrix.base.cfg")\"" \
      grep -q '^Ratio = 0.75' "$cfgdir/tech.rethunk.mortar.matrix.base.cfg"
  fi
  # settings.json keeps lastPlayed in its global block.
  out=$(mx_json "$mx_data/settings.json" 'd.get("global", {}).get("lastPlayed", {}).get("lethal-company", {})' 2>&1)
  mx_check lastplayed.version "lastPlayed[lethal-company] after launch (a) ($mx_base): $out" \
    test "$(mx_json "$mx_data/settings.json" 'd.get("global", {}).get("lastPlayed", {}).get("lethal-company", {}).get("profile", "")' 2>/dev/null)" = "$mx_base" \
    -a -n "$(mx_json "$mx_data/settings.json" 'd.get("global", {}).get("lastPlayed", {}).get("lethal-company", {}).get("gameVersion", "")' 2>/dev/null)"
  if go -C "$REPO" test ./internal/dotnet -run '^TestRealBepInExRun$' -count=1 -v -bepinex-profile "$(mx_dir "$mx_base")" >"$ROOT/guid-test.txt" 2>&1; then
    mx_pass mods.guids "$(grep -o '[0-9]* plugins found.*' "$ROOT/guid-test.txt")"
  else
    mx_fail mods.guids "$(grep -E 'realrun_test|FAIL' "$ROOT/guid-test.txt" | head -5 | tr '\n' ' ')"
  fi
  mx_problems "$mx_base" pack
  out=$(
    python3 - "$ROOT/problems-pack.json" "$ROOT/pack-mods.json" "$ROOT/LogOutput-a.log" "$ROOT/Player-a.log" <<'PY'
import json, re, sys, urllib.request
p = json.load(open(sys.argv[1]))
mods = {m["id"]: m for m in json.load(open(sys.argv[2]))}
logs = {"LogOutput.log": open(sys.argv[3], errors="replace").read().splitlines(),
        "Player.log": open(sys.argv[4], errors="replace").read().splitlines()}
wrong = []
for m in p.get("missing") or []:
    if m["id"] in mods and mods[m["id"]]["enabled"]:
        wrong.append(f'missing {m["id"]} is installed and enabled')
for d in p.get("deprecated") or []:
    ns, name = d["name"].split("-", 1)
    req = urllib.request.Request(f"https://thunderstore.io/api/experimental/package/{ns}/{name}/",
                                 headers={"User-Agent": "Mortar matrix regress (+https://mortar.rethunk.tech)"})
    try:
        with urllib.request.urlopen(req) as r:
            if not json.load(r).get("is_deprecated"):
                wrong.append(f'{d["name"]} is not deprecated on Thunderstore')
    except OSError as e:
        wrong.append(f'{d["name"]}: Thunderstore did not answer ({e})')
for c in p.get("pluginClashes") or []:
    if len({x["key"] for x in c["copies"]}) < 2:
        wrong.append(f'clash {c["guid"]} has one copy')
for f in p.get("loadFailures") or []:
    if f.get("line") and not any(f["plugin"].split()[0] in l or f["message"][:40] in l for l in sum(logs.values(), [])):
        wrong.append(f'load failure {f["plugin"]}: "{f["message"][:60]}" is in neither log')
    if not f.get("key"):
        wrong.append(f'load failure {f["plugin"]} names no installed package')
for k in ("broken", "duplicates", "damaged", "drift"):
    for x in p.get(k) or []:
        wrong.append(f"{k}: {json.dumps(x)[:120]}")
kinds = {k: len(v) for k, v in p.items() if isinstance(v, list) and v and k != "timings"}
print(json.dumps(kinds))
for w in wrong:
    print("WRONG " + w)
PY
  )
  mx_check problems.true "Problems after launch (a) lists $(echo "$out" | head -1); each checked against Thunderstore, the logs and the mod list$(echo "$out" | grep '^WRONG' | sed 's/^WRONG /; /' | tr -d '\n')" \
    test -z "$(echo "$out" | grep '^WRONG')" -a "${out:0:1}" = "{"

  # 4. Launches (c) and (b) differ in one variable each from launch (a): (c) is the newest pack with a plugin that throws
  # in Awake, and (b) the older pack pinned with no throwing plugin. Both also run a plugin that quits the game, and
  # (b) one that writes the game's own save. Row bridge.reachable asks the bridge at the main menu, before the
  # Quit probe's clock, which starts there, runs out. (c) runs first, while the newest pack is pinned.
  if [ -n "$mx_ready" ] && mx_install_probes "$mx_crash" Throw Quit >/dev/null && mx_launch "$mx_crash" c; then
    mx_bridge_reachable c "$mx_crash"
    # (c) was not measured: its bridge measures nothing and its patcher writes no startup report. The Quit probe ends
    # the game 8 s after the main menu, so this asks at once.
    if mx_bridge_relay "$mx_crash"; then
      out=$(mx_wails launchsvc.Service.MeasureInGame '"lethal-company"' "$(mx_q "$mx_crash")" true 2>&1)
      mx_relay_stop
    else
      out="no game process or bridge state file to relay to"
    fi
    mx_check perf.unmeasured "launch (c), unmeasured: MeasureInGame(start) answered $out; startup reports: $(find "$(mx_dir "$mx_crash")/startup" -name '*Z.json' 2>/dev/null | wc -l)" \
      test "$out|$(find "$(mx_dir "$mx_crash")/startup" -name '*Z.json' 2>/dev/null | wc -l)" = '{"measured":false}|0'
    if mx_idle 90; then
      mx_keep_logs c
      cli runs lethal-company "$mx_crash" --json >"$ROOT/runs-c.json"
      mx_check launch.exit "launch (c): the game quit by itself; Mortar went idle and recorded outcome $(mx_json "$ROOT/runs-c.json" 'd[0]["outcome"]'); game folder: $(mx_purged c || true)" \
        test -z "$(mx_purged c)" -a "$(mx_json "$ROOT/runs-c.json" 'd[0]["outcome"]')" = ran
      mx_problems "$mx_crash" crash
      out=$(mx_json "$ROOT/problems-crash.json" '[(f["name"], f["kind"], f["message"][:60]) for f in d.get("loadFailures") or []]')
      mx_check launch.crash "Problems: $out; run: loader $(mx_json "$ROOT/runs-c.json" 'd[0]["loaderVersion"]'), $(mx_json "$ROOT/runs-c.json" 'd[0]["errors"]') errors" \
        python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); r=json.load(open(sys.argv[2]))[0]; sys.exit(not (any(f["name"]=="ProbeThrow" and "matrix probe failed in Awake" in f["message"] for f in d.get("loadFailures") or []) and r["loaderVersion"] and r["errors"]>0))' "$ROOT/problems-crash.json" "$ROOT/runs-c.json"
    else
      mx_fail launch.exit "launch (c): Mortar still reports $(mx_state) 90s after the game should have quit"
      mx_stop c
    fi
    mx_bridge_survived c "$mx_log"
    reap_prefix "$mx_compat" >/dev/null
  else
    mx_fail launch.c "launch (c) did not start"
  fi

  rm -f "$mx_saves/LCSaveFile1"
  if [ -n "$mx_ready" ] && cli loader install lethal-company 5.4.2100 >"$ROOT/loader-2100-b.txt" 2>&1 &&
    cli loader pin lethal-company 5.4.2100 >/dev/null && mx_install_probes "$mx_pin" Save Quit >/dev/null &&
    mx_launch "$mx_pin" b; then
    out=$(grep -m1 -o 'BepInEx 5\.[0-9.]* - Lethal Company' "$mx_log")
    mx_check loader.pin-run "launch (b) on the pinned 5.4.2100: log opens with \"$out\"; $(mx_doorstop "$mx_pin" 5.4.2100)" \
      test "$out|$(mx_doorstop "$mx_pin" 5.4.2100)" = "BepInEx $(mx_bep_version 5.4.2100) - Lethal Company|$(mx_doorstop_ok)"
    mx_bridge_reachable b "$mx_pin"
    if mx_idle 90; then
      mx_keep_logs b
      cli runs lethal-company "$mx_pin" --json >"$ROOT/runs-b.json"
      mx_check loader.pin-exit "launch (b): the game quit by itself on the pinned pack; Mortar went idle and recorded outcome $(mx_json "$ROOT/runs-b.json" 'd[0]["outcome"]'); game folder: $(mx_purged b || true)" \
        test -z "$(mx_purged b)" -a "$(mx_json "$ROOT/runs-b.json" 'd[0]["outcome"]')" = ran
    else
      mx_fail loader.pin-exit "launch (b): Mortar still reports $(mx_state) 90s after the game should have quit"
      mx_stop b
    fi
    mx_bridge_survived b "$mx_log"
    reap_prefix "$mx_compat" >/dev/null
  else
    mx_fail launch.b "launch (b) did not start: $(head -c 300 "$ROOT/loader-2100-b.txt" 2>/dev/null)"
  fi
  if ! cli loader pin lethal-company "$mx_bepinex" >/dev/null || ! cli loader install lethal-company "$mx_bepinex" >/dev/null 2>&1; then
    mx_fail loader.repin "pinning $mx_bepinex again after launch (b) failed"
  fi

  # 5. Updates with the save the game wrote in launch (b).
  mx_check saves.fixture "the game wrote LCSaveFile1 ($(stat -c %s "$mx_saves/LCSaveFile1" 2>/dev/null || echo 0) bytes); Saves lists $(cli saves lethal-company "$mx_upd" --json | python3 -c 'import json,sys; print([s["folder"] for s in json.load(sys.stdin)])')" \
    test -s "$mx_saves/LCSaveFile1"
  local save_sum x
  save_sum=$(sha256sum "$mx_saves/LCSaveFile1" 2>/dev/null | cut -d' ' -f1)
  for x in AinaVT-LethalConfig:1.4.5 FlipMods-BetterStamina:1.5.6 Rune580-LethalCompany_InputUtils:0.7.12; do
    cli queue add lethal-company "$mx_upd" "${x%:*}" --source thunderstore --version "${x#*:}" >/dev/null
  done
  cli queue add lethal-company "$mx_upd2" FlipMods-BetterStamina --source thunderstore --version 1.5.6 >/dev/null
  mx_wait_queue "$mx_upd" >/dev/null
  mx_wait_queue "$mx_upd2" >/dev/null
  cli updates lethal-company "$mx_upd" --json >"$ROOT/updates.json"
  local backups_before
  backups_before=$(cli backups list --game lethal-company --json | python3 -c 'import json,sys; print(len(json.load(sys.stdin) or []))')
  cli update lethal-company "$mx_upd" --all >"$ROOT/update-all.txt" 2>&1
  mx_wait_queue "$mx_upd" >/dev/null
  cli mods lethal-company "$mx_upd" --json >"$ROOT/upd-mods.json"
  mx_check updates.all "updates listed $(mx_json "$ROOT/updates.json" '[u["package"] + " " + u["installed"] + "->" + u["version"] for u in d["updates"]]'); after Update all: $(mx_json "$ROOT/upd-mods.json" '[m["name"] + " " + m["version"] for m in d if m["source"] == "thunderstore"]')" \
    test "$(mx_json "$ROOT/upd-mods.json" 'sorted(m["version"] for m in d if m["source"] == "thunderstore")')" = "['0.7.13', '1.4.6', '1.5.7']"
  cli backups list --game lethal-company --json >"$ROOT/backups.json"
  mx_check updates.backup "backups before $backups_before, after $(mx_json "$ROOT/backups.json" 'len(d)'); newest is $(mx_json "$ROOT/backups.json" '(d[0]["kind"], [s["folder"] for s in d[0]["saves"]])')" \
    test "$(mx_json "$ROOT/backups.json" '(d[0]["kind"], [s["folder"] for s in d[0]["saves"]])')" = "('update', ['LCSaveFile1'])"
  local undo
  undo=$(sed -n 's/^Undo all: mortar profile revert lethal-company "[^"]*" \(.*\)$/\1/p' "$ROOT/update-all.txt")
  if [ -n "$undo" ] && cli profile revert lethal-company "$mx_upd" "$undo" >/dev/null 2>&1; then
    cli mods lethal-company "$mx_upd" --json >"$ROOT/upd-reverted.json"
    mx_check updates.rollback "Undo all ($undo) put back $(mx_json "$ROOT/upd-reverted.json" '[m["name"] + " " + m["version"] for m in d if m["source"] == "thunderstore"]')" \
      test "$(mx_json "$ROOT/upd-reverted.json" 'sorted(m["version"] for m in d if m["source"] == "thunderstore")')" = "['0.7.12', '1.4.5', '1.5.6']"
  else
    mx_fail updates.rollback "no undo id in: $(head -c 300 "$ROOT/update-all.txt")"
  fi
  cli updates apply --everywhere lethal-company thunderstore:FlipMods-BetterStamina >"$ROOT/update-everywhere.txt" 2>&1
  mx_wait_queue "$mx_upd2" >/dev/null
  mx_check updates.everywhere "$(tr '\n' ';' <"$ROOT/update-everywhere.txt" | tr -s ' ')" \
    test "$(cli mods lethal-company "$mx_upd2" --json | python3 -c 'import json,sys; print([m["version"] for m in json.load(sys.stdin) if m["name"] == "BetterStamina"])')" = "['1.5.7']" -a \
    "$(cli mods lethal-company "$mx_upd" --json | python3 -c 'import json,sys; print([m["version"] for m in json.load(sys.stdin) if m["name"] == "BetterStamina"])')" = "['1.5.7']"
  head -c 64 /dev/urandom >"$mx_saves/LCSaveFile1"
  local newest
  newest=$(mx_json "$ROOT/backups.json" 'd[0]["name"]')
  cli backups restore "$newest" LCSaveFile1 >"$ROOT/backup-restore.txt" 2>&1
  mx_check updates.restore "restoring $newest gives LCSaveFile1 back byte for byte" test "$(sha256sum "$mx_saves/LCSaveFile1" | cut -d' ' -f1)" = "$save_sum"

  # 6. Deprecated packages and their named replacements.
  cli queue add lethal-company "$mx_dep" VirusTLNR-MaskedFixes --source thunderstore --version 0.0.3 >/dev/null
  mx_wait_queue "$mx_dep" >/dev/null
  mx_problems "$mx_dep" dep
  out=$(mx_json "$ROOT/problems-dep.json" '{x["name"]: x.get("replacement", "") for x in d.get("deprecated") or []}')
  mx_check mods.deprecated "Problems: $out" test "$(mx_json "$ROOT/problems-dep.json" '{x["name"]: x.get("replacement", "") for x in d.get("deprecated") or []}.get("VirusTLNR-MaskedFixes")')" = VirusTLNR-MaskedInvisFix

  # 7. Sharing the base profile, which holds Thunderstore packages and the local probe packages.
  regress_bepinex_sharing "$mx_base" "$name"

  set -e -o pipefail
  echo "---- BepInEx matrix: $(grep -c "$(printf '\tPASS\t')" "$ROOT/matrix.tsv") PASS, $(grep -c "$(printf '\tFAIL\t')" "$ROOT/matrix.tsv") FAIL ($ROOT/matrix.tsv)"
}

# regress_bepinex_sharing PROFILE NAME shares the base profile: a link and a .mortar file carry its Thunderstore
# packages, a LAN send its local probe packages and edited config too.
regress_bepinex_sharing() {
  local pack=$1 name=$2 out
  # A share link carries the Thunderstore packages and imports into a new profile through the window's own calls.
  local link session res
  link=$(cli share lethal-company "$pack")
  session=$(mx_wails sharesvc.Service.PreviewLink '"lethal-company"' "$(mx_q "$link")" '""' | python3 -c 'import json,sys; print(json.load(sys.stdin)["session"])')
  res=$(mx_wails sharesvc.Service.Import '"lethal-company"' "$(mx_q "$session")" '""' null)
  out=$(
    python3 - "$res" "$ROOT/pack-mods.json" <<'PY'
import json, sys
res = json.loads(sys.argv[1])
src = sorted(m["id"] for m in json.load(open(sys.argv[2])) if m["source"] == "thunderstore")
got = sorted(m["id"] for e in res["profile"]["entries"] for m in e["mods"] if e["source"]["kind"] == "thunderstore")
print(res["profile"]["id"], len(src), len(got), "same" if src == got else f"differs: missing {sorted(set(src)-set(got))[:5]} extra {sorted(set(got)-set(src))[:5]}")
PY
  )
  mx_check share.link "link of ${#link} characters imported as profile ${out%% *}: ${out#* } (source, imported, verdict)" test "${out##* }" = same

  # A .mortar file of the same profile.
  cli export lethal-company "$pack" "$ROOT/pack.mortar" >/dev/null
  session=$(mx_wails sharesvc.Service.PreviewFile '"lethal-company"' "$(mx_q "$ROOT/pack.mortar")" '""' | python3 -c 'import json,sys; print(json.load(sys.stdin)["session"])')
  res=$(mx_wails sharesvc.Service.Import '"lethal-company"' "$(mx_q "$session")" '""' null)
  out=$(
    python3 - "$res" "$ROOT/pack-mods.json" <<'PY'
import json, sys
res = json.loads(sys.argv[1])
src = sorted(m["id"] for m in json.load(open(sys.argv[2])) if m["source"] == "thunderstore")
got = sorted(m["id"] for e in res["profile"]["entries"] for m in e["mods"] if e["source"]["kind"] == "thunderstore")
print(len(src), len(got), "same" if src == got else "differs")
PY
  )
  mx_check share.mortar "pack.mortar ($(stat -c %s "$ROOT/pack.mortar") bytes) imported: $out" test "${out##* }" = same

  # An r2modman code is published on thunderstore.io, so it is made only when asked for.
  if [ -n "${MORTAR_REGRESS_R2_EXPORT:-}" ]; then
    local key
    key=$(mx_wails packsvc.Service.ExportCode '"lethal-company"' "$(mx_q "$pack")" true | tr -d '"')
    cli profile import "$key" --preview --json >"$ROOT/r2-roundtrip.json" 2>&1
    out=$(
      python3 - "$ROOT/r2-roundtrip.json" "$ROOT/pack-mods.json" <<'PY'
import json, sys
pv = json.load(open(sys.argv[1]))
src = sorted(m["id"].split(":", 1)[1].lower() for m in json.load(open(sys.argv[2])) if m["source"] == "thunderstore")
got = sorted(p["native"].lower() for p in pv["packages"])
loader = "bepinex-bepinexpack" in got
got = [g for g in got if g != "bepinex-bepinexpack"]
print(len(src), len(got), "loader" if loader else "no-loader", "same" if src == got else "differs")
PY
    )
    mx_check share.r2code "code $key read back: $out" test "$out" = "$(echo "$out" | cut -d' ' -f1-2) loader same"
  else
    mx_pass share.r2code "skipped: set MORTAR_REGRESS_R2_EXPORT=1 to publish a code to thunderstore.io; export.r2x format and BepInExPack entry: TestExportCodeCarriesTheConfigFolder"
  fi

  # A LAN send to a second paired sandbox carries the local probe packages' files and the profile's configs.
  regress_bepinex_lan "$pack" "$name"
}

# regress_bepinex_lan PROFILE NAME starts a second server in $ROOT/peer with its own home, pairs it with this one and
# sends the base profile, whose probe packages exist only as local files.
regress_bepinex_lan() {
  local edge=$1 name=$2 peer=$ROOT/peer peer_port pa pb code
  # The send carries the config the edge step edited; without that edit there is nothing to look for at the receiver.
  if [ -z "${mx_edge_edited:-}" ]; then
    mx_fail share.lan "not run: the base profile holds no edited config (edge.install or mods.config failed first)"
    return
  fi
  peer_port=$((PORT + 1))
  while [ -n "$(ss -ltn "sport = :$peer_port" | tail -n +2)" ]; do peer_port=$((peer_port + 1)); done
  pa=$((47600 + RANDOM % 300))
  pb=$((pa + 1))
  mkdir -p "$peer/home/.local/share/Steam/steamapps" "$peer/home/.local/share/Steam/config"
  cp "$SANDBOX_STEAM/steamapps/appmanifest_$LC_APP_ID.acf" "$peer/home/.local/share/Steam/steamapps/"
  printf '"libraryfolders"\n{\n\t"0"\n\t{\n\t\t"path"\t\t"%s"\n\t\t"apps"\n\t\t{\n\t\t\t"%s"\t\t"1"\n\t\t}\n\t}\n}\n' "$SANDBOX_STEAM" "$LC_APP_ID" \
    >"$peer/home/.local/share/Steam/steamapps/libraryfolders.vdf"
  (cd "$ROOT" && env -u XDG_DATA_HOME -u XDG_CONFIG_HOME -u XDG_CACHE_HOME HOME="$peer/home" PATH="$ROOT/bin:$PATH" \
    WAILS_SERVER_HOST=127.0.0.1 WAILS_SERVER_PORT="$peer_port" nohup ./mortar-server >"$peer/server.log" 2>&1 &)
  peercli() { env -u XDG_DATA_HOME -u XDG_CONFIG_HOME -u XDG_CACHE_HOME HOME="$peer/home" "$ROOT/mortar-server" "$@"; }
  for _ in $(seq 1 30); do
    peercli games >/dev/null 2>&1 && break
    sleep 1
  done
  cli settings set lanSharing true >/dev/null && cli settings set lanPort "$pa" >/dev/null
  peercli settings set lanSharing true >/dev/null && peercli settings set lanPort "$pb" >/dev/null
  sleep 2
  cli lan pair >"$ROOT/lan-pair.txt" 2>&1 &
  local pairing=$!
  for _ in $(seq 1 20); do
    code=$(sed -n 's/^Code: \([^ ]*\) .*/\1/p' "$ROOT/lan-pair.txt")
    [ -n "$code" ] && break
    sleep 0.5
  done
  peercli lan pair --code "$code" --peer "127.0.0.1:$pa" >"$ROOT/lan-pair-peer.txt" 2>&1
  wait "$pairing"
  local sent=no accepted="" got
  if cli lan send lethal-company "$edge" "127.0.0.1:$pb" >"$ROOT/lan-send.txt" 2>&1; then
    sent=yes
    for _ in $(seq 1 20); do
      accepted=$(peercli lan inbox --json | python3 -c 'import json,sys; d=json.load(sys.stdin) or []; print(d[0]["id"] if d else "")')
      [ -n "$accepted" ] && break
      sleep 1
    done
    [ -n "$accepted" ] && peercli lan accept "$accepted" --json >"$ROOT/lan-accept.json" 2>&1
  fi
  got=$(peercli mods lethal-company "$name" --json 2>/dev/null | python3 -c 'import json,sys; print(sorted(m["name"] for m in json.load(sys.stdin) if m["source"] == "local"))' 2>/dev/null)
  local want
  want=$(cli mods lethal-company "$edge" --json | python3 -c 'import json,sys; print(sorted(m["name"] for m in json.load(sys.stdin) if m["source"] == "local"))')
  local cfg
  cfg=$(find "$peer/home/.local/share/mortar/profiles/lethal-company" -name tech.rethunk.mortar.matrix.base.cfg 2>/dev/null | head -1)
  local edited
  edited=$(grep -c 'Value = edited by mortar' "$cfg" 2>/dev/null || true)
  mx_check share.lan "paired ($(head -1 "$ROOT/lan-pair-peer.txt")), sent $sent, accepted ${accepted:-nothing}; receiver's local packages $got; edited config ${edited:-0} line(s)${cfg:+ in $cfg}" \
    test "$got" = "$want" -a "${edited:-0}" -gt 0
  local ppid
  ppid=$(ss -ltnp "sport = :$peer_port" | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2)
  if [ -n "$ppid" ] && [ "$(readlink "/proc/$ppid/exe")" = "$ROOT/mortar-server" ]; then kill "$ppid"; fi
}
