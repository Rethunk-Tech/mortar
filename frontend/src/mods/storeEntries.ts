import type { MessageDescriptor } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  AddToGroup,
  DeleteGroup,
  OpenConfig,
  RemoveEntries,
  RemoveFromGroup,
  RenameGroup,
  RestoreEntries,
  RestoreEntryFields,
  SetEntryCategoryMany,
  SetEntryNoteTags,
  SetEntryTagsMany,
  SetGroupEnabled,
  SetModEnabled,
  SetModsEnabled,
  SetPinned,
  SetPinnedMany,
  SetSkipSource,
  SetSkipSourceMany,
  SetSkipVersion,
  SetSkipVersionMany,
  SetUpdateChannel,
  ShowFiles,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { changeStillLatest } from '../toasts/history.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  entriesForKeys,
  entryFieldsOf,
  fieldsStillUndoable,
  type UndoEntry,
} from '../toasts/undo.ts'
import { enableRequirementsDecision, pendingRequired } from './enableRequirements.ts'
import { considerEnableRequirements } from './enableRequirementsApply.ts'
import { modId } from './lookup.ts'
import { useSelection } from './selection.ts'
import { announceAlso, openTarget } from './storeView.ts'
import { useUpdates } from './updates.ts'

function pushFieldsUndo(
  profileId: string,
  fields: ReturnType<typeof entryFieldsOf>,
  title: string,
) {
  useToasts.getState().push({
    kind: 'success',
    title,
    action: {
      label: i18n._(msg`Undo`),
      profileId,
      run: async () => {
        const target = openTarget()
        if (!target) {
          return
        }
        useProfiles.getState().replace(await RestoreEntryFields(target.game, target.id, fields))
        await useUpdates.getState().load()
      },
      live: () => {
        const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
        if (!profile) {
          return { disabled: true, reason: i18n._(msg`That profile is gone.`) }
        }
        return fieldsStillUndoable((profile.entries ?? []) as UndoEntry[], fields)
          ? { disabled: false }
          : { disabled: true, reason: i18n._(msg`This is no longer the latest change.`) }
      },
    },
  })
}

function pushRemovedUndo(
  profileId: string,
  entries: NonNullable<Profile['entries']>,
  load: () => Promise<void>,
) {
  const ids = entries.flatMap((entry) => (entry.mods ?? []).map((mod) => mod.id))
  const [first] = entries
  useToasts.getState().push({
    kind: 'success',
    title:
      new Set(ids).size === 1 && first?.mods?.[0]?.name
        ? i18n._(msg`Removed ${first.mods[0].name}`)
        : i18n._(
            msg`${plural(new Set(ids).size, { one: 'Removed # mod', other: 'Removed # mods' })}`,
          ),
    action: {
      label: i18n._(msg`Undo`),
      profileId,
      run: async () => {
        const target = openTarget()
        if (!target) {
          return
        }
        useProfiles.getState().replace(await RestoreEntries(target.game, target.id, entries))
        await load()
      },
      live: () => {
        const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
        if (!(profile && first)) {
          return { disabled: true, reason: i18n._(msg`That profile is gone.`) }
        }
        const state = changeStillLatest(profile, first.key, ids)
        return state.disabled
          ? { disabled: false }
          : { disabled: true, reason: i18n._(msg`This is no longer the latest change.`) }
      },
    },
  })
}

async function batchProfile(
  mods: Mod[],
  call: (game: string, id: string, keys: string[]) => Promise<Profile>,
  title: string,
  done?: string,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  const keys = mods.map((mod) => mod.key)
  const profile = useProfiles.getState().profiles.find((p) => p.id === target.id)
  const fields = entryFieldsOf((profile?.entries ?? []) as UndoEntry[], keys)
  try {
    useProfiles.getState().replace(await call(target.game, target.id, keys))
  } catch (e) {
    reportError(title)(e)
    return
  }
  if (done !== undefined) {
    pushFieldsUndo(target.id, fields, done)
  }
  await useUpdates.getState().load()
}

// Enables the required mods per the preference, or asks about them.
async function afterEnable(
  set: (fn: (s: { mods: Mod[] }) => { mods: Mod[] }) => void,
  get: () => { mods: Mod[] },
  pending: Mod[],
  enabling: Mod[],
) {
  const decision = enableRequirementsDecision(
    gamePrefs(useSettings.getState()).enableRequirements || 'always',
    pending.length,
  )
  if (decision === 'enable') {
    const extra = new Set(pending.map((m) => modId(m)))
    set((s) => ({ mods: s.mods.map((m) => (extra.has(modId(m)) ? { ...m, enabled: true } : m)) }))
  } else {
    await considerEnableRequirements(get().mods, enabling, 'toggle')
  }
}

async function groupCall(
  failure: MessageDescriptor,
  call: (game: string, id: string) => Promise<Profile>,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(await call(target.game, target.id))
  } catch (e) {
    reportError(i18n._(failure))(e)
  }
}

const removeModFromGroup = (key: string, name: string) =>
  groupCall(msg`Could not remove from the group`, (game, id) =>
    RemoveFromGroup(game, id, name, key),
  )

const renameGroup = (name: string, next: string) =>
  groupCall(msg`Could not rename the group`, (game, id) => RenameGroup(game, id, name, next))

const deleteGroup = (name: string) =>
  groupCall(msg`Could not delete the group`, (game, id) => DeleteGroup(game, id, name))

export async function enableMany(
  set: (fn: (s: { mods: Mod[] }) => { mods: Mod[] }) => void,
  get: () => { mods: Mod[]; loadProblems: () => Promise<void> },
  mods: Mod[],
  enabled: boolean,
) {
  const target = openTarget()
  if (!target || mods.length === 0) {
    return
  }
  const ids = new Set(mods.map((m) => modId(m)))
  const prev = new Map(get().mods.map((m) => [modId(m), m.enabled]))
  const pending = enabled ? pendingRequired(get().mods, mods) : []
  set((s) => ({
    mods: s.mods.map((m) => (ids.has(modId(m)) ? { ...m, enabled } : m)),
  }))
  try {
    useProfiles.getState().replace(
      await SetModsEnabled(
        target.game,
        target.id,
        mods.map((m) => ({ key: m.key, id: m.id })),
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
    reportError(i18n._(msg`Could not switch the selected mods`))(e)
    return
  }
  if (enabled) {
    await afterEnable(set, get, pending, mods)
  }
  await get().loadProblems()
}

export async function dropMods(
  get: () => { load: () => Promise<void> },
  mods: Mod[],
  failure: MessageDescriptor = msg`Could not remove the selected mods`,
) {
  const target = openTarget()
  if (!target || mods.length === 0) {
    return
  }
  const keys = [...new Set(mods.map((m) => m.key))]
  const profile = useProfiles.getState().profiles.find((p) => p.id === target.id)
  const removed = entriesForKeys(profile?.entries ?? [], keys)
  try {
    useProfiles.getState().replace(await RemoveEntries(target.game, target.id, keys))
    pushRemovedUndo(target.id, removed, get().load)
  } catch (e) {
    reportError(i18n._(failure))(e)
  }
  await get().load()
  useSelection.getState().clear()
}

export const dropMod = (get: () => { load: () => Promise<void> }, mod: Mod) =>
  dropMods(get, [mod], msg`Could not remove ${mod.name}`)

export async function setEntryNoteTags(mod: Mod, note: string, tags: string[]) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles
      .getState()
      .replace(await SetEntryNoteTags(target.game, target.id, mod.key, note, tags))
  } catch (e) {
    reportError(i18n._(msg`Could not save the note and tags for ${mod.name}`))(e)
  }
}

export async function setEnabledAction(
  set: (fn: (s: { mods: Mod[] }) => { mods: Mod[] }) => void,
  get: () => { mods: Mod[]; loadProblems: () => Promise<void> },
  mod: Mod,
  enabled: boolean,
) {
  const target = openTarget()
  if (!target) {
    return
  }
  const pending = enabled ? pendingRequired(get().mods, [mod]) : []
  const flip = (on: boolean) =>
    set((s) => ({
      mods: s.mods.map((m) => (modId(m) === modId(mod) ? { ...m, enabled: on } : m)),
    }))
  flip(enabled)
  try {
    const got = await SetModEnabled(target.game, target.id, mod.key, mod.id, enabled)
    useProfiles.getState().replace(got.profile)
    announceAlso(got.alsoEnabled)
    if (enabled) {
      await afterEnable(set, get, pending, [mod])
    }
    await get().loadProblems()
  } catch (e) {
    flip(!enabled)
    reportError(i18n._(msg`Could not switch ${mod.name}`))(e)
  }
}

export async function pinMod(mod: Mod, pinned: boolean, pinReason = '') {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles
      .getState()
      .replace(await SetPinned(target.game, target.id, mod.key, pinned, pinReason))
  } catch (e) {
    reportError(i18n._(pinned ? msg`Could not pin ${mod.name}` : msg`Could not unpin ${mod.name}`))(
      e,
    )
    return
  }
  await useUpdates.getState().load()
}

export function pinMany(mods: Mod[], pinned: boolean, pinReason = '') {
  return batchProfile(
    mods,
    (game, id, keys) => SetPinnedMany(game, id, keys, pinned, pinReason),
    i18n._(pinned ? msg`Could not pin the selected mods` : msg`Could not unpin the selected mods`),
    i18n._(pinned ? msg`Pinned selected mods` : msg`Unpinned selected mods`),
  )
}

export async function skipVersion(mod: Mod, version: string) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(await SetSkipVersion(target.game, target.id, mod.key, version))
  } catch (e) {
    reportError(
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

export async function setUpdateChannel(mod: Mod, channel: string) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(await SetUpdateChannel(target.game, target.id, mod.key, channel))
  } catch (e) {
    reportError(i18n._(msg`Could not set the update channel for ${mod.name}`))(e)
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
    i18n._(msg`Skipped updates for selected mods`),
  )
}

export async function skipSource(mod: Mod, source: string, skip: boolean) {
  const target = openTarget()
  if (!target) {
    return
  }
  useProfiles.getState().replace(await SetSkipSource(target.game, target.id, mod.key, source, skip))
  await useUpdates.getState().load()
}

export async function skipSourceMany(groups: ReadonlyMap<string, Mod[]>, skip: boolean) {
  for (const [source, mods] of groups) {
    await batchProfile(
      mods,
      (game, id, keys) => SetSkipSourceMany(game, id, keys, source, skip),
      i18n._(
        skip
          ? msg`Could not ignore updates from ${source}`
          : msg`Could not stop ignoring ${source} updates`,
      ),
    )
  }
}

export function setCategoryMany(mods: Mod[], category: string) {
  return batchProfile(
    mods,
    (game, id, keys) => SetEntryCategoryMany(game, id, keys, category),
    i18n._(msg`Could not set the category for the selected mods`),
    i18n._(msg`Updated categories`),
  )
}

export function setTagMany(mods: Mod[], tag: string, add: boolean) {
  return batchProfile(
    mods,
    (game, id, keys) => SetEntryTagsMany(game, id, keys, tag, add),
    i18n._(msg`Could not update tags for the selected mods`),
    i18n._(msg`Updated tags`),
  )
}

export async function showModFiles(mod: Mod) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await ShowFiles(target.game, target.id, mod.key, mod.id)
  } catch (e) {
    reportError(i18n._(msg`Could not open the folder of ${mod.name}`))(e)
  }
}

export async function openModConfig(mod: Mod) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    await OpenConfig(target.game, target.id, mod.key, mod.id)
  } catch (e) {
    reportError(i18n._(msg`Could not open config.json of ${mod.name}`))(e)
  }
}

export async function addModToGroup(key: string, name: string) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(await AddToGroup(target.game, target.id, name, key))
  } catch (e) {
    reportError(i18n._(msg`Could not add to the group`))(e)
  }
}

export async function setGroupEnabled(name: string, on: boolean) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    useProfiles.getState().replace(await SetGroupEnabled(target.game, target.id, name, on))
  } catch (e) {
    reportError(i18n._(msg`Could not switch the group`))(e)
  }
}

export { deleteGroup, removeModFromGroup, renameGroup }
