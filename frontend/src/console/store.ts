import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type {
  Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { Lines } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'
import { DEFAULT_FILTERS, type Filters } from './filter.ts'

const lastSeq = (entries: Entry[]) => entries[entries.length - 1]?.seq ?? 0

export const useConsole = create<{
  entries: Entry[]
  filters: Filters
  timestamps: boolean
  follow: boolean
  add: (entries: Entry[]) => void
  reset: () => void
  load: (game: string) => Promise<void>
  setSearch: (search: string) => void
  toggleLevel: (level: Level) => void
  setMods: (mods: string[]) => void
  clearFilters: () => void
  setTimestamps: (on: boolean) => void
  setFollow: (on: boolean) => void
}>((set, get) => ({
  entries: [],
  filters: DEFAULT_FILTERS,
  timestamps: true,
  follow: true,
  add: (entries) => {
    const seen = lastSeq(get().entries)
    const fresh = entries.filter((e) => e.seq > seen)
    if (fresh.length > 0) {
      set((s) => ({ entries: [...s.entries, ...fresh] }))
    }
  },
  reset: () => set({ entries: [] }),
  // Lines that arrived while the history was loading are newer than it and are kept.
  load: async (game) => {
    try {
      const history = (await Lines(game)) ?? []
      const after = lastSeq(history)
      set((s) => ({ entries: [...history, ...s.entries.filter((e) => e.seq > after)] }))
    } catch (e) {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Could not read the SMAPI log`),
        body: String(e),
      })
    }
  },
  setSearch: (search) => set((s) => ({ filters: { ...s.filters, search } })),
  toggleLevel: (level) =>
    set((s) => ({
      filters: {
        ...s.filters,
        levels: s.filters.levels.includes(level)
          ? s.filters.levels.filter((l) => l !== level)
          : [...s.filters.levels, level],
      },
    })),
  setMods: (mods) => set((s) => ({ filters: { ...s.filters, mods } })),
  clearFilters: () => set({ filters: DEFAULT_FILTERS }),
  setTimestamps: (timestamps) => set({ timestamps }),
  setFollow: (follow) => set({ follow }),
}))
