import { create } from 'zustand'

// store.Error is "store <game>/<key>: [malware] the antivirus (<scanner>) reports <name>[ in <file>]".
const detected = /store [^/\s]+\/(\S+): .*?the antivirus \(([^)]*)\) reports (.+?)(?: in (.+))?$/s

/** What the antivirus flagged, read out of the error a refused install raised. */
export interface DetectedFile {
  key: string
  scanner: string
  name: string
  file: string
}

export function parseDetection(e: unknown): DetectedFile | null {
  const text = e instanceof Error ? e.message : String(e)
  const m = detected.exec(text)
  if (!(m?.[1] && m[2] && m[3])) {
    return null
  }
  return { key: m[1], scanner: m[2], name: m[3].trim(), file: m[4]?.trim() ?? '' }
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
