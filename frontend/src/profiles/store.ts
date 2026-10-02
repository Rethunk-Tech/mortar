import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
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
  ExportProfile,
  List,
  ListTrash,
  Rename,
  Reorder,
  Restore,
  RestoreFromZip,
  SetAppearance,
  SetHidden,
  SetLaunchOptions,
  SetLaunchSettings,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { SetLastProfile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { loadGameStatus } from '../games/status.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(e) })
}

async function exportProfileZip(gameId: string, id: string) {
  try {
    const path = await ExportProfile(gameId, id)
    if (path) {
      useToasts.getState().push({ kind: 'success', title: i18n._(msg`Profile exported`) })
    }
  } catch (e) {
    fail(i18n._(msg`Could not export the profile`))(e)
  }
}

async function restoreProfileZip(gameId: string, apply: (p: Profile) => Promise<void>) {
  try {
    const p = await RestoreFromZip(gameId)
    if (!p?.id) {
      return
    }
    await apply(p)
    useToasts.getState().push({ kind: 'success', title: i18n._(msg`Created “${p.name}”`) })
  } catch (e) {
    fail(i18n._(msg`Could not restore the profile`))(e)
  }
}

const visibleId = (profiles: Profile[], id: string | undefined) =>
  profiles.find((p) => p.id === id && !p.hidden)?.id ?? ''

const firstVisible = (profiles: Profile[]) => profiles.find((p) => !p.hidden)?.id ?? ''

function ensureVisible(
  get: () => { profiles: Profile[]; openId: string; open: (id: string) => void },
) {
  const { profiles, openId } = get()
  if (!profiles.some((p) => p.id === openId && !p.hidden)) {
    get().open(firstVisible(profiles))
  }
}

async function duplicateProfile(
  get: () => { game: GameInfo | null },
  set: (p: { profiles: Profile[] }) => void,
  id: string,
) {
  const { game } = get()
  if (!game) {
    return
  }
  try {
    const p = await Duplicate(game.id, id)
    set({ profiles: (await List(game.id)) ?? [p] })
  } catch (e) {
    fail(i18n._(msg`Could not duplicate the profile`))(e)
  }
}

async function refreshList(
  get: () => { game: GameInfo | null },
  set: (p: { profiles: Profile[] }) => void,
) {
  const { game } = get()
  if (game) {
    set({ profiles: (await List(game.id)) ?? [] })
  }
}

async function applyLaunchOptions(
  get: () => { game: GameInfo | null; replace: (p: Profile) => void },
  id: string,
  options: string,
) {
  const { game } = get()
  if (game) {
    get().replace(await SetLaunchOptions(game.id, id, options))
  }
}

async function applyLaunchSettings(
  get: () => { game: GameInfo | null; replace: (p: Profile) => void },
  id: string,
  prefix: string,
  env: string,
) {
  const { game } = get()
  if (game) {
    get().replace(await SetLaunchSettings(game.id, id, prefix, env))
  }
}

async function read(gameId: string, current: string) {
  const [{ games }, list] = await Promise.all([loadGameStatus(), List(gameId)])
  const profiles = list ?? []
  const last = useSettings.getState().lastProfile?.[gameId]
  const openId = visibleId(profiles, current) || visibleId(profiles, last) || firstVisible(profiles)
  return { game: games.find((g) => g.id === gameId) ?? null, profiles, openId }
}

export const useProfiles = create<{
  game: GameInfo | null
  profiles: Profile[]
  trash: TrashItem[]
  openId: string
  loaded: boolean
  // The game whose last load failed, for Retry; empty when nothing failed.
  failed: string
  load: (gameId: string) => Promise<void>
  open: (id: string) => void
  create: (name: string) => Promise<void>
  // Rejects with the reason when the name is refused, for the field to show.
  rename: (id: string, name: string) => Promise<boolean>
  setAppearance: (id: string, color: string, icon: string, description: string) => Promise<void>
  setLaunchOptions: (id: string, options: string) => Promise<void>
  setLaunchSettings: (id: string, prefix: string, env: string) => Promise<void>
  replace: (profile: Profile) => void
  refresh: () => Promise<void>
  loadTrash: () => Promise<void>
  duplicate: (id: string) => Promise<void>
  exportProfile: (id: string) => Promise<void>
  restoreZip: () => Promise<void>
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
  failed: '',
  load: async (gameId) => {
    // Coming back to the same game refreshes behind the screen instead of blanking it.
    const same = get().loaded && get().game?.id === gameId
    if (!same) {
      set({ loaded: false, failed: '' })
    }
    try {
      set({ ...(await read(gameId, same ? get().openId : '')), loaded: true, failed: '' })
    } catch (e) {
      if (!same) {
        set({ failed: gameId })
      }
      fail(i18n._(msg`Could not read your profiles`))(e)
    }
  },
  open: (id) => {
    const { game } = get()
    set({ openId: id })
    if (game) {
      SetLastProfile(game.id, id).catch(fail(i18n._(msg`Could not save the open profile`)))
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
    const p = await Rename(game.id, id, name)
    set((s) => ({ profiles: s.profiles.map((x) => (x.id === id ? p : x)) }))
    return true
  },
  setAppearance: async (id, color, icon, description) => {
    const { game } = get()
    if (game) {
      get().replace(await SetAppearance(game.id, id, color, icon, description))
    }
  },
  setLaunchOptions: (id, options) => applyLaunchOptions(get, id, options),
  setLaunchSettings: (id, prefix, env) => applyLaunchSettings(get, id, prefix, env),
  replace: (p) => set((s) => ({ profiles: s.profiles.map((x) => (x.id === p.id ? p : x)) })),
  refresh: () => refreshList(get, set),
  loadTrash: async () => {
    const { game } = get()
    if (game) {
      try {
        set({ trash: (await ListTrash(game.id)) ?? [] })
      } catch (e) {
        fail(i18n._(msg`Could not read recently deleted profiles`))(e)
      }
    }
  },
  duplicate: (id) => duplicateProfile(get, set, id),
  exportProfile: async (id) => {
    const { game } = get()
    if (game) {
      await exportProfileZip(game.id, id)
    }
  },
  restoreZip: async () => {
    const { game } = get()
    if (game) {
      await restoreProfileZip(game.id, async (p) => {
        set({ profiles: (await List(game.id)) ?? [p] })
        get().open(p.id)
      })
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
      fail(i18n._(msg`Could not change the profile's visibility`))(e)
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
      fail(i18n._(msg`Could not delete the profile`))(e)
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
      fail(i18n._(msg`Could not restore the profile`))(e)
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
      fail(i18n._(msg`Could not save the profile order`))(e)
      set({ profiles })
    }
  },
  ensureOpen: () => ensureVisible(get),
}))

// The command line changes profiles through the running app, which then names the game; reload it when it is open.
export function initProfilesChanged() {
  if (typeof Events.On !== 'function') {
    return
  }
  Events.On('profiles:changed', (event) => {
    const s = useProfiles.getState()
    if (s.game?.id === event.data) {
      s.refresh().catch(() => undefined)
    }
  })
}
