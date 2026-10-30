import { create } from 'zustand'
import type { Adoption } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'

interface Ask {
  game: string
  dir: string
  found: Adoption
}

/** The folder waiting to be confirmed. It lives outside the Develop tab so opening the dialog does not redraw its cards. */
export const useAdopting = create<{ asked: Ask | null; ask: (a: Ask) => void; clear: () => void }>(
  (set) => ({ asked: null, ask: (asked) => set({ asked }), clear: () => set({ asked: null }) }),
)
