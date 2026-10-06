import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { ConfigFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import { Files as ReadConfigFiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/service.ts'
import type { Relations } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { Relations as ReadRelations } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type {
  Mod,
  ModState,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ModState as ReadModState,
  ResetConfig,
  RollBack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { changeStillLatest } from '../toasts/history.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { modId, reshow } from './lookup.ts'
import { useMods } from './store.ts'
import { openTarget } from './storeView.ts'

interface Extras {
  id: string
  relations: Relations
  state: ModState
  // What the config editor would open for the mod: config.json, its in-game menu or its plugins' .cfg files.
  configFiles: ConfigFile[]
}

const fail = reportError

async function redoUpdate(game: string, id: string, key: string) {
  try {
    useProfiles.getState().replace(await RollBack(game, id, key))
  } catch (e) {
    fail(i18n._(msg`Could not redo the update`))(e)
    return
  }
  await useMods.getState().load()
}

function versionAfterRollBack(
  entries: { mods?: { id: string; version: string }[] | null }[] | null | undefined,
  id: string,
): string | undefined {
  return entries?.flatMap((e) => e.mods ?? []).find((m) => m.id === id)?.version
}

function keyAfterRollBack(
  entries: { key: string; mods?: { id: string }[] | null }[] | null | undefined,
  id: string,
  fallback: string,
) {
  return entries?.find((e) => (e.mods ?? []).some((m) => m.id === id))?.key ?? fallback
}

// Arrow keys move the selection, so the panel follows the row that took focus.
const showModId = (id: string) =>
  useDetail.getState().show(useMods.getState().mods.find((m) => modId(m) === id) ?? null)

// The selected mod (by modId) shown in the sidebar, whether its details dialog is open, and what the dialog reads beyond the mod list.
export const useDetail = create<{
  detailId: string
  pendingId: string
  open: boolean
  extras: Extras | null
  show: (mod: Mod | null) => void
  showAfterLoad: (mod: Pick<Mod, 'key' | 'id'>) => void
  takePending: () => string
  setOpen: (isOpen: boolean) => void
  loadExtras: (mod: Mod) => Promise<void>
  rollBack: (mod: Mod) => Promise<void>
  resetConfig: (mod: Mod) => Promise<void>
}>((set, get) => ({
  detailId: '',
  pendingId: '',
  open: false,
  extras: null,
  show: (mod) => set((s) => ({ ...reshow(s, mod), open: false })),
  showAfterLoad: (mod) => set({ pendingId: modId(mod) }),
  takePending: () => {
    const id = get().pendingId
    set({ pendingId: '' })
    return id
  },
  setOpen: (isOpen) => set({ open: isOpen }),
  loadExtras: async (mod) => {
    const target = openTarget()
    if (!target) {
      return
    }
    try {
      const [relations, state, configFiles] = await Promise.all([
        ReadRelations(target.game, target.id, mod.key, mod.id),
        ReadModState(target.game, target.id, mod.key, mod.id),
        ReadConfigFiles(target.game, target.id, mod.id),
      ])
      if (get().detailId === modId(mod)) {
        set({ extras: { id: modId(mod), relations, state, configFiles: configFiles ?? [] } })
      }
    } catch (e) {
      fail(i18n._(msg`Could not read the details of ${mod.name}`))(e)
    }
  },
  rollBack: async (mod) => {
    const target = openTarget()
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
    const next = useProfiles.getState().profiles.find((p) => p.id === target.id)
    const backTo = versionAfterRollBack(next?.entries, mod.id)
    const key = keyAfterRollBack(next?.entries, mod.id, mod.key)
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`${mod.name} rolled back`),
      ...(backTo ? { body: i18n._(msg`Back to ${backTo}. Saves untouched.`) } : {}),
      picture: mod.picture,
      action: {
        label: i18n._(msg`Redo update`),
        run: () => redoUpdate(target.game, target.id, key),
        profileId: target.id,
        live: () =>
          changeStillLatest(
            useProfiles.getState().profiles.find((p) => p.id === target.id),
            key,
            [mod.id],
          ),
      },
    })
    set({ open: false, extras: null })
    await useMods.getState().load()
  },
  resetConfig: async (mod) => {
    const target = openTarget()
    if (!target) {
      return
    }
    try {
      await ResetConfig(target.game, target.id, mod.key, mod.id)
    } catch (e) {
      fail(i18n._(msg`Could not reset the settings of ${mod.name}`))(e)
    }
    await get().loadExtras(mod)
  },
}))

export { keyAfterRollBack, showModId, versionAfterRollBack }
