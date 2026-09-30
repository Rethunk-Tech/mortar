import { t } from '@lingui/core/macro'
import { create } from 'zustand'
import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  Create,
  List,
  Rename,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { SetLastProfile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { loadGameStatus } from '../games/status.ts'
import { useSettings } from '../settings/store.ts'
import { useToasts } from '../toasts/store.ts'

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: String(e) })
}

export const useProfiles = create<{
  game: GameInfo | null
  profiles: Profile[]
  openId: string
  loaded: boolean
  load: (gameId: string) => Promise<void>
  open: (id: string) => void
  create: (name: string) => Promise<void>
  rename: (id: string, name: string) => Promise<boolean>
}>((set, get) => ({
  game: null,
  profiles: [],
  openId: '',
  loaded: false,
  load: async (gameId) => {
    set({ loaded: false })
    try {
      const [{ games }, list] = await Promise.all([loadGameStatus(), List(gameId)])
      const profiles = list ?? []
      const last = useSettings.getState().lastProfile?.[gameId]
      const openId = profiles.find((p) => p.id === last)?.id ?? profiles[0]?.id ?? ''
      set({ game: games.find((g) => g.id === gameId) ?? null, profiles, openId, loaded: true })
    } catch (e) {
      fail(t`Could not read your profiles`)(e)
    }
  },
  open: (id) => {
    const { game } = get()
    set({ openId: id })
    if (game) {
      SetLastProfile(game.id, id).catch(fail(t`Could not save the open profile`))
    }
  },
  create: async (name) => {
    const { game } = get()
    if (!game) {
      throw new Error('no game open')
    }
    const p = await Create(game.id, name)
    set((s) => ({ profiles: [...s.profiles, p] }))
    get().open(p.id)
  },
  rename: async (id, name) => {
    const { game } = get()
    if (!game) {
      return false
    }
    try {
      const p = await Rename(game.id, id, name)
      set((s) => ({ profiles: s.profiles.map((x) => (x.id === id ? p : x)) }))
      return true
    } catch (e) {
      fail(t`Could not rename the profile`)(e)
      return false
    }
  },
}))
