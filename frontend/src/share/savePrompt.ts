import { create } from 'zustand'

export interface Target {
  game: string
  profileId: string
  name: string
}

export const useSaveImported = create<{ target: Target | null }>(() => ({ target: null }))

/** Opens the Save as template prompt for an imported profile, named after the shared profile by default. */
export const promptSaveImported = (target: Target) => useSaveImported.setState({ target })
