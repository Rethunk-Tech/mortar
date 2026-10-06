import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type {
  Profile,
  TrashItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  Create,
  Delete,
  Duplicate,
  ExportProfile,
  List,
  ListDamaged,
  ListTrash,
  OpenFolder,
  Purge,
  PurgeTrash,
  Rename,
  Reorder,
  Repair,
  Restore,
  RestoreFromZip,
  SetAppearance,
  SetHidden,
  SetLaunchOptions,
  SetLaunchSettings,
  UndoRepair,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SetLastProfile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { loadGameStatus } from '../games/status.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const fail = reportError

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
    useToasts.getState().push({ kind: 'success', title: i18n._(msg`Created ${{ name: p.name }}`) })
  } catch (e) {
    fail(i18n._(msg`Could not restore the profile`))(e)
  }
}

function splitListed(list: Profile[] | null | undefined) {
  const all = list ?? []
  return {
    profiles: all.filter((p) => !p.error),
    damaged: all.filter((p) => Boolean(p.error)),
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
  get: () => { game: GameInfo | null; open: (id: string) => void },
  set: (p: { profiles: Profile[] }) => void,
  id: string,
) {
  const { game } = get()
  if (!game) {
    return
  }
  try {
    const p = await Duplicate(game.id, id)
    const listed = splitListed((await List(game.id)) ?? [p])
    set({ ...listed })
    get().open(p.id)
  } catch (e) {
    fail(i18n._(msg`Could not duplicate the profile`))(e)
  }
}

async function refreshList(
  get: () => { game: GameInfo | null },
  set: (p: { profiles: Profile[]; damaged: Profile[] }) => void,
) {
  const { game } = get()
  if (game) {
    set(splitListed((await List(game.id)) ?? []))
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

async function purgeProfile(
  get: () => { game: GameInfo | null; loadTrash: () => Promise<void> },
  id: string,
) {
  const { game } = get()
  if (!game) {
    return
  }
  await Purge(game.id, id)
  await get().loadTrash()
}

async function purgeAllTrash(get: () => { game: GameInfo | null; loadTrash: () => Promise<void> }) {
  const { game } = get()
  if (!game) {
    return
  }
  await PurgeTrash(game.id)
  await get().loadTrash()
}

async function deleteProfile(
  get: () => {
    game: GameInfo | null
    profiles: Profile[]
    ensureOpen: () => void
    loadTrash: () => Promise<void>
    restore: (id: string) => Promise<void>
  },
  set: (
    fn: (s: { profiles: Profile[]; damaged: Profile[] }) => {
      profiles: Profile[]
      damaged: Profile[]
    },
  ) => void,
  id: string,
) {
  const { game } = get()
  if (!game) {
    return
  }
  const gone = get().profiles.find((p) => p.id === id)
  await Delete(game.id, id)
  set((s) => ({
    profiles: s.profiles.filter((x) => x.id !== id),
    damaged: s.damaged.filter((x) => x.id !== id),
  }))
  get().ensureOpen()
  useToasts.getState().push({
    kind: 'success',
    title: gone ? i18n._(msg`Deleted ${{ name: gone.name }}`) : i18n._(msg`Deleted a profile`),
    action: {
      label: i18n._(msg`Undo`),
      profileId: id,
      run: () => get().restore(id),
    },
  })
}

async function reorderProfiles(
  get: () => { game: GameInfo | null; profiles: Profile[] },
  set: (p: { profiles: Profile[] }) => void,
  ids: string[],
) {
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
}

async function repairProfile(
  get: () => {
    game: GameInfo | null
    refresh: () => Promise<void>
  },
  id: string,
) {
  const { game } = get()
  if (!game) {
    return
  }
  const repaired = await Repair(game.id, id)
  await get().refresh()
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Repaired ${{ name: repaired.name || id }}`),
    action: {
      label: i18n._(msg`Undo`),
      profileId: id,
      run: async () => {
        await UndoRepair(game.id, id)
        await get().refresh()
      },
    },
  })
}

async function restoreProfile(
  get: () => {
    game: GameInfo | null
    profiles: Profile[]
    ensureOpen: () => void
    loadTrash: () => Promise<void>
  },
  set: (patch: { profiles: Profile[]; damaged?: Profile[] }) => void,
  id: string,
) {
  const { game } = get()
  if (!game) {
    return
  }
  await Restore(game.id, id)
  set(splitListed((await List(game.id)) ?? []))
  get().ensureOpen()
  await get().loadTrash()
}

async function read(gameId: string, current: string) {
  const [{ games }, list, damagedList] = await Promise.all([
    loadGameStatus(),
    List(gameId),
    ListDamaged(gameId),
  ])
  const { profiles } = splitListed(list)
  const damaged = damagedList ?? splitListed(list).damaged
  const last = useSettings.getState().lastProfile?.[gameId]
  const openId = visibleId(profiles, current) || visibleId(profiles, last) || firstVisible(profiles)
  return { game: games.find((g) => g.id === gameId) ?? null, profiles, damaged, openId }
}

export const useProfiles = create<{
  game: GameInfo | null
  profiles: Profile[]
  damaged: Profile[]
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
  openFolder: (id: string) => Promise<void>
  repair: (id: string) => Promise<void>
  setHidden: (id: string, hidden: boolean) => Promise<void>
  remove: (id: string) => Promise<void>
  restore: (id: string) => Promise<void>
  purge: (id: string) => Promise<void>
  purgeTrash: () => Promise<void>
  reorder: (ids: string[]) => Promise<void>
  ensureOpen: () => void
}>((set, get) => ({
  game: null,
  profiles: [],
  damaged: [],
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
  openFolder: async (id) => {
    const { game } = get()
    if (!game) {
      return
    }
    try {
      await OpenFolder(game.id, id)
    } catch (e) {
      fail(i18n._(msg`Could not open the profile folder`))(e)
    }
  },
  repair: async (id) => {
    try {
      await repairProfile(get, id)
    } catch (e) {
      fail(i18n._(msg`Could not repair the profile`))(e)
    }
  },
  restoreZip: async () => {
    const { game } = get()
    if (game) {
      await restoreProfileZip(game.id, async (p) => {
        set(splitListed((await List(game.id)) ?? [p]))
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
    try {
      await deleteProfile(get, set, id)
    } catch (e) {
      fail(i18n._(msg`Could not delete the profile`))(e)
    }
    await get().loadTrash()
  },
  restore: async (id) => {
    try {
      await restoreProfile(get, set, id)
    } catch (e) {
      fail(i18n._(msg`Could not restore the profile`))(e)
    }
  },
  purge: async (id) => {
    try {
      await purgeProfile(get, id)
    } catch (e) {
      fail(i18n._(msg`Could not permanently delete the profile`))(e)
    }
  },
  purgeTrash: async () => {
    try {
      await purgeAllTrash(get)
    } catch (e) {
      fail(i18n._(msg`Could not empty the trash`))(e)
    }
  },
  reorder: (ids) => reorderProfiles(get, set, ids),
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

export const openProfileOf = (s: { profiles: Profile[]; openId: string }) =>
  s.profiles.find((p) => p.id === s.openId)

// useProfileLoader is the loader of the profile with this id (the open one by default), whose flags say which tabs
// and tools apply to it.
export function useProfileLoader(profileId?: string) {
  const game = useProfiles((s) => s.game)
  const loaderId =
    useProfiles((s) => s.profiles.find((p) => p.id === (profileId ?? s.openId))?.loader) ||
    game?.loaderId
  return game?.loaders?.find((l) => l.id === loaderId)
}
