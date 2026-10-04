import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SetModEnabled } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import {
  Dismiss,
  RestoreDismissed,
  Saves,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { sameId } from '../mods/lookup.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { dropMissing } from './dropMissing.ts'

interface State {
  key: string
  fits: Fit[]
  status: 'idle' | 'loading' | 'ready' | 'error'
  error: string
  load: (game: string, profileId: string, stamp: string) => Promise<void>
  dismiss: (folder: string, uniqueId: string) => Promise<void>
  enable: (game: string, profile: Profile, uniqueId: string) => Promise<void>
}

export const useSaves = create<State>((set, get) => ({
  key: '',
  fits: [],
  status: 'idle',
  error: '',
  load: async (game, profileId, stamp) => {
    const key = `${game}/${profileId}/${stamp}`
    // A reload of the same profile keeps its rows on screen; another profile's rows never show.
    const same = get().key.startsWith(`${game}/${profileId}/`)
    set({ key, fits: same ? get().fits : [], status: 'loading', error: '' })
    try {
      const fits = (await Saves(game, profileId)) ?? []
      if (get().key === key) {
        set({ fits, status: 'ready' })
      }
    } catch (e) {
      if (get().key === key) {
        set({ fits: [], status: 'error', error: errorMessage(e) })
      }
    }
  },
  dismiss: async (folder, uniqueId) => {
    const previous = get().fits
    try {
      await Dismiss(folder, uniqueId)
      set({ fits: dropMissing(get().fits, uniqueId, folder) })
      useToasts.getState().push({
        kind: 'success',
        title: i18n._(msg`Dismissed for this save`),
        action: {
          label: i18n._(msg`Undo`),
          run: async () => {
            await RestoreDismissed(folder, uniqueId)
            set({ fits: previous })
          },
        },
      })
    } catch (e) {
      reportUnexpected(e)
    }
  },
  enable: async (game, profile, uniqueId) => {
    const key = (profile.entries ?? []).find((e) =>
      (e.mods ?? []).some((m) => sameId(m.uniqueId, uniqueId)),
    )?.key
    if (!key) {
      return
    }
    try {
      useProfiles
        .getState()
        .replace((await SetModEnabled(game, profile.id, key, uniqueId, true)).profile)
      set((s) => ({ fits: dropMissing(s.fits, uniqueId) }))
    } catch (e) {
      reportUnexpected(e)
    }
  },
}))
