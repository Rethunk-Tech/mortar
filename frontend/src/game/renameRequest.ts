import { create } from 'zustand'
import { useTab } from './tab.ts'

// renameRequest lets a menu outside the profile page start the page's inline rename: the page's name field is the
// one place a profile is renamed while it is open.
export const useRenameRequest = create<{
  id: string
  request: (id: string) => void
  clear: () => void
}>((set) => ({
  id: '',
  request: (id) => {
    // The name field lives on Home, so a rename asked for from another section opens Home first.
    useTab.getState().setTab('home')
    set({ id })
  },
  clear: () => set({ id: '' }),
}))
