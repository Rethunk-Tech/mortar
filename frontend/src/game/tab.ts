import { create } from 'zustand'
import { readStored, writeStored } from '../shell/useStoredState.ts'

const TABS = [
  'browse',
  'mods',
  'problems',
  'load-order',
  'saves',
  'notes',
  'console',
  'performance',
] as const
const TAB_KEY = 'mortar.tab'

const isTabId = (value: unknown): value is TabId => (TABS as readonly unknown[]).includes(value)

export type TabId = (typeof TABS)[number]

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
