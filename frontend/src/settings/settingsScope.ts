import { create } from 'zustand'

/** Which values a game's settings page edits: the game's, for every profile, or the open profile's own overrides. */
export const useSettingsScope = create<{ profile: boolean }>(() => ({ profile: false }))
