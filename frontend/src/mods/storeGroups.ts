import type { MessageDescriptor } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  AddToGroup,
  DeleteGroup,
  RemoveFromGroup,
  RenameGroup,
  SetGroupEnabled,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError } from '../toasts/report.ts'
import { openTarget } from './storeView.ts'

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

export const removeModFromGroup = (key: string, name: string) =>
  groupCall(msg`Could not remove from the group`, (game, id) =>
    RemoveFromGroup(game, id, name, key),
  )

export const renameGroup = (name: string, next: string) =>
  groupCall(msg`Could not rename the group`, (game, id) => RenameGroup(game, id, name, next))

export const deleteGroup = (name: string) =>
  groupCall(msg`Could not delete the group`, (game, id) => DeleteGroup(game, id, name))

// Puts the entries in the named group, creating it when the profile has none by that name.
export async function addModsToGroup(keys: string[], name: string) {
  const target = openTarget()
  if (!target) {
    return
  }
  try {
    let latest: Profile | undefined
    for (const key of keys) {
      latest = await AddToGroup(target.game, target.id, name, key)
    }
    if (latest) {
      useProfiles.getState().replace(latest)
    }
  } catch (e) {
    reportError(i18n._(msg`Could not make the group`))(e)
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
