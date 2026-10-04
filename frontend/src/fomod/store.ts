import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  FomodAsk,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  FomodPreview,
  InstallFomod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  AnswerFomod,
  Skip,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { useMods } from '../mods/store.ts'
import { openTarget } from '../mods/storeView.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

let watching = false

interface FomodSession {
  game: string
  profileId: string
  key: string
  source: Source
  queueId?: string
  ask: FomodAsk
}

export type { FomodSession }

export const useFomod = create<{
  session: FomodSession | null
  open: (s: FomodSession) => void
  close: () => void
  refresh: (choices: Record<string, Record<string, string[]>>) => Promise<void>
  install: (choices: Record<string, Record<string, string[]>>) => Promise<void>
}>((set, get) => ({
  session: null,
  open: (session) => set({ session }),
  close: () => {
    const s = get().session
    if (s?.queueId) {
      Skip(s.queueId).catch(reportUnexpected)
    }
    set({ session: null })
  },
  refresh: async (choices) => {
    const s = get().session
    if (!s) {
      return
    }
    const ask = await FomodPreview(s.game, s.profileId, s.key, choices)
    set({ session: { ...s, ask } })
  },
  install: async (choices) => {
    const s = get().session
    if (!s) {
      return
    }
    if (s.queueId) {
      await AnswerFomod(s.queueId, choices)
      set({ session: null })
      return
    }
    const res = await InstallFomod(s.game, s.profileId, s.key, s.source, choices)
    if (res.profile) {
      useProfiles.getState().replace(res.profile)
    }
    await useMods.getState().load()
    set({ session: null })
  },
}))

export function watchFomodQueue() {
  if (watching || typeof Events.On !== 'function') {
    return
  }
  watching = true
  Events.On('queue:changed', (event) => {
    const items = event.data.items ?? []
    const key = items.find((i) => i.state === 'needs-fomod')?.fomodKey
    const waiting = items.find((i) => i.state === 'needs-fomod' && i.fomodKey)
    if (!(waiting && key)) {
      return
    }
    const at = openTarget()
    if (!at) {
      return
    }
    const cur = useFomod.getState().session
    if (cur?.queueId === waiting.id) {
      return
    }
    FomodPreview(at.game, at.id, key, {}).then((ask) => {
      useFomod.getState().open({
        game: at.game,
        profileId: at.id,
        key,
        source: ask.source,
        queueId: waiting.id,
        ask,
      })
    }, reportUnexpected)
  })
}
