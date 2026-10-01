import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { CustomCategory } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ListCustomCategories,
  SaveCustomCategories,
  SetEntryCategory,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, detail: errorMessage(e) })
}

export const useCustomCategories = create<{
  categories: CustomCategory[]
  loadedGame: string
  load: (game: string) => Promise<void>
  save: (game: string, categories: CustomCategory[]) => Promise<CustomCategory[]>
  setEntryCategory: (key: string, override: string) => Promise<void>
}>((set) => ({
  categories: [],
  loadedGame: '',
  load: async (game) => {
    if (game === '') {
      set({ categories: [], loadedGame: '' })
      return
    }
    const got = await ListCustomCategories(game)
    set({ categories: got ?? [], loadedGame: game })
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
      fail(i18n._(msg`Could not set the category`))(e)
    }
  },
}))
