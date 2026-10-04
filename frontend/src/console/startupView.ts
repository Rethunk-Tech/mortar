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
  for (const mod of mods ?? []) {
    const total = modTotal(mod)
    if (total >= FOLD_BELOW_MS) {
      shown.push(mod)
    } else {
      folded.count += 1
      folded.ms += total
    }
  }
  shown.sort((a, b) => modTotal(b) - modTotal(a))
  return { shown, folded }
}

export {
  foldMods,
  formatDuration,
  modTotal,
  type PhaseId,
  type PhaseSegment,
  phaseSegments,
  type StartupRegression,
  slowestEvent,
  startupRegressions,
}
