import { create } from 'zustand'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export interface PinTarget {
  mods: Mod[]
  pinned: boolean
}

export const usePinReasonDialog = create<{
  target: PinTarget | null
  open: (target: PinTarget) => void
  close: () => void
}>((set) => ({
  target: null,
  open: (target) => set({ target }),
  close: () => set({ target: null }),
}))

export function requestPinWithReason(mods: Mod[], pinned: boolean) {
  if (!pinned) {
    return { unpinned: true as const, mods }
  }
  usePinReasonDialog.getState().open({ mods, pinned })
  return { unpinned: false as const, mods }
}
