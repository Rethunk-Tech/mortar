# shellcheck shell=bash
# The BepInEx test matrix (docs/bepinex-test-matrix.md), run by `MORTAR_REGRESS_MATRIX=1 scripts/selftest.sh regress
# --game lethal-company` after the base run. selftest.sh sources this file; regress_lc calls regress_bepinex_matrix
# with its sandbox set up (ROOT, PORT, data, game, compat, profile, timeout) and its helpers defined (cli, reap_prefix,
# tree_hash). Each matrix item ends as one PASS or FAIL row in $ROOT/matrix.tsv; a FAIL is also a regress failure.
# The modpack and the update packages come from Thunderstore, so the matrix needs the network.

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

mx_result() {
  printf '%s\t%s\t%s\n' "$1" "$2" "$3" >>"$ROOT/matrix.tsv"
  echo "matrix $2 $1: $3"
  [ "$2" = PASS ] || failures+=("matrix $1: $3")
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

# mx_launches prints how many games the matrix starts (each mx_launch call below), which the session's launch cap must
# have left before the run begins (scripts/selftest.sh need_launches).
mx_launches() { echo 8; }

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
  cp "$mx_log" "$ROOT/LogOutput-$1.log" 2>/dev/null || true
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
    "Base|plugins/Base.dll=base.dll|config/tech.rethunk.mortar.matrix.base.cfg=[General]\nValue = shipped\n"
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

# regress_bepinex_matrix DATA COMPAT GAME TIMEOUT PROFILE BEPINEX runs the matrix: Mortar's data folder, the Proton
# compatdata folder, the copied game, the launch wait in seconds, the base run's profile id and the BepInEx version.
regress_bepinex_matrix() {
  mx_data=$1 mx_compat=$2 mx_game=$3 mx_timeout=$4 mx_base=$5 mx_bepinex=$6 mx_log=/dev/null mx_edge_edited=
  # Every item runs and records its own verdict, so one failing command must not end the run.
  set +e +o pipefail
  mx_saves="$mx_compat/pfx/drive_c/users/steamuser/AppData/LocalLow/ZeekerssRBLX/Lethal Company"
  : >"$ROOT/matrix.tsv"
  echo "---- BepInEx matrix"
  local loader_dir
  loader_dir=$(mx_dir "$mx_base")

  # 1. Loader installs. The base run installed and pinned $mx_bepinex; its launch passed only if Doorstop started it.
  local v
  for v in 5.4.2100 "$mx_bepinex"; do
    curl -fsSL -o "$ROOT/bep-$v.zip" "https://thunderstore.io/package/download/BepInEx/BepInExPack/$v/" || mx_fail loader.download "BepInExPack $v did not download"
  done
  mx_check loader.fresh "BepInExPack $mx_bepinex in the profile: marker $(cat "$loader_dir/.mortar-bepinex.json"), $(mx_doorstop_files "$mx_base" "$mx_bepinex")" \
    test "$(mx_doorstop_files "$mx_base" "$mx_bepinex")" = "the pack's doorstop_config.ini and .doorstop_version"

  # Every later profile is created now, so each install below reaches all of them.
  local pack edge crash quit upd upd2 dep p
  pack=$(mx_profile "Matrix Pack")
  edge=$(mx_profile "Matrix Edge")
  crash=$(mx_profile "Matrix Crash")
  quit=$(mx_profile "Matrix Quit")
  upd=$(mx_profile "Matrix Update")
  upd2=$(mx_profile "Matrix Update Two")
  dep=$(mx_profile "Matrix Deprecated")
  for p in "$pack" "$edge" "$crash" "$quit" "$upd" "$upd2" "$dep"; do
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

  # 2. Pin an older pack (Doorstop 3, no .doorstop_version) over the newer one, launch, then follow the latest again.
  local out
  if cli loader install lethal-company 5.4.2100 >"$ROOT/loader-2100.txt" 2>&1 && cli loader pin lethal-company 5.4.2100 >/dev/null; then
    if mx_launch "$mx_base" pin; then
      out=$(grep -m1 -o 'BepInEx 5\.[0-9.]* - Lethal Company' "$mx_log")
      mx_check loader.pin "pinned 5.4.2100; log opens with \"$out\"; $(mx_doorstop "$mx_base" 5.4.2100)" \
        test "$out|$(mx_doorstop "$mx_base" 5.4.2100)" = "BepInEx 5.4.21.0 - Lethal Company|$(mx_doorstop_ok)"
    fi
    mx_stop pin
  else
    mx_fail loader.pin "installing or pinning 5.4.2100 failed: $(head -c 300 "$ROOT/loader-2100.txt")"
  fi
  if cli loader pin lethal-company latest >/dev/null && cli loader install lethal-company "$mx_bepinex" >"$ROOT/loader-latest.txt" 2>&1; then
    if mx_launch "$mx_base" unpin; then
      out=$(grep -m1 -o 'BepInEx 5\.[0-9.]* - Lethal Company' "$mx_log")
      mx_check loader.unpin "unpinned and installed $mx_bepinex; log opens with \"$out\"; $(mx_doorstop "$mx_base" "$mx_bepinex")" \
        test "$out|$(mx_doorstop "$mx_base" "$mx_bepinex")" = "BepInEx 5.4.23.5 - Lethal Company|$(mx_doorstop_ok)"
    fi
    mx_stop unpin
  else
    mx_fail loader.unpin "unpin or install $mx_bepinex failed: $(head -c 300 "$ROOT/loader-latest.txt")"
  fi
  if cli loader install lethal-company "$mx_bepinex" >"$ROOT/loader-again.txt" 2>&1; then
    if mx_launch "$mx_base" reinstall; then
      mx_check loader.reinstall "reinstalled $mx_bepinex over itself; BepInEx loaded $(grep -c 'BepInEx\] Loading \[' "$mx_log") plugins; $(mx_doorstop "$mx_base" "$mx_bepinex")" \
        test "$(mx_doorstop "$mx_base" "$mx_bepinex")" = "$(mx_doorstop_ok)"
    fi
    mx_stop reinstall
  else
    mx_fail loader.reinstall "$(head -c 300 "$ROOT/loader-again.txt")"
  fi

  # 3. The modpack, through the queue as Browse adds it.
  local pkg open
  for pkg in "${MATRIX_MODPACK[@]}"; do
    cli queue add lethal-company "$pack" "${pkg%/*}" --source thunderstore --version "${pkg##*/}" >>"$ROOT/queue-add.txt" 2>&1 ||
      mx_fail modpack.queue "queue add $pkg failed"
  done
  open=$(mx_wait_queue "$pack")
  cli mods lethal-company "$pack" --json >"$ROOT/pack-mods.json"
  local npack
  npack=$(mx_json "$ROOT/pack-mods.json" 'len(d)')
  mx_check modpack.install "${#MATRIX_MODPACK[@]} packages queued; profile holds $npack entries (dependencies and the bridge included); open: ${open:-none}" \
    test -z "$open" -a "$npack" -gt "${#MATRIX_MODPACK[@]}"
  cli profile load-order lethal-company "$pack" --json >"$ROOT/pack-order.json"
  out=$(
    python3 - "$ROOT/pack-order.json" "$ROOT/pack-mods.json" <<'PY'
import json, sys
order = json.load(open(sys.argv[1]))
mods = [m for m in json.load(open(sys.argv[2])) if m["enabled"] and m["source"] != "mortar"]
pos = {r["id"]: r["position"] for r in order}
bad = [f'{r["id"]} before its dependent {dep}' for r in order for dep in r.get("dependents", []) if pos.get(dep, 0) < r["position"]]
missing = [m["id"] for m in mods if m["id"] not in pos]
print("; ".join(bad + [f"{m} missing from the load order" for m in missing]))
PY
  )
  mx_check modpack.loadorder "load order lists $(mx_json "$ROOT/pack-order.json" 'len(d)') packages, each after what it needs${out:+: $out}" test -z "$out"

  mx_launch "$pack" pack
  local planned loading skipped
  planned=$(grep -oE 'BepInEx\] [0-9]+ plugins? to load' "$mx_log" | grep -oE '[0-9]+' | head -1)
  loading=$(grep -cE 'BepInEx\] Loading \[' "$mx_log")
  skipped=$(grep -cE 'BepInEx\] (Skipping|Could not load) \[' "$mx_log")
  mx_check launch.count "BepInEx counted ${planned:-no} plugins to load, logged $loading Loading lines and skipped $skipped: $(grep -oE 'Skipping \[[^]]*\] because [^(]*' "$mx_log" | head -3 | tr '\n' ';')" \
    test "${planned:-x}" = "$((loading + skipped))"
  if go -C "$REPO" test ./internal/dotnet -run '^TestRealBepInExRun$' -count=1 -v -bepinex-profile "$(mx_dir "$pack")" >"$ROOT/guid-test.txt" 2>&1; then
    mx_pass mods.guids "$(grep -o '[0-9]* plugins found.*' "$ROOT/guid-test.txt")"
  else
    mx_fail mods.guids "$(grep -E 'realrun_test|FAIL' "$ROOT/guid-test.txt" | head -5 | tr '\n' ' ')"
  fi
  mx_check edge.flatten "MirageCore ships FSharp.Core/FSharp.Core.dll and Mirage loads it from beside its plugin: $(grep -c 'FSharp.Core.dll.*or one of its dependencies' "$mx_saves/Player.log") load failures" \
    test "$(grep -c 'FSharp.Core.dll.*or one of its dependencies' "$mx_saves/Player.log")" = 0
  mx_check edge.patchers "the modpack's preloader patchers loaded: $(grep -c 'Loaded 1 patcher method from' "$mx_log") patcher lines" \
    grep -q 'Loaded 1 patcher method from \[BepInEx.MonoMod.AutoHookGenPatcher' "$mx_log"
  mx_stop pack
  mx_problems "$pack" pack
  out=$(
    python3 - "$ROOT/problems-pack.json" "$ROOT/pack-mods.json" "$ROOT/LogOutput-pack.log" "$mx_saves/Player.log" <<'PY'
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
  mx_check problems.true "Problems lists $(echo "$out" | head -1); each checked against Thunderstore, the logs and the mod list$(echo "$out" | grep '^WRONG' | sed 's/^WRONG /; /' | tr -d '\n')" \
    test -z "$(echo "$out" | grep '^WRONG')" -a "${out:0:1}" = "{"

  # 4. Edge cases and Console, config and duplicate GUIDs, in one profile of probe packages.
  if ! probe_err=$(mx_install_probes "$edge" Base Beat Nested Own Caps Flat Deep Patcher DupA DupB); then
    mx_fail edge.install "$probe_err"
  else
    cli queue add lethal-company "$edge" FlipMods-BetterStamina --source thunderstore --version 1.5.7 >/dev/null
    mx_wait_queue "$edge" >/dev/null
    mx_problems "$edge" edge
    mx_check mods.dupguid "Problems flags $(mx_json "$ROOT/problems-edge.json" '[c["guid"] for c in d.get("pluginClashes") or []]') with versions $(mx_json "$ROOT/problems-edge.json" '[x["version"] for c in d.get("pluginClashes") or [] for x in c["copies"]]')" \
      test "$(mx_json "$ROOT/problems-edge.json" '[(c["guid"], sorted(x["version"] for x in c["copies"])) for c in d.get("pluginClashes") or []]')" = "[('tech.rethunk.mortar.matrix.dup', ['1.0.0', '1.0.0'])]"
    if mx_launch "$edge" edge; then
      local lines1 lines2
      sleep 4
      lines1=$(mx_wails launchsvc.Service.Lines '"lethal-company"' "$(mx_q "$edge")")
      sleep 4
      lines2=$(mx_wails launchsvc.Service.Lines '"lethal-company"' "$(mx_q "$edge")")
      out=$(
        python3 - "$lines1" "$lines2" <<'PY'
import json, sys
a, b = json.loads(sys.argv[1]), json.loads(sys.argv[2])
beats = lambda es: [e for e in es if e["message"].startswith("matrix heartbeat")]
levels = {e["level"] for e in b}
sources = {e["mod"] for e in b}
ok = (beats(a) and len(beats(b)) > len(beats(a)) and max(e["seq"] for e in b) > max(e["seq"] for e in a)
      and all(e["level"] == "WARN" and e["mod"] == "Matrix beat" for e in beats(b))
      and any(e["level"] == "ERROR" and e["message"] == "matrix error line" for e in b) and "BepInEx" in sources)
print(("OK " if ok else "BAD ") + f"heartbeats {len(beats(a))} then {len(beats(b))}, levels {sorted(levels)}, {len(sources)} sources")
PY
      )
      mx_check launch.console "Console lines read twice during the run: ${out#* }" test "${out%% *}" = OK
      local want
      for want in "base cfg=shipped:edge.config-placement" "own cfg=own-layout:edge.own-layout" "caps cfg=caps:edge.case" "flat beside=True:edge.flatten-probe"; do
        mx_check "${want#*:}" "LogOutput.log: $(grep -m1 -o "matrix ${want%%:*}" "$mx_log" || echo "no \"matrix ${want%%:*}\"")" grep -q "matrix ${want%%:*}" "$mx_log"
      done
      mx_check edge.nested "plugins/Deep/Er/Nested.dll: $(grep -m1 -o 'Loading \[Matrix nested[^]]*\]' "$mx_log" || echo not loaded)" grep -q 'Loading \[Matrix nested' "$mx_log"
      mx_check edge.longpath "a plugin $(find "$(mx_dir "$edge")/BepInEx/plugins" -name Deep.dll | awk '{print length("Z:" $0)}') characters deep as Wine sees it: $(grep -m1 -o 'Loading \[Matrix deep[^]]*\]' "$mx_log" || echo not loaded)" grep -q 'Loading \[Matrix deep' "$mx_log"
      mx_check edge.patcher "patchers/MatrixPatcher.dll: $(grep -m1 -o 'matrix patcher initialized' "$mx_log" || echo not initialized)" grep -q 'matrix patcher initialized' "$mx_log"
      mx_check mods.dupguid-runtime "BepInEx loaded the duplicated GUID $(grep -c 'Loading \[Matrix dup' "$mx_log") time(s)" test "$(grep -c 'Loading \[Matrix dup' "$mx_log")" = 1
      mx_stop edge
      # The typed editor writes a string and a float; BepInEx rewrites every .cfg it reads at startup, so a relaunch
      # that keeps both values proves the edit is in BepInEx's own format.
      local cfgdir
      cfgdir="$(mx_dir "$edge")/BepInEx/config"
      if mx_wails configsvc.Service.Set '"lethal-company"' "$(mx_q "$edge")" '"thunderstore:MortarMatrix-ProbeBase"' '"tech.rethunk.mortar.matrix.base.cfg"' '"General"' '"Value"' '"edited by mortar"' >/dev/null &&
        mx_wails configsvc.Service.Set '"lethal-company"' "$(mx_q "$edge")" '"thunderstore:FlipMods-BetterStamina"' '"FlipMods.BetterStamina.cfg"' '"CarryWeight"' '"CarryWeightPenaltyMultiplier"' '"0.75"' >/dev/null; then
        mx_edge_edited=1
        mx_launch "$edge" edge-relaunch
        mx_stop edge-relaunch
        mx_check mods.config "after Set and a relaunch: probe logged \"$(grep -m1 -o 'matrix base cfg=[^\r]*' "$mx_log")\", BetterStamina.cfg has \"$(grep -m1 'CarryWeightPenaltyMultiplier =' "$cfgdir/FlipMods.BetterStamina.cfg")\"" \
          grep -q 'matrix base cfg=edited by mortar' "$mx_log"
        grep -q '^CarryWeightPenaltyMultiplier = 0.75' "$cfgdir/FlipMods.BetterStamina.cfg" || mx_fail mods.config-float "BetterStamina.cfg lost 0.75 across the relaunch"
      else
        mx_fail mods.config "configsvc Set failed"
      fi
    fi
  fi

  # 5. An induced plugin exception, then a natural exit.
  if mx_install_probes "$crash" Throw >/dev/null && mx_launch "$crash" crash; then
    sleep 3
    mx_stop crash
    mx_problems "$crash" crash
    cli runs lethal-company "$crash" --json >"$ROOT/runs-crash.json"
    out=$(mx_json "$ROOT/problems-crash.json" '[(f["name"], f["kind"], f["message"][:60]) for f in d.get("loadFailures") or []]')
    mx_check launch.crash "Problems: $out; run: loader $(mx_json "$ROOT/runs-crash.json" 'd[0]["loaderVersion"]'), $(mx_json "$ROOT/runs-crash.json" 'd[0]["errors"]') errors, outcome $(mx_json "$ROOT/runs-crash.json" 'd[0]["outcome"]')" \
      python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); r=json.load(open(sys.argv[2]))[0]; sys.exit(not (any(f["name"]=="ProbeThrow" and "matrix probe failed in Awake" in f["message"] for f in d.get("loadFailures") or []) and r["loaderVersion"] and r["errors"]>0))' "$ROOT/problems-crash.json" "$ROOT/runs-crash.json"
  fi
  rm -f "$mx_saves/LCSaveFile1"
  if mx_install_probes "$quit" Save Quit >/dev/null && mx_launch "$quit" quit; then
    if mx_idle 90; then
      cli runs lethal-company "$quit" --json >"$ROOT/runs-quit.json"
      mx_check launch.exit "the game quit by itself; Mortar went idle and recorded outcome $(mx_json "$ROOT/runs-quit.json" 'd[0]["outcome"]'); game folder: $(mx_purged quit || true)" \
        test -z "$(mx_purged quit)" -a "$(mx_json "$ROOT/runs-quit.json" 'd[0]["outcome"]')" = ran
    else
      mx_fail launch.exit "Mortar still reports $(mx_state) 90s after the game quit"
      mx_stop quit
    fi
    reap_prefix "$mx_compat" >/dev/null
  fi

  # 6. Updates with the save the game just wrote.
  mx_check saves.fixture "the game wrote LCSaveFile1 ($(stat -c %s "$mx_saves/LCSaveFile1" 2>/dev/null || echo 0) bytes); Saves lists $(cli saves lethal-company "$upd" --json | python3 -c 'import json,sys; print([s["folder"] for s in json.load(sys.stdin)])')" \
    test -s "$mx_saves/LCSaveFile1"
  local save_sum
  save_sum=$(sha256sum "$mx_saves/LCSaveFile1" 2>/dev/null | cut -d' ' -f1)
  local x
  for x in AinaVT-LethalConfig:1.4.5 FlipMods-BetterStamina:1.5.6 Rune580-LethalCompany_InputUtils:0.7.12; do
    cli queue add lethal-company "$upd" "${x%:*}" --source thunderstore --version "${x#*:}" >/dev/null
  done
  cli queue add lethal-company "$upd2" FlipMods-BetterStamina --source thunderstore --version 1.5.6 >/dev/null
  mx_wait_queue "$upd" >/dev/null
  mx_wait_queue "$upd2" >/dev/null
  cli updates lethal-company "$upd" --json >"$ROOT/updates.json"
  local backups_before
  backups_before=$(cli backups list --game lethal-company --json | python3 -c 'import json,sys; print(len(json.load(sys.stdin) or []))')
  cli update lethal-company "$upd" --all >"$ROOT/update-all.txt" 2>&1
  mx_wait_queue "$upd" >/dev/null
  cli mods lethal-company "$upd" --json >"$ROOT/upd-mods.json"
  mx_check updates.all "updates listed $(mx_json "$ROOT/updates.json" '[u["package"] + " " + u["installed"] + "->" + u["version"] for u in d["updates"]]'); after Update all: $(mx_json "$ROOT/upd-mods.json" '[m["name"] + " " + m["version"] for m in d if m["source"] == "thunderstore"]')" \
    test "$(mx_json "$ROOT/upd-mods.json" 'sorted(m["version"] for m in d if m["source"] == "thunderstore")')" = "['0.7.13', '1.4.6', '1.5.7']"
  cli backups list --game lethal-company --json >"$ROOT/backups.json"
  mx_check updates.backup "backups before $backups_before, after $(mx_json "$ROOT/backups.json" 'len(d)'); newest is $(mx_json "$ROOT/backups.json" '(d[0]["kind"], [s["folder"] for s in d[0]["saves"]])')" \
    test "$(mx_json "$ROOT/backups.json" '(d[0]["kind"], [s["folder"] for s in d[0]["saves"]])')" = "('update', ['LCSaveFile1'])"
  local undo
  undo=$(sed -n 's/^Undo all: mortar profile revert lethal-company "[^"]*" \(.*\)$/\1/p' "$ROOT/update-all.txt")
  if [ -n "$undo" ] && cli profile revert lethal-company "$upd" "$undo" >/dev/null 2>&1; then
    cli mods lethal-company "$upd" --json >"$ROOT/upd-reverted.json"
    mx_check updates.rollback "Undo all ($undo) put back $(mx_json "$ROOT/upd-reverted.json" '[m["name"] + " " + m["version"] for m in d if m["source"] == "thunderstore"]')" \
      test "$(mx_json "$ROOT/upd-reverted.json" 'sorted(m["version"] for m in d if m["source"] == "thunderstore")')" = "['0.7.12', '1.4.5', '1.5.6']"
  else
    mx_fail updates.rollback "no undo id in: $(head -c 300 "$ROOT/update-all.txt")"
  fi
  cli updates apply --everywhere lethal-company thunderstore:FlipMods-BetterStamina >"$ROOT/update-everywhere.txt" 2>&1
  mx_wait_queue "$upd2" >/dev/null
  mx_check updates.everywhere "$(tr '\n' ';' <"$ROOT/update-everywhere.txt" | tr -s ' ')" \
    test "$(cli mods lethal-company "$upd2" --json | python3 -c 'import json,sys; print([m["version"] for m in json.load(sys.stdin) if m["name"] == "BetterStamina"])')" = "['1.5.7']" -a \
    "$(cli mods lethal-company "$upd" --json | python3 -c 'import json,sys; print([m["version"] for m in json.load(sys.stdin) if m["name"] == "BetterStamina"])')" = "['1.5.7']"
  head -c 64 /dev/urandom >"$mx_saves/LCSaveFile1"
  local newest
  newest=$(mx_json "$ROOT/backups.json" 'd[0]["name"]')
  cli backups restore "$newest" LCSaveFile1 >"$ROOT/backup-restore.txt" 2>&1
  mx_check updates.restore "restoring $newest gives LCSaveFile1 back byte for byte" test "$(sha256sum "$mx_saves/LCSaveFile1" | cut -d' ' -f1)" = "$save_sum"

  # 7. Deprecated packages and their named replacements.
  cli queue add lethal-company "$dep" VirusTLNR-MaskedFixes --source thunderstore --version 0.0.3 >/dev/null
  mx_wait_queue "$dep" >/dev/null
  mx_problems "$dep" dep
  out=$(mx_json "$ROOT/problems-dep.json" '{x["name"]: x.get("replacement", "") for x in d.get("deprecated") or []}')
  mx_check mods.deprecated "Problems: $out" test "$(mx_json "$ROOT/problems-dep.json" '{x["name"]: x.get("replacement", "") for x in d.get("deprecated") or []}.get("VirusTLNR-MaskedFixes")')" = VirusTLNR-MaskedInvisFix

  # 8. Sharing.
  regress_bepinex_sharing "$edge" "$pack"

  set -e -o pipefail
  echo "---- BepInEx matrix: $(grep -c "$(printf '\tPASS\t')" "$ROOT/matrix.tsv") PASS, $(grep -c "$(printf '\tFAIL\t')" "$ROOT/matrix.tsv") FAIL ($ROOT/matrix.tsv)"
}

# regress_bepinex_sharing EDGE PACK shares a probe profile (local packages only) and the modpack (Thunderstore only).
regress_bepinex_sharing() {
  local edge=$1 pack=$2 out
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
  regress_bepinex_lan "$edge"
}

# regress_bepinex_lan EDGE starts a second server in $ROOT/peer with its own home, pairs it with this one and sends
# the probe profile, whose packages exist only as local files.
regress_bepinex_lan() {
  local edge=$1 peer=$ROOT/peer peer_port pa pb code
  # The send carries the config the edge step edited; without that edit there is nothing to look for at the receiver.
  if [ -z "${mx_edge_edited:-}" ]; then
    mx_fail share.lan "not run: the probe profile holds no edited config (edge.install or mods.config failed first)"
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
  got=$(peercli mods lethal-company "Matrix Edge" --json 2>/dev/null | python3 -c 'import json,sys; print(sorted(m["name"] for m in json.load(sys.stdin) if m["source"] == "local"))' 2>/dev/null)
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
