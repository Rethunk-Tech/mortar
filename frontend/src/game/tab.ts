import { create } from 'zustand'

export type TabId =
  | 'browse'
  | 'mods'
  | 'problems'
  | 'load-order'
  | 'saves'
  | 'notes'
  | 'console'
  | 'performance'

export const useTab = create<{
  tab: TabId
  setTab: (tab: TabId) => void
  pendingLoadOrder: { id: string; fallback: string } | null
  revealLoadOrder: (id: string, fallback?: string) => void
  takePendingLoadOrder: () => { id: string; fallback: string } | null
}>((set, get) => ({
  tab: 'mods',
  pendingLoadOrder: null,
  setTab: (tab) => set({ tab }),
  revealLoadOrder: (id, fallback = '') =>
    set({ tab: 'load-order', pendingLoadOrder: { id, fallback } }),
  takePendingLoadOrder: () => {
    const pending = get().pendingLoadOrder
    set({ pendingLoadOrder: null })
    return pending
  },
}))
