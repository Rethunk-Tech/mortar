import { create } from 'zustand'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export interface EnableAskOffer {
  dependentName: string
  mods: Mod[]
}

export const useEnableAsk = create<{
  offers: EnableAskOffer[]
  enqueue: (offer: EnableAskOffer) => void
  dismiss: () => void
}>((set) => ({
  offers: [],
  enqueue: (offer) => set((s) => ({ offers: [...s.offers, offer] })),
  dismiss: () => set((s) => ({ offers: s.offers.slice(1) })),
}))
