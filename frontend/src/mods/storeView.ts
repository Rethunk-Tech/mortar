import { msg } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

type View = 'grid' | 'list'

const VIEW_KEY = 'mortar.modsView'

export function storedView(): View {
  try {
    const stored = localStorage.getItem(VIEW_KEY)
    if (stored === 'list' || stored === 'grid') {
      return stored
    }
  } catch {
    // Storage can be blocked.
  }
  return useSettings.getState().defaultModsView === 'list' ? 'list' : 'grid'
}

export function setStoredView(set: (p: { view: View }) => void, view: View) {
  set({ view })
  try {
    localStorage.setItem(VIEW_KEY, view)
  } catch {
    // Storage can be blocked; the view then lasts for this session only.
  }
}

export function viewActions(set: (p: { view: View }) => void) {
  return {
    setView: (view: View) => setStoredView(set, view),
  }
}

export function announceAlso(names: string[] | null | undefined) {
  const also = (names ?? []).filter(Boolean)
  if (also.length === 0) {
    return
  }
  useToasts.getState().push({
    kind: 'info',
    title: i18n._(msg`Also enabled ${also.join(', ')}`),
  })
}

export const fail = (title: string) => (e: unknown) => {
  useToasts
    .getState()
    .push({ kind: 'error', title, body: errorMessage(e), detail: errorDetails(e) })
}

export const open = () => {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

export function removingOf(mod: Mod | readonly Mod[] | null): Mod[] {
  if (mod === null) {
    return []
  }
  const list: readonly Mod[] = Array.isArray(mod) ? mod : [mod]
  return [...list]
}

export type { View }
