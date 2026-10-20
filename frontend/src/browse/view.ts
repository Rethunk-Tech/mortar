import { create } from 'zustand'
import type { ViewMode } from '../shell/viewMode.ts'

/** Browse's grid/list choice, kept while Mortar runs so switching tabs does not reset it. */
const useBrowseView = create<{
  view: ViewMode
  setView: (view: ViewMode) => void
  /** A search another screen asked Browse to run; Browse takes it once. */
  pendingQuery: string
  setPendingQuery: (query: string) => void
  /** The search box's text before it settles; only the box reads it, so typing re-renders the box alone. */
  draft: string
  /** Counts hides, so a result list asks again once a hit is gone. */
  hides: number
  noteHide: () => void
  setDraft: (draft: string) => void
}>((set) => ({
  view: 'grid',
  setView: (view) => set({ view }),
  pendingQuery: '',
  setPendingQuery: (pendingQuery) => set({ pendingQuery }),
  draft: '',
  hides: 0,
  noteHide: () => set((s) => ({ hides: s.hides + 1 })),
  setDraft: (draft) => set({ draft }),
}))

export { useBrowseView }
