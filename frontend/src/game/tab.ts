import { create } from 'zustand'
import { readStored, writeStored } from '../shell/useStoredState.ts'

const TABS = [
  'home',
  'browse',
  'mods',
  'problems',
  'load-order',
  'saves',
  'console',
  'performance',
] as const
const TAB_KEY = 'mortar.tab'

const isTabId = (value: unknown): value is TabId => (TABS as readonly unknown[]).includes(value)

export type TabId = (typeof TABS)[number]

// The sidebar's tabs and the workspace they switch share these ids, so each tab points at its panel.
export const PANEL_ID = 'profile-panel'
export const tabDomId = (tab: TabId) => `profile-tab-${tab}`

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
