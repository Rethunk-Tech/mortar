import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type {
  Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import {
  Lines,
  Send,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { DEFAULT_FILTERS, type Filters } from './filter.ts'
import { pushCommand } from './history.ts'

const lastSeq = (entries: Entry[]) => entries.at(-1)?.seq ?? 0

export const useConsole = create<{
  entries: Entry[]
  filters: Filters
  timestamps: boolean
  follow: boolean
  // Lines up to this seq were cleared from the view and stay hidden when the history is read again.
  cleared: number
  // Commands sent per game, oldest first, kept in memory only.
  history: Record<string, string[]>
  // Each request carries a fresh n so jumping to the same row twice scrolls twice.
  jump: { index: number; n: number } | null
  // The Get help dialog, opened from the Console tab or the sidebar's Support menu.
  helping: boolean
  add: (entries: Entry[]) => void
  reset: () => void
  clear: () => void
  jumpTo: (index: number) => void
  load: (game: string) => Promise<void>
  send: (game: string, command: string) => Promise<boolean>
  setSearch: (search: string) => void
  toggleLevel: (level: Level) => void
  setMods: (mods: string[]) => void
  clearFilters: () => void
  setTimestamps: (on: boolean) => void
  setFollow: (on: boolean) => void
  setHelping: (on: boolean) => void
}>((set, get) => ({
  entries: [],
  filters: DEFAULT_FILTERS,
  timestamps: true,
  follow: true,
  cleared: 0,
  history: {},
  jump: null,
  helping: false,
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
        body: errorMessage(e),
      })
    }
  },
  // Sending brings back the tail so the command's output scrolls into view.
  send: async (game, command) => {
    try {
      await Send(game, command)
    } catch (e) {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Could not run the command`),
        body: errorMessage(e),
      })
      return false
    }
    set((s) => ({
      follow: true,
      history: { ...s.history, [game]: pushCommand(s.history[game] ?? [], command) },
    }))
    return true
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
  setHelping: (helping) => set({ helping }),
}))
