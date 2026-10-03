import { create } from 'zustand'

interface TourReplayState {
  pending: boolean
  request: () => void
  clear: () => void
}

const useTourReplay = create<TourReplayState>((set) => ({
  pending: false,
  request: () => set({ pending: true }),
  clear: () => set({ pending: false }),
}))

export { useTourReplay }
