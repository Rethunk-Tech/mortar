import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type {
  Duplicate,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { Problems } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  Mods,
  RemoveEntry,
  SetModEnabled,
  ShowFiles,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useBadges } from './badges.ts'
import { modId, problemCount } from './lookup.ts'
import { useUpdates } from './updates.ts'

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
  useToasts.getState().push({ kind: 'error', title, body: String(e) })
}

const open = () => {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

export const useMods = create<{
  mods: Mod[]
  loaded: boolean
  view: View
  removing: Mod | null
  problems: Result | null
  resolving: Duplicate | null
  setView: (view: View) => void
  load: () => Promise<void>
  loadProblems: () => Promise<void>
  setEnabled: (mod: Mod, enabled: boolean) => Promise<void>
  askRemove: (mod: Mod | null) => void
  remove: (mod: Mod) => Promise<void>
  showFiles: (mod: Mod) => Promise<void>
  resolve: (dup: Duplicate | null) => void
  keepCopy: (dup: Duplicate, keepKey: string) => Promise<void>
}>((set, get) => ({
  mods: [],
  loaded: false,
  view: storedView(),
  removing: null,
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
  load: async () => {
    const target = open()
    if (!target) {
      return
    }
    try {
      const mods = (await Mods(target.game, target.id)) ?? []
      if (open()?.id === target.id) {
        set({ mods, loaded: true })
      }
    } catch (e) {
      fail(i18n._(msg`Could not read the mods`))(e)
      return
    }
    await Promise.all([get().loadProblems(), useUpdates.getState().load()])
  },
  loadProblems: async () => {
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
  },
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
      useProfiles
        .getState()
        .replace(await SetModEnabled(target.game, target.id, mod.key, mod.uniqueId, enabled))
    } catch (e) {
      flip(!enabled)
      fail(i18n._(msg`Could not switch ${mod.name}`))(e)
      return
    }
    await get().loadProblems()
  },
  askRemove: (removing) => set({ removing }),
  remove: async (mod) => {
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
  },
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
          .replace(await SetModEnabled(target.game, target.id, c.key, dup.uniqueId, false))
      }
    } catch (e) {
      fail(i18n._(msg`Could not switch off the other copy of ${dup.name}`))(e)
    }
    set({ resolving: null })
    await get().load()
  },
}))

export type { View }
