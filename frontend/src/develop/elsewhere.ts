import { create } from 'zustand'
import type { Adoption } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'

interface Ask {
  game: string
  dir: string
  found: Adoption
}

/** A project folder that says it is for another game than the one it was brought into. */
export const useElsewhere = create<{ asked: Ask | null; ask: (a: Ask) => void; clear: () => void }>(
  (set) => ({ asked: null, ask: (asked) => set({ asked }), clear: () => set({ asked: null }) }),
)
