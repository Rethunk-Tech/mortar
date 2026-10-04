import { msg } from '@lingui/core/macro'
import type {
  CollectionRef,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ClearCollection,
  SetCollection,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useProfiles } from './store.ts'

async function restoreCollection(game: string, profileId: string, ref: CollectionRef) {
  try {
    useProfiles.getState().replace(await SetCollection(game, profileId, ref))
  } catch (e) {
    reportError(i18n._(msg`Could not restore the collection link`))(e)
  }
}

export async function unlinkCollection(game: string, profile: Profile) {
  const previous = profile.collection
  if (!previous) {
    return
  }
  try {
    const next = await ClearCollection(game, profile.id)
    useProfiles.getState().replace(next)
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`Unlinked collection`),
      action: {
        label: i18n._(msg`Undo`),
        run: () => restoreCollection(game, profile.id, previous),
      },
    })
  } catch (e) {
    reportError(i18n._(msg`Could not unlink the collection`))(e)
  }
}
