import { msg } from '@lingui/core/macro'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import {
  Pages,
  Problems,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { Mods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SetListGroupBy } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { coalescer } from '../shell/coalesce.ts'
import { errorMessage, reportError, reportUnexpected } from '../toasts/report.ts'
import { useBadges } from './badges.ts'
import { loadCollapsed, persistCollapsed } from './group.ts'
import { missingCount, problemCount } from './lookup.ts'
import { openTarget } from './storeView.ts'
import { useUpdates } from './updates.ts'

// Loads overlap when installs finish back to back; only the newest one may write, or an older snapshot
// replaces the newer list.
let latestLoad = 0

async function scanProblems(
  target: { game: string; id: string },
  set: (p: { problems: Result | null; problemsFor?: string }) => void,
  get: () => { problemsFor: string },
) {
  if (get().problemsFor !== target.id) {
    set({ problems: null })
  }
  try {
    const problems = await Problems(target.game, target.id)
    if (openTarget()?.id === target.id) {
      set({ problems, problemsFor: target.id })
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
    loadError?: string
    mods?: Mod[]
    pages?: Record<string, string | undefined>
    loaded?: boolean
    modsFor?: string
  }) => void,
  get: () => { loadProblems: () => Promise<void> },
) {
  const target = openTarget()
  if (!target) {
    return
  }
  latestLoad += 1
  const seq = latestLoad
  const current = () => seq === latestLoad && openTarget()?.id === target.id
  set({ loadError: '' })
  try {
    const [mods, pages] = await Promise.all([
      Mods(target.game, target.id),
      Pages(target.game, target.id),
    ])
    if (current()) {
      set({ mods: mods ?? [], pages: pages ?? {}, loaded: true, modsFor: target.id })
    }
  } catch (e) {
    if (current()) {
      set({ loadError: errorMessage(e) })
    }
    return
  }
  await Promise.all([get().loadProblems(), useUpdates.getState().load()])
}

// One Problems scan per profile at a time, since a scan is slow and the page, the sidebar and a profile switch all ask.
export function loadModProblems(
  set: (p: { problems: Result | null; problemsFor?: string }) => void,
  get: () => { problemsFor: string },
): Promise<void> {
  const target = openTarget()
  if (!target) {
    return Promise.resolve()
  }
  return coalesceProblems(target.id, () => scanProblems(target, set, get))
}
