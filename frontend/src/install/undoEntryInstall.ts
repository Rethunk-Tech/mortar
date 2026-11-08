import { msg } from '@lingui/core/macro'
import {
  RemoveEntry,
  RollBack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useMods } from '../mods/store.ts'
import { profileLocked } from '../mods/useLocked.ts'
import { useProfiles } from '../profiles/store.ts'
import { toastError } from '../toasts/report.ts'

export async function undoEntryInstall(
  game: string,
  profileId: string,
  entryKey: string,
  updated: boolean,
) {
  if (profileLocked(profileId)) {
    return
  }
  try {
    const next = updated
      ? await RollBack(game, profileId, entryKey)
      : await RemoveEntry(game, profileId, entryKey)
    useProfiles.getState().replace(next)
  } catch (e) {
    toastError(i18n._(msg`Could not undo the install`), e)
    return
  }
  await useMods.getState().load()
}
