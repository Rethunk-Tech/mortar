import { create } from 'zustand'
import { readStored, writeStored } from '../shell/useStoredState.ts'

export type TabId =
  | 'browse'
  | 'mods'
  | 'problems'
  | 'load-order'
  | 'saves'
  | 'notes'
  | 'console'
  | 'performance'

const TABS: readonly unknown[] = [
  'browse',
  'mods',
  'problems',
  'load-order',
  'saves',
  'notes',
  'console',
  'performance',
]
const TAB_KEY = 'mortar.tab'

const isTabId = (value: unknown): value is TabId => TABS.includes(value)

export const useTab = create<{
  tab: TabId
  setTab: (tab: TabId) => void
  pendingLoadOrder: { id: string; fallback: string } | null
  revealLoadOrder: (id: string, fallback?: string) => void
  takePendingLoadOrder: () => { id: string; fallback: string } | null
}>((set, get) => ({
  tab: readStored<TabId>(TAB_KEY, 'mods', isTabId),
  pendingLoadOrder: null,
  setTab: (tab) => {
    writeStored(TAB_KEY, tab)
    set({ tab })
  },
  revealLoadOrder: (id, fallback = '') =>
    set({ tab: 'load-order', pendingLoadOrder: { id, fallback } }),
  takePendingLoadOrder: () => {
    const pending = get().pendingLoadOrder
    set({ pendingLoadOrder: null })
    return pending
  },
}))
