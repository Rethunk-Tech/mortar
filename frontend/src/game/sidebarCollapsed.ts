import { create } from 'zustand'

const KEY = 'mortar.sidebarCollapsed'

function read(): boolean {
  try {
    return localStorage.getItem(KEY) === '1'
  } catch {
    return false
  }
}

export const useSidebarCollapsed = create<{ collapsed: boolean; toggle: () => void }>((set) => ({
  collapsed: read(),
  toggle: () =>
    set((s) => {
      const collapsed = !s.collapsed
      try {
        localStorage.setItem(KEY, collapsed ? '1' : '0')
      } catch {
        // Storage can be blocked; the rail then lasts for this session only.
      }
      return { collapsed }
    }),
}))
