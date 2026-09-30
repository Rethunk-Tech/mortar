import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { UpdatesResult } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { Updates } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useBadges } from './badges.ts'
import { updateCount } from './lookup.ts'

export const useUpdates = create<{
  updates: UpdatesResult | null
  reviewing: boolean
  load: () => Promise<void>
  setReviewing: (reviewing: boolean) => void
}>((set) => ({
  updates: null,
  reviewing: false,
  load: async () => {
    const { game, openId } = useProfiles.getState()
    if (!(game && openId)) {
      return
    }
    try {
      const updates = await Updates(game.id, openId)
      if (useProfiles.getState().openId === openId) {
        set({ updates })
      }
      const profile = useProfiles.getState().profiles.find((p) => p.id === openId)
      useBadges.getState().patch(openId, { updates: updateCount(updates, profile) })
    } catch (e) {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Couldn't reach Nexus`),
        body: errorMessage(e),
        action: {
          label: i18n._(msg`Retry now`),
          run: () => useUpdates.getState().load(),
        },
      })
    }
  },
  setReviewing: (reviewing) => set({ reviewing }),
}))
