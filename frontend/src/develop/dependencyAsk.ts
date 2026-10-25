import { create } from 'zustand'

/** A Browse pick waiting for the player to say how the project depends on it. */
export interface DependencyAsk {
  game: string
  dir: string
  project: string
  pkg: string
  name: string
  version: string
  // Installs the package into the development profile; null when it is already there.
  install: (() => void) | null
}

export const useDependencyAsk = create<{ ask: DependencyAsk | null }>(() => ({ ask: null }))
