import type {
  StartupMod,
  StartupPhases,
  StartupReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

type PhaseId = 'smapi' | 'entry' | 'content' | 'firstTicks' | 'intro'

interface PhaseSegment {
  id: PhaseId
  ms: number
}

/** Mods below this share fold into one row; a 190-mod list is mostly noise under it. */
const FOLD_BELOW_MS = 50
/** A launch-to-launch change smaller than this is run-to-run noise (measured spread on one profile: about 0.4 s). */
const REGRESSION_MIN_MS = 1000
const MS_PER_SECOND = 1000

/** A mod at or above this share of startup earns a hint in Problems; measured on a 193-mod profile, two mods do. */
const SLOW_MOD_MS = 3000
/** Content packs are smaller units; the slowest on the same profile took 0.9 s. */
const SLOW_PACK_MS = 1000

type SlowStartup =
  | { kind: 'mod'; id: string; name: string; ms: number }
  | { kind: 'pack'; id: string; name: string; ms: number; framework: string }

/** Mods and content packs slow enough to mention. A framework that loads content packs (Content Patcher) is never a
 * row itself: so many mods need it that it cannot be avoided, so only its slow packs are named. */
function slowStartups(report: StartupReport): SlowStartup[] {
  const out: SlowStartup[] = []
  for (const mod of report.mods ?? []) {
    const packs = mod.packs ?? []
    const total = modTotal(mod)
    if (packs.length === 0 && total >= SLOW_MOD_MS) {
      out.push({ kind: 'mod', id: mod.id, name: mod.name, ms: total })
    }
    for (const pack of packs) {
      if (pack.ms >= SLOW_PACK_MS) {
        out.push({
          kind: 'pack',
          id: pack.id,
          name: pack.name,
          ms: pack.ms,
          framework: mod.name,
        })
      }
    }
  }
  return out.sort((a, b) => b.ms - a.ms)
}

interface StartupRegression {
  name: string
  version: string
  addedMs: number
}

function formatDuration(ms: number, locale: string): string {
  return ms >= MS_PER_SECOND
    ? `${(ms / MS_PER_SECOND).toLocaleString(locale, { maximumFractionDigits: 1 })} s`
    : `${ms.toLocaleString(locale)} ms`
}

/** Mods that were updated or added since the previous launch and made startup slower by a noticeable amount. */
function startupRegressions(
  latest: StartupReport,
  previous: StartupReport | undefined,
): StartupRegression[] {
  if (!previous) {
    return []
  }
  const before = new Map((previous.mods ?? []).map((m) => [m.id, m]))
  const out: StartupRegression[] = []
  for (const mod of latest.mods ?? []) {
    const old = before.get(mod.id)
    const changed = !old || old.version !== mod.version
    const added = modTotal(mod) - (old ? modTotal(old) : 0)
    if (changed && added >= REGRESSION_MIN_MS) {
      out.push({ name: mod.name, version: mod.version, addedMs: added })
    }
  }
  return out.sort((a, b) => b.addedMs - a.addedMs)
}

/** SMAPI's own mod loading is worth a finding past this time and share of the start; the one measured run (225 mods)
 * spent 10.8 s, 15%, so both sit well below it while staying clear of run-to-run noise. */
const SMAPI_FINDING_MS = 5000
const SMAPI_FINDING_SHARE = 0.1
const MAJORITY = 0.5
const MAX_FINDINGS = 3

type Finding =
  | { kind: 'heavy'; mod: StartupMod; ms: number; titleMs: number; packs: number }
  | { kind: 'slower'; ms: number }
  | { kind: 'smapi'; ms: number }

/** Content packs, slowest first, that together make up most of a framework's pack time. */
function packsBehind(mod: StartupMod): number {
  const packs = [...(mod.packs ?? [])].sort((a, b) => b.ms - a.ms)
  const sum = packs.reduce((n, p) => n + p.ms, 0)
  let running = 0
  let count = 0
  for (const p of packs) {
    if (running >= sum * MAJORITY) {
      break
    }
    running += p.ms
    count += 1
  }
  return count
}

/** What the selected start says to do, from the data the page already has: the mod that dominates it, a slowdown
 * since the launch before, and SMAPI's own loading when it is large. */
function startupFindings(report: StartupReport, previous: StartupReport | undefined): Finding[] {
  const out: Finding[] = []
  const titleMs = report.phases.titleScreen
  const [heaviest] = [...(report.mods ?? [])].sort((a, b) => modTotal(b) - modTotal(a))
  if (heaviest && modTotal(heaviest) >= SLOW_MOD_MS) {
    out.push({
      kind: 'heavy',
      mod: heaviest,
      ms: modTotal(heaviest),
      titleMs,
      packs: packsBehind(heaviest),
    })
  }
  const before = previous?.phases.titleScreen ?? 0
  if (titleMs > 0 && before > 0 && titleMs - before >= REGRESSION_MIN_MS) {
    out.push({ kind: 'slower', ms: titleMs - before })
  }
  const smapi = phaseSegments(report.phases).find((s) => s.id === 'smapi')?.ms ?? 0
  if (smapi >= SMAPI_FINDING_MS && smapi >= titleMs * SMAPI_FINDING_SHARE) {
    out.push({ kind: 'smapi', ms: smapi })
  }
  return out.slice(0, MAX_FINDINGS)
}

type WhyKind = 'event' | 'assets' | 'entry' | 'sampled'

/** The one thing that costs a mod the most, from its own timings: its slowest event, assets and packs, Entry, or the
 * sampled time inside its patches. */
function whyOf(mod: StartupMod): { kind: WhyKind; ms: number; event?: string } | null {
  const event = slowestEvent(mod)
  const candidates: { kind: WhyKind; ms: number; event?: string }[] = [
    { kind: 'event', ms: event?.[1] ?? 0, ...(event ? { event: event[0] } : {}) },
    { kind: 'assets', ms: mod.assetMs + mod.loadMs },
    { kind: 'entry', ms: mod.entryMs },
    { kind: 'sampled', ms: mod.sampleMs },
  ]
  const best = candidates.reduce((a, b) => (b.ms > a.ms ? b : a))
  return best.ms > 0 ? best : null
}

const rowAnchor = (id: string) => `startup-mod-${id}`

function modTotal(mod: StartupMod): number {
  const events = Object.values(mod.eventMs ?? {}).reduce((n: number, ms) => n + (ms ?? 0), 0)
  return mod.entryMs + mod.assetMs + mod.loadMs + events
}

function slowestEvent(mod: StartupMod): [string, number] | null {
  let best: [string, number] | null = null
  for (const [name, value] of Object.entries(mod.eventMs ?? {})) {
    const ms = value ?? 0
    if (best === null || ms > best[1]) {
      best = [name, ms]
    }
  }
  return best
}

/** Consecutive phases from process start to the title screen; a phase the bridge did not see joins the next one. */
function phaseSegments(p: StartupPhases): PhaseSegment[] {
  const marks: [PhaseId, number][] = [
    ['smapi', p.bridgeEntry],
    ['entry', p.entryDone],
    ['content', p.gameLaunched],
    ['firstTicks', p.titleMenu],
    ['intro', p.titleScreen],
  ]
  const out: PhaseSegment[] = []
  let at = 0
  for (const [id, end] of marks) {
    if (end > at) {
      out.push({ id, ms: end - at })
      at = end
    }
  }
  return out
}

function foldMods(mods: StartupMod[] | null): {
  shown: StartupMod[]
  folded: { count: number; ms: number }
} {
  const shown: StartupMod[] = []
  const folded = { count: 0, ms: 0 }
  // A mod whose cost is mostly in its patches on game code shows only in the sampled time.
  const weight = (mod: StartupMod): number => Math.max(modTotal(mod), mod.sampleMs)
  for (const mod of mods ?? []) {
    const total = modTotal(mod)
    if (weight(mod) >= FOLD_BELOW_MS) {
      shown.push(mod)
    } else {
      folded.count += 1
      folded.ms += total
    }
  }
  shown.sort((a, b) => weight(b) - weight(a))
  return { shown, folded }
}

export {
  type Finding,
  foldMods,
  formatDuration,
  modTotal,
  type PhaseId,
  phaseSegments,
  rowAnchor,
  type SlowStartup,
  type StartupRegression,
  slowestEvent,
  slowStartups,
  startupFindings,
  startupRegressions,
  whyOf,
}

/** The report the picker shows: the chosen one, else the newest. */
export function pickReport(reports: StartupReport[], selected: string): StartupReport | undefined {
  return reports.find((r) => r.id === selected) ?? reports[0]
}
