import { msg } from '@lingui/core/macro'
import type { Result } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  Pages,
  Problems,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { Mods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { SetListGroupBy } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useBadges } from './badges.ts'
import { loadCollapsed, persistCollapsed } from './group.ts'
import { missingCount, problemCount } from './lookup.ts'
import { fail, open } from './storeView.ts'
import { useUpdates } from './updates.ts'

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
  const target = open()
  if (!target) {
    return
  }
  set({ loadError: '' })
  try {
    const [mods, pages] = await Promise.all([
      Mods(target.game, target.id),
      Pages(target.game, target.id),
    ])
    if (open()?.id === target.id) {
      set({ mods: mods ?? [], pages: pages ?? {}, loaded: true, modsFor: target.id })
    }
  } catch (e) {
    if (open()?.id === target.id) {
      set({ loadError: errorMessage(e) })
    }
    return
  }
  await Promise.all([get().loadProblems(), useUpdates.getState().load()])
}

export async function loadModProblems(
  set: (p: { problems: Result | null; problemsFor?: string }) => void,
  get: () => { problemsFor: string },
) {
  const target = open()
  if (!target) {
    return
  }
  if (get().problemsFor !== target.id) {
    set({ problems: null })
  }
  try {
    const problems = await Problems(target.game, target.id)
    if (open()?.id === target.id) {
      set({ problems, problemsFor: target.id })
    }
    const missing = missingCount(problems)
    useBadges.getState().patch(target.id, {
      missing,
      problems: problemCount(problems) - missing,
    })
  } catch (e) {
    fail(i18n._(msg`Could not check the mods for problems`))(e)
  }
}
