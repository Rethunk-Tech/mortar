import { create } from 'zustand'

export type TabId = 'mods' | 'console'

export const useTab = create<{ tab: TabId; setTab: (tab: TabId) => void }>((set) => ({
  tab: 'mods',
  setTab: (tab) => set({ tab }),
}))
