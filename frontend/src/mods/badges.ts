import { create } from 'zustand'
import {
  Problems,
  Updates,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { problemCount, updateCount } from './lookup.ts'

interface Counts {
  updates: number
  problems: number
}

// What the sidebar shows beside each profile. A profile whose check failed keeps its last counts (or none):
// the checks warn and never block.
export const useBadges = create<{
  byProfile: Record<string, Counts>
  patch: (profileId: string, counts: Partial<Counts>) => void
  loadAll: (game: string, profiles: Profile[]) => Promise<void>
}>((set) => ({
  byProfile: {},
  patch: (profileId, counts) =>
    set((s) => ({
      byProfile: {
        ...s.byProfile,
        [profileId]: { updates: 0, problems: 0, ...s.byProfile[profileId], ...counts },
      },
    })),
  loadAll: async (game, profiles) => {
    await Promise.all(
      profiles
        .filter((p) => !p.hidden)
        .map(async (p) => {
          try {
            const [problems, updates] = await Promise.all([
              Problems(game, p.id),
              Updates(game, p.id),
            ])
            set((s) => ({
              byProfile: {
                ...s.byProfile,
                [p.id]: { problems: problemCount(problems), updates: updateCount(updates) },
              },
            }))
          } catch {
            // No badge without an answer; opening the profile reports the failure.
          }
        }),
    )
  },
}))
