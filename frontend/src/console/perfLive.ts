import { create } from 'zustand'

/** The running game's in-game measurement as the Performance page shows it: the panel writes it, the In game card
 * reads it. since is when measuring started (ms); reason is why the game stopped measuring, when it did. */
export const usePerfLive = create<{
  measuring: boolean
  since: number
  reason: string
  top: string
}>(() => ({ measuring: false, since: 0, reason: '', top: '' }))
