import { t } from '@lingui/core/macro'
import { create } from 'zustand'
import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import type {
  Profile,
  TrashItem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  Create,
  Delete,
  Duplicate,
  List,
  ListTrash,
  Rename,
  Reorder,
  Restore,
  SetHidden,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { SetLastProfile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { loadGameStatus } from '../games/status.ts'
import { useSettings } from '../settings/store.ts'
import { useToasts } from '../toasts/store.ts'

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: String(e) })
}

const firstVisible = (profiles: Profile[]) => profiles.find((p) => !p.hidden)?.id ?? ''

export const useProfiles = create<{
  game: GameInfo | null
  profiles: Profile[]
  trash: TrashItem[]
  openId: string
  loaded: boolean
  load: (gameId: string) => Promise<void>
  open: (id: string) => void
  create: (name: string) => Promise<void>
  rename: (id: string, name: string) => Promise<boolean>
  replace: (profile: Profile) => void
  loadTrash: () => Promise<void>
  duplicate: (id: string) => Promise<void>
  setHidden: (id: string, hidden: boolean) => Promise<void>
  remove: (id: string) => Promise<void>
  restore: (id: string) => Promise<void>
  reorder: (ids: string[]) => Promise<void>
  ensureOpen: () => void
}>((set, get) => ({
  game: null,
  profiles: [],
  trash: [],
  openId: '',
  loaded: false,
  load: async (gameId) => {
    set({ loaded: false })
    try {
      const [{ games }, list] = await Promise.all([loadGameStatus(), List(gameId)])
      const profiles = list ?? []
      const last = useSettings.getState().lastProfile?.[gameId]
      const openId = profiles.find((p) => p.id === last && !p.hidden)?.id ?? firstVisible(profiles)
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
  replace: (p) => set((s) => ({ profiles: s.profiles.map((x) => (x.id === p.id ? p : x)) })),
  loadTrash: async () => {
    const { game } = get()
    if (!game) {
      return
    }
    try {
      set({ trash: (await ListTrash(game.id)) ?? [] })
    } catch (e) {
      fail(t`Could not read recently deleted profiles`)(e)
    }
  },
  duplicate: async (id) => {
    const { game } = get()
    if (!game) {
      return
    }
    try {
      const p = await Duplicate(game.id, id)
      set({ profiles: (await List(game.id)) ?? [p] })
    } catch (e) {
      fail(t`Could not duplicate the profile`)(e)
    }
  },
  setHidden: async (id, hidden) => {
    const { game } = get()
    if (!game) {
      return
    }
    try {
      get().replace(await SetHidden(game.id, id, hidden))
      get().ensureOpen()
    } catch (e) {
      fail(t`Could not change the profile's visibility`)(e)
    }
  },
  remove: async (id) => {
    const { game } = get()
    if (!game) {
      return
    }
    try {
      await Delete(game.id, id)
      set((s) => ({ profiles: s.profiles.filter((x) => x.id !== id) }))
      get().ensureOpen()
    } catch (e) {
      fail(t`Could not delete the profile`)(e)
    }
    await get().loadTrash()
  },
  restore: async (id) => {
    const { game } = get()
    if (!game) {
      return
    }
    try {
      await Restore(game.id, id)
      set({ profiles: (await List(game.id)) ?? [] })
      get().ensureOpen()
    } catch (e) {
      fail(t`Could not restore the profile`)(e)
    }
    await get().loadTrash()
  },
  reorder: async (ids) => {
    const { game, profiles } = get()
    if (!game) {
      return
    }
    const byId = new Map(profiles.map((p) => [p.id, p]))
    set({ profiles: ids.flatMap((id) => byId.get(id) ?? []) })
    try {
      await Reorder(game.id, ids)
    } catch (e) {
      fail(t`Could not save the profile order`)(e)
      set({ profiles })
    }
  },
  ensureOpen: () => {
    const { profiles, openId } = get()
    if (!profiles.some((p) => p.id === openId && !p.hidden)) {
      get().open(firstVisible(profiles))
    }
  },
}))
