import { create } from 'zustand'
import type { ViewMode } from '../shell/viewMode.ts'
import type { BrowseFilter } from './browseTypes.ts'

/** Browse's grid/list choice, kept while Mortar runs so switching tabs does not reset it. */
const useBrowseView = create<{
  view: ViewMode
  setView: (view: ViewMode) => void
  /** A search another screen asked Browse to run; Browse takes it once. */
  /** Category and sort choices per game, kept while Mortar runs. */
  filters: Record<string, BrowseFilter>
  setFilter: (game: string, filter: BrowseFilter) => void
  pendingQuery: string
  setPendingQuery: (query: string) => void
}>((set) => ({
  view: 'grid',
  setView: (view) => set({ view }),
  filters: {},
  setFilter: (game, filter) => set((s) => ({ filters: { ...s.filters, [game]: filter } })),
  pendingQuery: '',
  setPendingQuery: (pendingQuery) => set({ pendingQuery }),
}))

export { useBrowseView }
