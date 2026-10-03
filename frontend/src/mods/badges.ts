import { create } from 'zustand'
import {
  Problems,
  Updates,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
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
  // loadAll fills the badges of every visible profile but skip, whose own Mods tab loads keep its badges current.
  loadAll: (game: string, profiles: Profile[], skip: string) => Promise<void>
}>((set) => ({
  byProfile: {},
  patch: (profileId, counts) =>
    set((s) => ({
      byProfile: {
        ...s.byProfile,
        [profileId]: { updates: 0, problems: 0, ...s.byProfile[profileId], ...counts },
      },
    })),
  loadAll: async (game, profiles, skip) => {
    if (useSettings.getState().backgroundBadgeChecks === false) {
      return
    }
    // One profile at a time: a cold problem check of a large profile is seconds of CPU, and the open profile's
    // own check should not have to share it.
    for (const p of profiles.filter((q) => !q.hidden && q.id !== skip)) {
      try {
        const [problems, updates] = await Promise.all([Problems(game, p.id), Updates(game, p.id)])
        set((s) => ({
          byProfile: {
            ...s.byProfile,
            [p.id]: { problems: problemCount(problems), updates: updateCount(updates, p) },
          },
        }))
      } catch {
        // No badge without an answer; opening the profile reports the failure.
      }
    }
  },
}))
