import { create } from 'zustand'

// The Recent changes panel's open state, shared so the command palette can open it too.
export const useRecentChanges = create<{ open: boolean; setOpen: (open: boolean) => void }>(
  (set) => ({
    open: false,
    setOpen: (open) => set({ open }),
  }),
)
