import { create } from 'zustand'
import type { ViewMode } from '../shell/viewMode.ts'

/** Browse's grid/list choice, kept while Mortar runs so switching tabs does not reset it. */
const useBrowseView = create<{ view: ViewMode; setView: (view: ViewMode) => void }>((set) => ({
  view: 'grid',
  setView: (view) => set({ view }),
}))

export { useBrowseView }
