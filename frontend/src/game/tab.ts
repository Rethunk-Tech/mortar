import { create } from 'zustand'

export type TabId = 'mods' | 'saves' | 'notes' | 'console'

export const useTab = create<{ tab: TabId; setTab: (tab: TabId) => void }>((set) => ({
  tab: 'mods',
  setTab: (tab) => set({ tab }),
}))
