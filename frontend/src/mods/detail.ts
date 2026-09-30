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
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { modId, reshow } from './lookup.ts'
import { useMods } from './store.ts'

interface Extras {
  id: string
  relations: Relations
  state: ModState
}

const fail = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(e) })
}

const open = () => {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

async function redoUpdate(game: string, id: string, key: string) {
  try {
    useProfiles.getState().replace(await RollBack(game, id, key))
  } catch (e) {
    fail(i18n._(msg`Could not redo the update`))(e)
    return
  }
  await useMods.getState().load()
}

function keyAfterRollBack(
  entries: { key: string; mods?: { uniqueId: string }[] | null }[] | null | undefined,
  uniqueId: string,
  fallback: string,
) {
  return entries?.find((e) => (e.mods ?? []).some((m) => m.uniqueId === uniqueId))?.key ?? fallback
}

// The selected mod (by modId) shown in the sidebar, whether its details dialog is open, and what the dialog reads beyond the mod list.
export const useDetail = create<{
  detailId: string
  open: boolean
  extras: Extras | null
  show: (mod: Mod | null) => void
  setOpen: (isOpen: boolean) => void
  loadExtras: (mod: Mod) => Promise<void>
  rollBack: (mod: Mod) => Promise<void>
  resetConfig: (mod: Mod) => Promise<void>
}>((set, get) => ({
  detailId: '',
  open: false,
  extras: null,
  show: (mod) => set((s) => ({ ...reshow(s, mod), open: false })),
  setOpen: (isOpen) => set({ open: isOpen }),
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
      const rolled = await RollBack(target.game, target.id, mod.key)
      useProfiles.getState().replace(rolled)
    } catch (e) {
      fail(i18n._(msg`Could not roll back ${mod.name}`))(e)
      return
    }
    const key = keyAfterRollBack(
      useProfiles.getState().profiles.find((p) => p.id === target.id)?.entries,
      mod.uniqueId,
      mod.key,
    )
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`${mod.name} rolled back`),
      picture: mod.picture,
      action: {
        label: i18n._(msg`Redo update`),
        run: () => redoUpdate(target.game, target.id, key),
      },
    })
    set({ open: false, extras: null })
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

export { keyAfterRollBack }
