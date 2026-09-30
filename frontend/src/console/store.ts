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

const lastSeq = (entries: Entry[]) => entries.at(-1)?.seq ?? 0

export const useConsole = create<{
  entries: Entry[]
  filters: Filters
  timestamps: boolean
  follow: boolean
  // Lines up to this seq were cleared from the view and stay hidden when the history is read again.
  cleared: number
  // Each request carries a fresh n so jumping to the same row twice scrolls twice.
  jump: { index: number; n: number } | null
  add: (entries: Entry[]) => void
  reset: () => void
  clear: () => void
  jumpTo: (index: number) => void
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
  cleared: 0,
  jump: null,
  add: (entries) => {
    const seen = Math.max(lastSeq(get().entries), get().cleared)
    const fresh = entries.filter((e) => e.seq > seen)
    if (fresh.length > 0) {
      set((s) => ({ entries: [...s.entries, ...fresh] }))
    }
  },
  reset: () => set({ entries: [], cleared: 0, jump: null, follow: true }),
  clear: () =>
    set((s) => ({ entries: [], jump: null, cleared: Math.max(s.cleared, lastSeq(s.entries)) })),
  jumpTo: (index) => set((s) => ({ follow: false, jump: { index, n: (s.jump?.n ?? 0) + 1 } })),
  // Lines that arrived while the history was loading are newer than it and are kept.
  load: async (game) => {
    try {
      const history = ((await Lines(game)) ?? []).filter((e) => e.seq > get().cleared)
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
