import { t } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  Mods,
  RemoveEntry,
  SetModEnabled,
  ShowFiles,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'

export type View = 'grid' | 'list'

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
  setView: (view: View) => void
  load: () => Promise<void>
  setEnabled: (mod: Mod, enabled: boolean) => Promise<void>
  askRemove: (mod: Mod | null) => void
  remove: (mod: Mod) => Promise<void>
  showFiles: (mod: Mod) => Promise<void>
}>((set, get) => ({
  mods: [],
  loaded: false,
  view: storedView(),
  removing: null,
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
      fail(t`Could not read the mods`)(e)
    }
  },
  setEnabled: async (mod, enabled) => {
    const target = open()
    if (!target) {
      return
    }
    const flip = (on: boolean) =>
      set((s) => ({
        mods: s.mods.map((m) => (m.uniqueId === mod.uniqueId ? { ...m, enabled: on } : m)),
      }))
    flip(enabled)
    try {
      useProfiles
        .getState()
        .replace(await SetModEnabled(target.game, target.id, mod.uniqueId, enabled))
    } catch (e) {
      flip(!enabled)
      fail(t`Could not switch ${mod.name}`)(e)
    }
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
      fail(t`Could not remove ${mod.name}`)(e)
    }
    await get().load()
  },
  showFiles: async (mod) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      await ShowFiles(target.game, target.id, mod.uniqueId)
    } catch (e) {
      fail(t`Could not open the folder of ${mod.name}`)(e)
    }
  },
}))
