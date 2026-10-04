import { msg } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { useToasts } from '../toasts/store.ts'
import type { ModFilter } from './Toolbar.tsx'

type View = 'grid' | 'list'

const VIEW_KEY = 'mortar.modsView'

const isView = (v: unknown): v is View => v === 'list' || v === 'grid'

const isFilter = (v: unknown): v is ModFilter =>
  typeof v === 'string' &&
  ['all', 'disabled', 'update', 'problem', 'pinned', 'local', 'recent'].includes(v)
const isTags = (v: unknown): v is string[] =>
  Array.isArray(v) && v.every((t) => typeof t === 'string')

function setStoredView(set: (p: { view: View }) => void, view: View) {
  set({ view })
  writeStored(VIEW_KEY, view)
}

export function storedView(): View {
  const stored = readStored<View | ''>(VIEW_KEY, '', isView)
  if (stored !== '') {
    return stored
  }
  return useSettings.getState().defaultModsView === 'list' ? 'list' : 'grid'
}

export function viewActions(set: (p: { view: View }) => void) {
  return {
    setView: (view: View) => setStoredView(set, view),
  }
}

export const storedFilter = (profileId: string): ModFilter =>
  readStored(`mortar.modsFilter.${profileId}`, 'all', isFilter)
export const storeFilter = (profileId: string, filter: ModFilter) =>
  writeStored(`mortar.modsFilter.${profileId}`, filter)
export const storedTags = (profileId: string): string[] =>
  readStored(`mortar.modsTags.${profileId}`, [], isTags)
export const storeTags = (profileId: string, tags: string[]) =>
  writeStored(`mortar.modsTags.${profileId}`, tags)

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

export const openTarget = () => {
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
