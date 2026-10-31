import { msg } from '@lingui/core/macro'
import { sameValue } from '@rethunk/react-load-hooks'
import { create } from 'zustand'
import type { CustomCategory } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ListCustomCategories,
  SaveCustomCategories,
  SetEntryCategory,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError } from '../toasts/report.ts'

export function categoriesDiffer(a: CustomCategory[], b: CustomCategory[]) {
  return !sameValue(a, b)
}

export const useCustomCategories = create<{
  categories: CustomCategory[]
  loadedGame: string
  load: (game: string) => Promise<void>
  save: (game: string, categories: CustomCategory[]) => Promise<CustomCategory[]>
  setEntryCategory: (key: string, override: string) => Promise<void>
}>((set, get) => ({
  categories: [],
  loadedGame: '',
  load: async (game) => {
    const got = game === '' ? [] : ((await ListCustomCategories(game)) ?? [])
    // An unchanged answer keeps the stored list, so the mods list does not redraw each time the tab opens.
    const prev = get().categories
    set({ categories: categoriesDiffer(prev, got) ? got : prev, loadedGame: game })
  },
  save: async (game, categories) => {
    const saved = await SaveCustomCategories(game, categories)
    const next = saved ?? []
    set({ categories: next, loadedGame: game })
    return next
  },
  setEntryCategory: async (key, override) => {
    const game = useProfiles.getState().game?.id
    const id = useProfiles.getState().openId
    if (!(game && id)) {
      return
    }
    try {
      useProfiles.getState().replace(await SetEntryCategory(game, id, key, override))
    } catch (e) {
      reportError(i18n._(msg`Could not set the category`))(e)
    }
  },
}))
