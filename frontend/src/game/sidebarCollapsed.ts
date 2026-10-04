import { create } from 'zustand'
import { readStored, writeStored } from '../shell/useStoredState.ts'

const KEY = 'mortar.sidebarCollapsed'

const isBool = (v: unknown): v is boolean => typeof v === 'boolean'

export const useSidebarCollapsed = create<{ collapsed: boolean; toggle: () => void }>((set) => ({
  collapsed: readStored(KEY, false, isBool),
  toggle: () =>
    set((s) => {
      const collapsed = !s.collapsed
      writeStored(KEY, collapsed)
      return { collapsed }
    }),
}))
