import type {
  StartupMod,
  StartupPhases,
  StartupReport,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'

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
  | { kind: 'mod'; uniqueId: string; name: string; ms: number }
  | { kind: 'pack'; uniqueId: string; name: string; ms: number; framework: string }
  | { kind: 'packs'; uniqueId: string; name: string; ms: number; count: number }

/** Mods and content packs slow enough to mention. A framework is named for its packs' total, not offered for switching off. */
function slowStartups(report: StartupReport): SlowStartup[] {
  const out: SlowStartup[] = []
  for (const mod of report.mods ?? []) {
    const packs = mod.packs ?? []
    const total = modTotal(mod)
    if (packs.length > 0 && total >= SLOW_MOD_MS) {
      out.push({ kind: 'packs', uniqueId: mod.id, name: mod.name, ms: total, count: packs.length })
    } else if (packs.length === 0 && total >= SLOW_MOD_MS) {
      out.push({ kind: 'mod', uniqueId: mod.id, name: mod.name, ms: total })
    }
    for (const pack of packs) {
      if (pack.ms >= SLOW_PACK_MS) {
        out.push({
          kind: 'pack',
          uniqueId: pack.id,
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
  foldMods,
  formatDuration,
  modTotal,
  type PhaseId,
  type PhaseSegment,
  phaseSegments,
  type SlowStartup,
  type StartupRegression,
  slowestEvent,
  slowStartups,
  startupRegressions,
}
