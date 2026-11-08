import { msg } from '@lingui/core/macro'
import { coalescer, keepUnchanged, sameValue } from '@rethunk/react-load-hooks'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import {
  Last,
  Pages,
  ProblemsNoWait,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { Mods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SetListGroupBy } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { getLast, peekLast, putLast } from '../shell/lastGood.ts'
import { type InlineError, inlineError, reportError, reportUnexpected } from '../toasts/report.ts'
import { useBadges } from './badges.ts'
import { loadCollapsed, persistCollapsed } from './group.ts'
import { missingCount, modId, problemCount } from './lookup.ts'
import { openTarget } from './storeView.ts'
import { useUpdates } from './updates.ts'

// Loads overlap when installs finish back to back; only the newest one may write, or an older snapshot
// replaces the newer list.
let latestLoad = 0

interface ProblemsPatch {
  problems: Result | null
  problemsFor?: string
  problemsAt?: number
  problemsSeed?: boolean
}

const problemsKey = (id: string) => `problems:${id}`
const modsKey = (id: string) => `mods:${id}`

interface RememberedMods {
  mods: Mod[]
  pages: Record<string, string | undefined>
}

// What the store shows for a profile that was open before in this session: its mods and its last problem check, so a
// switch back draws them at once and the reads below refresh them.
export function recallProfile(id: string) {
  const mods = getLast<RememberedMods>(modsKey(id))
  const problems = peekLast<Result>(problemsKey(id))
  return {
    ...(mods
      ? { mods: mods.mods, pages: mods.pages, loaded: true, modsFor: id }
      : { mods: [], loaded: false }),
    ...(problems
      ? {
          problems: problems.value,
          problemsFor: id,
          problemsAt: problems.at,
          problemsSeed: problems.seed,
        }
      : { problems: null, problemsSeed: false }),
  }
}

// Keeps what the store shows for the profile it holds, so recallProfile has it after a switch.
export function rememberShown(
  s: { loaded: boolean; modsFor: string; mods: Mod[]; pages: Record<string, string | undefined> },
  prev: { mods: Mod[]; pages: Record<string, string | undefined> },
) {
  if (s.loaded && s.modsFor && (s.mods !== prev.mods || s.pages !== prev.pages)) {
    putLast<RememberedMods>(modsKey(s.modsFor), { mods: s.mods, pages: s.pages })
  }
}

async function scanProblems(
  target: { game: string; id: string },
  set: (p: ProblemsPatch) => void,
  get: () => { problemsFor: string; problems: Result | null },
) {
  if (get().problemsFor !== target.id) {
    const remembered = peekLast<Result>(problemsKey(target.id))
    if (remembered) {
      set({
        problems: remembered.value,
        problemsFor: target.id,
        problemsAt: remembered.at,
        problemsSeed: remembered.seed,
      })
    } else {
      set({ problems: null })
      seedProblems(target, set, get)
    }
  }
  try {
    const problems = await ProblemsNoWait(target.game, target.id)
    if (openTarget()?.id === target.id) {
      const was = get().problemsFor === target.id ? get().problems : null
      const shown = putLast(problemsKey(target.id), problems)
      set({
        problems: was && sameValue(was, shown) ? was : shown,
        problemsFor: target.id,
        problemsAt: Date.now(),
        problemsSeed: false,
      })
    }
    const missing = missingCount(problems)
    useBadges.getState().patch(target.id, {
      missing,
      problems: problemCount(problems) - missing,
    })
  } catch (e) {
    reportError(i18n._(msg`Could not check the mods for problems`))(e)
  }
}

// The check stored by the previous run shows while the first live check of a profile runs. It is marked as a snapshot,
// so the tab keeps its fixes off until the live check answers.
function seedProblems(
  target: { game: string; id: string },
  set: (p: ProblemsPatch) => void,
  get: () => { problemsFor: string; problems: Result | null },
) {
  Last(target.game, target.id)
    .then((last) => {
      if (last && openTarget()?.id === target.id && get().problems === null) {
        putLast(problemsKey(target.id), last.result, { seed: true, at: last.at })
        set({
          problems: last.result,
          problemsFor: target.id,
          problemsAt: last.at,
          problemsSeed: true,
        })
      }
    })
    .catch(() => undefined)
}

const coalesceProblems = coalescer()

export function showUpdatesView() {
  useSettings.setState({ listGroupBy: 'status' })
  SetListGroupBy('status').catch(reportUnexpected)
  const gameId = useProfiles.getState().game?.id ?? ''
  if (gameId !== '') {
    const collapsed = loadCollapsed(gameId)
    if (collapsed.update) {
      const next: Record<string, boolean> = {}
      for (const [key, value] of Object.entries(collapsed)) {
        if (key !== 'update' && value) {
          next[key] = true
        }
      }
      persistCollapsed(gameId, next)
    }
  }
  useUpdates.getState().load().catch(reportUnexpected)
}

export async function loadMods(
  set: (p: {
    loadError?: InlineError | null
    mods?: Mod[]
    pages?: Record<string, string | undefined>
    loaded?: boolean
    modsFor?: string
  }) => void,
  get: () => {
    loadProblems: () => Promise<void>
    mods: Mod[]
    modsFor: string
    pages: Record<string, string | undefined>
  },
) {
  const target = openTarget()
  if (!target) {
    return
  }
  latestLoad += 1
  const seq = latestLoad
  const current = () => seq === latestLoad && openTarget()?.id === target.id
  set({ loadError: null })
  try {
    const [mods, pages] = await Promise.all([
      Mods(target.game, target.id),
      Pages(target.game, target.id),
    ])
    if (current()) {
      const same = get().modsFor === target.id
      const prevPages = get().pages
      set({
        mods: same ? keepUnchanged(get().mods, mods ?? [], modId) : (mods ?? []),
        pages: same && sameValue(prevPages, pages ?? {}) ? prevPages : (pages ?? {}),
        loaded: true,
        modsFor: target.id,
      })
    }
  } catch (e) {
    if (current()) {
      set({ loadError: inlineError(e) })
    }
    return
  }
  await Promise.all([get().loadProblems(), useUpdates.getState().load()])
}

// One Problems scan per profile at a time, since a scan is slow and the page, the sidebar and a profile switch all ask.
export function loadModProblems(
  set: (p: ProblemsPatch) => void,
  get: () => { problemsFor: string; problems: Result | null },
): Promise<void> {
  const target = openTarget()
  if (!target) {
    return Promise.resolve()
  }
  return coalesceProblems(target.id, () => scanProblems(target, set, get))
}
