import type {
  StartupMod,
  StartupPhases,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'

type PhaseId = 'smapi' | 'entry' | 'content' | 'firstTicks' | 'intro'

interface PhaseSegment {
  id: PhaseId
  ms: number
}

/** Mods below this share fold into one row; a 190-mod list is mostly noise under it. */
const FOLD_BELOW_MS = 50

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

export { foldMods, modTotal, type PhaseId, type PhaseSegment, phaseSegments, slowestEvent }
