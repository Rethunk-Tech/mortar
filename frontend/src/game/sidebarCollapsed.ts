import { useMediaQuery } from '@mui/material'
import { create } from 'zustand'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { compactQuery } from './compact.ts'

const KEY = 'mortar.sidebarCollapsed'

const isBool = (v: unknown): v is boolean => typeof v === 'boolean'

// The viewer's saved choice; the sidebar starts expanded.
export const readCollapsed = (): boolean => readStored(KEY, false, isBool)

export const useSidebarCollapsed = create<{ collapsed: boolean; toggle: () => void }>((set) => ({
  collapsed: readCollapsed(),
  toggle: () =>
    set((s) => {
      const collapsed = !s.collapsed
      writeStored(KEY, collapsed)
      return { collapsed }
    }),
}))

// Whether the sidebar shows as the icon rail: by the viewer's choice, or always in a narrow window (`narrow`),
// where there is no room to expand it.
export function useRail(): { rail: boolean; narrow: boolean } {
  const collapsed = useSidebarCollapsed((s) => s.collapsed)
  const narrow = useMediaQuery(compactQuery)
  return { rail: narrow || collapsed, narrow }
}
