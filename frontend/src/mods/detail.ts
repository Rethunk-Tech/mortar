import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { Relations } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { Relations as ReadRelations } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type {
  Mod,
  ModState,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ModState as ReadModState,
  ResetConfig,
  RollBack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'
import { modId } from './lookup.ts'
import { useMods } from './store.ts'

interface Extras {
  id: string
  relations: Relations
  state: ModState
}

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: String(e) })
}

const open = () => {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

// The mod whose detail panel is open, by modId, with what the panel reads beyond the mod list.
export const useDetail = create<{
  detailId: string
  extras: Extras | null
  show: (mod: Mod | null) => void
  loadExtras: (mod: Mod) => Promise<void>
  rollBack: (mod: Mod) => Promise<void>
  resetConfig: (mod: Mod) => Promise<void>
}>((set, get) => ({
  detailId: '',
  extras: null,
  show: (mod) => set({ detailId: mod ? modId(mod) : '', extras: null }),
  loadExtras: async (mod) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      const [relations, state] = await Promise.all([
        ReadRelations(target.game, target.id, mod.key, mod.uniqueId),
        ReadModState(target.game, target.id, mod.key, mod.uniqueId),
      ])
      if (get().detailId === modId(mod)) {
        set({ extras: { id: modId(mod), relations, state } })
      }
    } catch (e) {
      fail(i18n._(msg`Could not read the details of ${mod.name}`))(e)
    }
  },
  rollBack: async (mod) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      useProfiles.getState().replace(await RollBack(target.game, target.id, mod.key))
    } catch (e) {
      fail(i18n._(msg`Could not roll back ${mod.name}`))(e)
      return
    }
    set({ detailId: '', extras: null })
    await useMods.getState().load()
  },
  resetConfig: async (mod) => {
    const target = open()
    if (!target) {
      return
    }
    try {
      await ResetConfig(target.game, target.id, mod.key, mod.uniqueId)
    } catch (e) {
      fail(i18n._(msg`Could not reset the settings of ${mod.name}`))(e)
    }
    await get().loadExtras(mod)
  },
}))
