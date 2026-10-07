import { create } from 'zustand'

const isDetected = (d: unknown): d is DetectedFile =>
  typeof d === 'object' &&
  d !== null &&
  ['game', 'key', 'scanner', 'name', 'file'].every(
    (k) => typeof (d as Record<string, unknown>)[k] === 'string',
  )

/** What the antivirus flagged, as the malware error's `detail` (store.DetectedError) carries it. */
export interface DetectedFile {
  game: string
  key: string
  scanner: string
  name: string
  file: string
}

/** The detection a refused install raised, or null for any other error. */
export function detectionOf(e: unknown): DetectedFile | null {
  const cause: unknown = e instanceof Error ? e.cause : undefined
  if (typeof cause !== 'object' || cause === null || !('detail' in cause)) {
    return null
  }
  return isDetected(cause.detail) ? cause.detail : null
}

/** A refused install waiting for the player's "Install anyway" answer. */
export interface Override {
  title: string
  scanner: string
  name: string
  file: string
  confirm: () => unknown
}

export const useOverride = create<{
  asking: Override | null
  ask: (o: Override) => void
  close: () => void
}>((set) => ({
  asking: null,
  ask: (asking) => set({ asking }),
  close: () => set({ asking: null }),
}))
