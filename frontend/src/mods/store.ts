import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type {
  AssetConflict,
  Duplicate,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  DismissAbandonedMod,
  DismissAssetConflict,
  DismissListedRequirement,
  Pages,
  Problems,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  Mods,
  OpenConfig,
  RemoveEntries,
  RemoveEntry,
  SetEntryNoteTags,
  SetModEnabled,
  SetModsEnabled,
  SetPinned,
  SetSkipVersion,
  ShowFiles,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useBadges } from './badges.ts'
import { modId, problemCount } from './lookup.ts'
import { useSelection } from './selection.ts'
import { useUpdates } from './updates.ts'

function announceAlso(names: string[] | null | undefined) {
  const also = (names ?? []).filter(Boolean)
  if (also.length === 0) {
    return
  }
  useToasts.getState().push({
    kind: 'info',
    title: i18n._(msg`Also enabled ${also.join(', ')}`),
  })
}

type View = 'grid' | 'list'

const VIEW_KEY = 'mortar.modsView'

function storedView(): View {
  try {
    return localStorage.getItem(VIEW_KEY) === 'list' ? 'list' : 'grid'
  } catch {
    return 'grid'
  }
}

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(e) })
}

const open = () => {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

function removingOf(mod: Mod | readonly Mod[] | null): Mod[] {
  if (mod === null) {
    return []
  }
  const list: readonly Mod[] = Array.isArray(mod) ? mod : [mod]
  return [...list]
}

async function loadMods(
  set: (p: {
    loadError?: string
    mods?: Mod[]
    pages?: Record<string, string | undefined>
    loaded?: boolean
  }) => void,
  get: () => { loadProblems: () => Promise<void> },
) {
  const target = open()
  if (!target) {
    return
  }
  set({ loadError: '' })
  try {
    const [mods, pages] = await Promise.all([
      Mods(target.game, target.id),
      Pages(target.game, target.id),
    ])
    if (open()?.id === target.id) {
      set({ mods: mods ?? [], pages: pages ?? {}, loaded: true })
    }
  } catch (e) {
    if (open()?.id === target.id) {
      set({ loadError: errorMessage(e) })
    }
    return
  }
  await Promise.all([get().loadProblems(), useUpdates.getState().load()])
}

async function loadModProblems(set: (p: { problems: Result | null }) => void) {
  const target = open()
  if (!target) {
    return
  }
  try {
    const problems = await Problems(target.game, target.id)
    if (open()?.id === target.id) {
      set({ problems })
    }
    useBadges.getState().patch(target.id, { problems: problemCount(problems) })
  } catch (e) {
    fail(i18n._(msg`Could not check the mods for problems`))(e)
  }
}

async function enableMany(
  set: (fn: (s: { mods: Mod[] }) => { mods: Mod[] }) => void,
  get: () => { mods: Mod[]; loadProblems: () => Promise<void> },
  mods: Mod[],
  enabled: boolean,
) {
  const target = open()
  if (!target || mods.length === 0) {
    return
  }
  const ids = new Set(mods.map((m) => modId(m)))
  const prev = new Map(get().mods.map((m) => [modId(m), m.enabled]))
  set((s) => ({
    mods: s.mods.map((m) => (ids.has(modId(m)) ? { ...m, enabled } : m)),
  }))
  try {
    useProfiles.getState().replace(
      await SetModsEnabled(
        target.game,
        target.id,
        mods.map((m) => ({ key: m.key, uniqueId: m.uniqueId })),
        enabled,
      ).then((r) => {
        announceAlso(r.alsoEnabled)
        return r.profile
      }),
    )
  } catch (e) {
    set((s) => ({
      mods: s.mods.map((m) => {
        const was = prev.get(modId(m))
        return was === undefined ? m : { ...m, enabled: was }
      }),
    }))
    fail(i18n._(msg`Could not switch the selected mods`))(e)
    return
  }
  await get().loadProblems()
}

async function dropMods(get: () => { load: () => Promise<void> }, mods: Mod[]) {
  const target = open()
  if (!target || mods.length === 0) {
    return
  }
  const keys = [...new Set(mods.map((m) => m.key))]
  try {
    useProfiles.getState().replace(await RemoveEntries(target.game, target.id, keys))
  } catch (e) {
    fail(i18n._(msg`Could not remove the selected mods`))(e)
  }
  await get().load()
  useSelection.getState().clear()
}

async function dismissAbandonedMod(
  get: () => { loadProblems: () => Promise<void> },
  uniqueId: string,
) {
  const target = open()
  if (!target) {
    return
  }
  try {
    await DismissAbandonedMod(target.game, target.id, uniqueId)
  } catch (e) {
    fail(i18n._(msg`Could not dismiss the warning`))(e)
    return
  }
  await get().loadProblems()
}

async function dismissListedRequirement(
  get: () => { loadProblems: () => Promise<void> },
  uniqueId: string,
) {
  const target = open()
  if (!target) {
    return
  }
  try {
    await DismissListedRequirement(target.game, target.id, uniqueId)
  } catch (e) {
    fail(i18n._(msg`Could not dismiss the warning`))(e)
    return
  }
  await get().loadProblems()
}

async function dismissAssetConflict(
  get: () => { loadProblems: () => Promise<void> },
  conflict: AssetConflict,
) {
  const target = open()
  if (!target) {
    return
  }
  try {
    await DismissAssetConflict(target.game, target.id, conflict.kind, conflict.target)
  } catch (e) {
    fail(i18n._(msg`Could not dismiss the overlap`))(e)
    return
  }
  await get().loadProblems()
}

async function dropMod(get: () => { load: () => Promise<void> }, mod: Mod) {
  const target = open()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(await RemoveEntry(target.game, target.id, mod.key))
  } catch (e) {
    fail(i18n._(msg`Could not remove ${mod.name}`))(e)
  }
  await get().load()
  useSelection.getState().clear()
}

export const useMods = create<{
  mods: Mod[]
  loaded: boolean
  loadError: string
  pages: Record<string, string | undefined>
  view: View
  removing: Mod[]
  problems: Result | null
  resolving: Duplicate | null
  setView: (view: View) => void
  load: () => Promise<void>
  loadProblems: () => Promise<void>
  setEnabled: (mod: Mod, enabled: boolean) => Promise<void>
  setEnabledMany: (mods: Mod[], enabled: boolean) => Promise<void>
  setPinned: (mod: Mod, pinned: boolean) => Promise<void>
  setSkipVersion: (mod: Mod, version: string) => Promise<void>
  setNoteTags: (mod: Mod, note: string, tags: string[]) => Promise<void>
  askRemove: (mod: Mod | readonly Mod[] | null) => void
  remove: (mod: Mod) => Promise<void>
  removeMany: (mods: Mod[]) => Promise<void>
  showFiles: (mod: Mod) => Promise<void>
  openConfig: (mod: Mod) => Promise<void>
  resolve: (dup: Duplicate | null) => void
  keepCopy: (dup: Duplicate, keepKey: string) => Promise<void>
  dismissAsset: (conflict: AssetConflict) => Promise<void>
  dismissAbandoned: (uniqueId: string) => Promise<void>
  dismissListed: (uniqueId: string) => Promise<void>
}>((set, get) => ({
  mods: [],
  loaded: false,
  loadError: '',
  pages: {},
  view: storedView(),
  removing: [],
  problems: null,
  resolving: null,
  setView: (view) => {
    set({ view })
    try {
      localStorage.setItem(VIEW_KEY, view)
    } catch {
      // Storage can be blocked; the view then lasts for this session only.
    }
  },
  load: () => loadMods(set, get),
  loadProblems: () => loadModProblems(set),
  setEnabled: async (mod, enabled) => {
    const target = open()
    if (!target) {
      return
    }
    const flip = (on: boolean) =>
      set((s) => ({
        mods: s.mods.map((m) => (modId(m) === modId(mod) ? { ...m, enabled: on } : m)),
      }))
    flip(enabled)
    try {
      const got = await SetModEnabled(target.game, target.id, mod.key, mod.uniqueId, enabled)
      useProfiles.getState().replace(got.profile)
      announceAlso(got.alsoEnabled)
    } catch (e) {
      flip(!enabled)
      fail(i18n._(msg`Could not switch ${mod.name}`))(e)
      return
    }
    await get().loadProblems()
  },
  setEnabledMany: (mods, enabled) => enableMany(set, get, mods, enabled),
  setPinned: async (mod, pinned) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      useProfiles.getState().replace(await SetPinned(target.game, target.id, mod.key, pinned))
    } catch (e) {
      fail(i18n._(pinned ? msg`Could not pin ${mod.name}` : msg`Could not unpin ${mod.name}`))(e)
      return
    }
    await useUpdates.getState().load()
  },
  setSkipVersion: async (mod, version) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      useProfiles.getState().replace(await SetSkipVersion(target.game, target.id, mod.key, version))
    } catch (e) {
      fail(
        i18n._(
          version === ''
            ? msg`Could not show the skipped update for ${mod.name}`
            : msg`Could not skip the update for ${mod.name}`,
        ),
      )(e)
      return
    }
    await useUpdates.getState().load()
  },
  setNoteTags: async (mod, note, tags) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      useProfiles
        .getState()
        .replace(await SetEntryNoteTags(target.game, target.id, mod.key, note, tags))
    } catch (e) {
      fail(i18n._(msg`Could not save the note and tags for ${mod.name}`))(e)
    }
  },
  askRemove: (mod) => set({ removing: removingOf(mod) }),
  remove: (mod) => dropMod(get, mod),
  removeMany: (mods) => dropMods(get, mods),
  showFiles: async (mod) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      await ShowFiles(target.game, target.id, mod.key, mod.uniqueId)
    } catch (e) {
      fail(i18n._(msg`Could not open the folder of ${mod.name}`))(e)
    }
  },
  openConfig: async (mod) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      await OpenConfig(target.game, target.id, mod.key, mod.uniqueId)
    } catch (e) {
      fail(i18n._(msg`Could not open config.json of ${mod.name}`))(e)
    }
  },
  resolve: (resolving) => set({ resolving }),
  keepCopy: async (dup, keepKey) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      for (const c of (dup.copies ?? []).filter((x) => x.key !== keepKey)) {
        useProfiles
          .getState()
          .replace(
            (await SetModEnabled(target.game, target.id, c.key, dup.uniqueId, false)).profile,
          )
      }
    } catch (e) {
      fail(i18n._(msg`Could not switch off the other copy of ${dup.name}`))(e)
    }
    set({ resolving: null })
    await get().load()
  },
  dismissAsset: (conflict) => dismissAssetConflict(get, conflict),
  dismissAbandoned: (uniqueId) => dismissAbandonedMod(get, uniqueId),
  dismissListed: (uniqueId) => dismissListedRequirement(get, uniqueId),
}))

export type { View }
