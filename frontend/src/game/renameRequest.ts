import { create } from 'zustand'

// renameRequest lets a menu outside the profile page start the page's inline rename: the page's name field is the
// one place a profile is renamed while it is open.
export const useRenameRequest = create<{
  id: string
  request: (id: string) => void
  clear: () => void
}>((set) => ({
  id: '',
  request: (id) => set({ id }),
  clear: () => set({ id: '' }),
}))
