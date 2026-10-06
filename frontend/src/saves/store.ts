import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetModEnabled } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import {
  Dismiss,
  RestoreDismissed,
  Saves,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
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
  reload: () => Promise<void>
  dismiss: (folder: string, id: string) => Promise<void>
  // Resolves to the history event the change recorded, or '' when nothing changed.
  enable: (game: string, profile: Profile, id: string) => Promise<string>
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
  reload: async () => {
    const [game, profileId, ...stamp] = get().key.split('/')
    if (game && profileId) {
      await get().load(game, profileId, stamp.join('/'))
    }
  },
  dismiss: async (folder, id) => {
    const previous = get().fits
    try {
      await Dismiss(folder, id)
      set({ fits: dropMissing(get().fits, id, folder) })
      useToasts.getState().push({
        kind: 'success',
        title: i18n._(msg`Dismissed for this save`),
        action: {
          label: i18n._(msg`Undo`),
          run: async () => {
            await RestoreDismissed(folder, id)
            set({ fits: previous })
          },
        },
      })
    } catch (e) {
      reportUnexpected(e)
    }
  },
  enable: async (game, profile, id) => {
    const key = (profile.entries ?? []).find((e) =>
      (e.mods ?? []).some((m) => sameId(m.id, id)),
    )?.key
    if (!key) {
      return ''
    }
    try {
      const { profile: next } = await SetModEnabled(game, profile.id, key, id, true)
      useProfiles.getState().replace(next)
      set((s) => ({ fits: dropMissing(s.fits, id) }))
      return next.lastChange ?? ''
    } catch (e) {
      reportUnexpected(e)
      return ''
    }
  },
}))
