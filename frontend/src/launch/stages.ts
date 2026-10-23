import type { Progress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'

/** A mark on the launch screen: the launcher the game went through, the game, and its mod loader. */
export type StageId = 'via' | 'game' | 'loader'

type StageState = 'pending' | 'active' | 'done' | 'failed'

export interface Stage {
  id: StageId
  state: StageState
}

/** stageIds are the marks a launch shows: the launcher only when one started the game, the loader only with mods. */
function stageIds(p: Progress): StageId[] {
  return [...(p.via ? (['via'] as const) : []), 'game', ...(p.loader ? (['loader'] as const) : [])]
}

function reached(p: Progress, id: StageId): boolean {
  if (id === 'via') {
    return p.handed || p.up > 0 || p.loaderStarted
  }
  if (id === 'game') {
    return p.up >= p.copies || p.loaderStarted
  }
  // A loader whose log never says it is done is done once its log has started.
  return p.ready || (p.loaderStarted && !p.readies)
}

/**
 * launchStages are the marks of a launch: those it has passed are done, the first it has not is active (or failed,
 * when the launch failed), and the rest wait.
 */
export function launchStages(p: Progress, failed: boolean): Stage[] {
  let current = false
  return stageIds(p).map((id) => {
    if (!current && reached(p, id)) {
      return { id, state: 'done' }
    }
    if (current) {
      return { id, state: 'pending' }
    }
    current = true
    return { id, state: failed ? 'failed' : 'active' }
  })
}

/** waitsForLoader reports whether the launch screen stays up after the game is running, until the loader is done. */
export function waitsForLoader(p: Progress): boolean {
  return p.loader !== '' && p.readies && !p.ready
}
