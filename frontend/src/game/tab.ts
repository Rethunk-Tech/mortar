import { create } from 'zustand'

export type TabId =
  | 'mods'
  | 'problems'
  | 'load-order'
  | 'saves'
  | 'notes'
  | 'console'
  | 'performance'

export const useTab = create<{ tab: TabId; setTab: (tab: TabId) => void }>((set) => ({
  tab: 'mods',
  setTab: (tab) => set({ tab }),
}))
