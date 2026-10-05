import { useProfiles } from '../profiles/store.ts'

// The Nexus site segment of the open game; empty when the game has no Nexus source.
export const nexusDomain = (): string => useProfiles.getState().game?.sourceKeys?.nexus ?? ''
