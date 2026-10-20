// The website demo's Lethal Company profiles, seeded into the recorder's sandbox through the commands an agent uses.
import { execFileSync } from 'node:child_process'
import { chmodSync, mkdirSync, writeFileSync } from 'node:fs'
import { serverEnv } from './sandbox.ts'

const GAME = 'lethal-company'
export const CREW = 'Friday Crew'
export const SOLO = 'Solo Runs'
const TEMPLATE = 'Crew Basics'
// Real Thunderstore packages, downloaded from Thunderstore by the sandbox as a player's would be.
const CREW_MODS = [
  'notnotnotswipez-MoreCompany',
  'anormaltwig-LateCompany',
  'tinyhoot-ShipLoot',
  'x753-More_Suits',
  'Verity-TooManySuits',
  'FlipMods-ReservedFlashlightSlot',
  'FlipMods-HotbarPlus',
]
const SOLO_MODS = ['tinyhoot-ShipLoot', 'FlipMods-HotbarPlus', 'x753-More_Suits']
// Switched off, so the flashlight slot that needs it is the profile's problem.
const OFF = 'thunderstore:FlipMods-ReservedItemSlotCore'
const QUEUE_WAIT_MS = 120_000
const POLL_MS = 1000

/** Each package's plugin as BepInEx names it when it loads, by the package that holds it. */
const PLUGINS: Record<string, { guid: string; name: string }> = {
  'tinyhoot-ShipLoot': { guid: 'com.github.tinyhoot.ShipLoot', name: 'ShipLoot' },
  'FlipMods-HotbarPlus': { guid: 'FlipMods.HotbarPlus', name: 'HotbarPlus' },
  'FlipMods-ReservedFlashlightSlot': {
    guid: 'FlipMods.ReservedFlashlightSlot',
    name: 'ReservedFlashlightSlot',
  },
  'notnotnotswipez-MoreCompany': { guid: 'me.swipez.melonloader.morecompany', name: 'MoreCompany' },
  'x753-More_Suits': { guid: 'x753.More_Suits', name: 'More Suits' },
  'Verity-TooManySuits': { guid: 'TooManySuits', name: 'TooManySuits' },
  'anormaltwig-LateCompany': { guid: 'twig.latecompany', name: 'Late Company' },
}

let env: Record<string, string> = {}
const dir = () => process.env.MORTAR_E2E_DIR ?? ''
const cli = (...args: string[]) =>
  execFileSync(`${dir()}/mortar-server`, args, { env, encoding: 'utf8' }).trim()

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null

interface Listed {
  id: string
  version: string
}

function modsOf(profile: string): Listed[] {
  const listed: unknown = JSON.parse(cli('mods', GAME, profile, '--json'))
  return Array.isArray(listed)
    ? listed.filter(isRecord).map((m) => ({ id: String(m.id), version: String(m.version) }))
    : []
}

async function queued(profile: string, mods: string[]) {
  for (const id of mods) {
    cli('queue', 'add', GAME, profile, id, '--source', 'thunderstore')
  }
  const end = Date.now() + QUEUE_WAIT_MS
  while (Date.now() < end) {
    const items: unknown = JSON.parse(cli('queue', '--json'))
    const states = isRecord(items) && Array.isArray(items.items) ? items.items.filter(isRecord) : []
    if (states.every((i) => i.state === 'done' || i.state === 'failed')) {
      return
    }
    await Bun.sleep(POLL_MS)
  }
  throw new Error(`the downloads for ${profile} did not finish`)
}

function profileId(name: string): string {
  const listed: unknown = JSON.parse(cli('profiles', GAME, '--json'))
  const found = Array.isArray(listed)
    ? listed.filter(isRecord).find((p) => p.name === name)
    : undefined
  return String(found?.id ?? '')
}

/** The demo's profiles; answers the id of the one the demo opens on. */
export async function seedDemo(): Promise<string> {
  env = serverEnv(dir())
  dropFixtures()
  cli('profile', 'create', GAME, CREW)
  const crew = profileId(CREW)
  await queued(CREW, CREW_MODS)
  switchOff()
  cli('templates', 'save', GAME, CREW, TEMPLATE)
  cli('profile', 'create', GAME, SOLO)
  await queued(SOLO, SOLO_MODS)
  // Play goes straight to the stand-in rather than through Steam, which the sandbox refuses.
  cli('settings', 'set', '--game', GAME, 'defaultLaunchMethod', 'direct')
  cli('profile', 'set', GAME, CREW, 'launchPrefix', writeStandIn(crew))
  return crew
}

// The e2e seed's own profiles would read as test data beside the demo's.
function dropFixtures() {
  for (const game of [GAME, 'stardew']) {
    const listed: unknown = JSON.parse(cli('profiles', game, '--json'))
    for (const p of Array.isArray(listed) ? listed.filter(isRecord) : []) {
      cli('profile', 'delete', game, String(p.id))
    }
    cli('trash', 'empty', '--yes', '--game', game)
  }
}

/** Switches the demo profile's dependency off again, as the demo opens with it. */
export function switchOff() {
  cli('mods', 'disable', GAME, CREW, OFF)
}

/** A profile to stand in for one a visitor creates; answers its id. */
export function seedBlank(name: string): string {
  cli('profile', 'create', GAME, name)
  return profileId(name)
}

export function dropBlank(id: string) {
  cli('profile', 'delete', GAME, id)
  cli('trash', 'empty', '--yes', '--game', GAME)
}

export const standInPath = () => `${dir()}/run-demo.sh`

const pad = (level: string) => `${level}:`.padEnd(8)
const line = (level: string, source: string, text: string) =>
  `[${pad(level)}${source.padStart(10)}] ${text}`

interface Plugin {
  guid: string
  name: string
  version: string
}

function plugins(): Plugin[] {
  const versions = new Map(modsOf(CREW).map((m) => [m.id.replace('thunderstore:', ''), m.version]))
  return CREW_MODS.flatMap((pkg) => {
    const p = PLUGINS[pkg]
    return p ? [{ ...p, version: versions.get(pkg) ?? '1.0.0' }] : []
  })
}

// LogOutput.log as BepInEx 5 writes it for this profile, each line with the seconds to wait after it.
function logLines(list: Plugin[]): [number, string][] {
  const b = (level: string, text: string, wait = 0.15): [number, string] => [
    wait,
    line(level, 'BepInEx', text),
  ]
  const loading = list.flatMap((p): [number, string][] =>
    p.guid === 'FlipMods.ReservedFlashlightSlot'
      ? [
          b(
            'Error',
            `Could not load [${p.name} ${p.version}] because it has missing dependencies: FlipMods.ReservedItemSlotCore`,
            0.3,
          ),
        ]
      : [
          b('Info', `Loading [${p.name} ${p.version}]`, 0.4),
          [0.5, line('Info', p.name, `Plugin ${p.guid} is loaded!`)],
        ],
  )
  return [
    b('Message', 'BepInEx 5.4.23.2 - Lethal Company'),
    b('Info', 'Running under Unity v2022.3.62.12290227'),
    b('Info', 'CLR runtime version: 4.0.30319.42000'),
    b('Info', 'Supports SRE: True'),
    b('Info', 'System platform: Bits64, Windows'),
    b('Message', 'Preloader started'),
    b('Info', 'Loaded 1 patcher method from [BepInEx.Preloader 5.4.23.2]'),
    b('Info', '1 patcher plugin loaded'),
    b('Info', 'Patching [UnityEngine.CoreModule] with [BepInEx.Chainloader]'),
    b('Message', 'Preloader finished', 1.2),
    b('Message', 'Chainloader ready'),
    b('Message', 'Chainloader started'),
    b('Info', `${list.length} plugins to load`, 0.4),
    ...loading,
    [
      0.6,
      line(
        'Warning',
        'HarmonyX',
        'AccessTools.DeclaredMethod: Could not find method for type GameNetcodeStuff.PlayerControllerB and name __rpc_handler_2013428264 and parameters ',
      ),
    ],
    b('Message', 'Chainloader startup complete', 1.5),
    [
      1.5,
      line('Warning', 'Unity Log', 'The referenced script (Unknown) on this Behaviour is missing!'),
    ],
    [2, line('Info', 'Unity Log', 'Initializing Steamworks')],
    [3, line('Info', 'Unity Log', 'Lobby list refreshed: 14 lobbies')],
    [4, line('Info', 'Unity Log', 'Loaded save file LCSaveFile1')],
  ]
}

// The bridge's startup report for the run (internal/launchsvc/startup.go StartupReport), timed per plugin.
function startupReport(list: Plugin[], start: string) {
  const loaded = list.filter((p) => p.guid !== 'FlipMods.ReservedFlashlightSlot')
  return {
    schema: 1,
    loader: '5.4.23.2',
    processStart: start,
    phases: {
      bridgeEntry: 2100,
      entryDone: 6400,
      gameLaunched: 7300,
      titleMenu: 9800,
      titleScreen: 11_900,
    },
    entryTimed: true,
    mods: loaded.map((p, i) => ({
      id: p.guid,
      name: p.name,
      version: p.version,
      entryMs: [640, 85, 1240, 410, 320, 150][i] ?? 100,
      patchMs: [210, 30, 380, 60, 45, 70][i] ?? 20,
      sampleMs: [600, 80, 1180, 390, 300, 140][i] ?? 90,
    })),
    otherMs: 8700,
    sampledOtherMs: 5100,
  }
}

/** The launch prefix that stands in for Proton: writes LogOutput.log and the startup report, then runs as the game. */
function writeStandIn(crew: string): string {
  const profile = `${env.HOME}/.local/share/mortar/profiles/${GAME}/${crew}`
  const list = plugins()
  const lines = logLines(list)
  const script = standInPath()
  writeFileSync(`${dir()}/run-demo.lines`, lines.map(([wait, l]) => `${wait}|${l}`).join('\n'))
  writeFileSync(`${dir()}/run-demo.report`, JSON.stringify(startupReport(list, '__START__')))
  mkdirSync(`${profile}/startup`, { recursive: true })
  writeFileSync(
    script,
    `#!/bin/bash
log='${profile}/BepInEx/LogOutput.log'
: >"$log"
start=$(date -u +%Y-%m-%dT%H:%M:%S.000Z)
game=$$
(
  while IFS= read -r l; do
    kill -0 "$game" 2>/dev/null || exit
    printf '%s\\r\\n' "\${l#*|}" >>"$log"
    sleep "\${l%%|*}"
    case "$l" in *'Chainloader startup complete'*)
      sed "s/__START__/$start/" '${dir()}/run-demo.report' >'${profile}/startup/'"$(date -u +%Y%m%dT%H%M%SZ)"'.json' ;;
    esac
  done <'${dir()}/run-demo.lines'
) &
exec -a "$1" sleep 600
`,
  )
  chmodSync(script, 0o755)
  return script
}
