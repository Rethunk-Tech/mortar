import { msg } from '@lingui/core/macro'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  OpenConfig,
  RemoveEntries,
  RemoveEntry,
  SetEntryCategoryMany,
  SetEntryNoteTags,
  SetEntryTagsMany,
  SetModEnabled,
  SetModsEnabled,
  SetPinned,
  SetPinnedMany,
  SetSkipSource,
  SetSkipVersion,
  SetSkipVersionMany,
  ShowFiles,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { modId } from './lookup.ts'
import { useSelection } from './selection.ts'
import { announceAlso, fail, open } from './storeView.ts'
import { useUpdates } from './updates.ts'

export async function enableMany(
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

export async function batchProfile(
  mods: Mod[],
  call: (game: string, id: string, keys: string[]) => Promise<Profile>,
  title: string,
) {
  const target = open()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(
      await call(
        target.game,
        target.id,
        mods.map((mod) => mod.key),
      ),
    )
  } catch (e) {
    fail(title)(e)
    return
  }
  await useUpdates.getState().load()
}

export async function dropMods(get: () => { load: () => Promise<void> }, mods: Mod[]) {
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

export async function dropMod(get: () => { load: () => Promise<void> }, mod: Mod) {
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

export async function setEntryNoteTags(mod: Mod, note: string, tags: string[]) {
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
}

export function setEnabledAction(
  set: (fn: (s: { mods: Mod[] }) => { mods: Mod[] }) => void,
  get: () => { loadProblems: () => Promise<void> },
  mod: Mod,
  enabled: boolean,
) {
  const target = open()
  if (!target) {
    return Promise.resolve()
  }
  const flip = (on: boolean) =>
    set((s) => ({
      mods: s.mods.map((m) => (modId(m) === modId(mod) ? { ...m, enabled: on } : m)),
    }))
  flip(enabled)
  return SetModEnabled(target.game, target.id, mod.key, mod.uniqueId, enabled).then(
    (got) => {
      useProfiles.getState().replace(got.profile)
      announceAlso(got.alsoEnabled)
      return get().loadProblems()
    },
    (e) => {
      flip(!enabled)
      fail(i18n._(msg`Could not switch ${mod.name}`))(e)
    },
  )
}

export async function pinMod(mod: Mod, pinned: boolean) {
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
}

export function pinMany(mods: Mod[], pinned: boolean) {
  return batchProfile(
    mods,
    (game, id, keys) => SetPinnedMany(game, id, keys, pinned),
    i18n._(pinned ? msg`Could not pin the selected mods` : msg`Could not unpin the selected mods`),
  )
}

export async function skipVersion(mod: Mod, version: string) {
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
}

export function skipVersionMany(mods: Mod[]) {
  return batchProfile(
    mods,
    (game, id, keys) =>
      SetSkipVersionMany(
        game,
        id,
        mods.map((mod, index) => ({ key: keys[index] ?? mod.key, version: mod.version })),
      ),
    i18n._(msg`Could not skip updates for the selected mods`),
  )
}

export async function skipSource(mod: Mod, source: string, skip: boolean) {
  const target = open()
  if (!target) {
    return
  }
  useProfiles.getState().replace(await SetSkipSource(target.game, target.id, mod.key, source, skip))
  await useUpdates.getState().load()
}

export function setCategoryMany(mods: Mod[], category: string) {
  return batchProfile(
    mods,
    (game, id, keys) => SetEntryCategoryMany(game, id, keys, category),
    i18n._(msg`Could not set the category for the selected mods`),
  )
}

export function setTagMany(mods: Mod[], tag: string, add: boolean) {
  return batchProfile(
    mods,
    (game, id, keys) => SetEntryTagsMany(game, id, keys, tag, add),
    i18n._(msg`Could not update tags for the selected mods`),
  )
}

export async function showModFiles(mod: Mod) {
  const target = open()
  if (!target) {
    return
  }
  try {
    await ShowFiles(target.game, target.id, mod.key, mod.uniqueId)
  } catch (e) {
    fail(i18n._(msg`Could not open the folder of ${mod.name}`))(e)
  }
}

export async function openModConfig(mod: Mod) {
  const target = open()
  if (!target) {
    return
  }
  try {
    await OpenConfig(target.game, target.id, mod.key, mod.uniqueId)
  } catch (e) {
    fail(i18n._(msg`Could not open config.json of ${mod.name}`))(e)
  }
}
