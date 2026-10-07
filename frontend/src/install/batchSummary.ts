import { msg, plural } from '@lingui/core/macro'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  RemoveEntries,
  RollBack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useMods } from '../mods/store.ts'
import { profileLocked } from '../mods/useLocked.ts'
import { useProfiles } from '../profiles/store.ts'
import { changeStillLatest } from '../toasts/history.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

async function undoBatchInstall(game: string, profileId: string, landed: Landed[]) {
  if (profileLocked(profileId)) {
    return
  }
  try {
    let next = await RemoveEntries(
      game,
      profileId,
      landed.filter((l) => !l.updated).map((l) => l.key),
    )
    for (const done of landed.filter((l) => l.updated)) {
      next = await RollBack(game, profileId, done.key)
    }
    useProfiles.getState().replace(next)
  } catch (e) {
    toastError(i18n._(msg`Could not undo the install`), e)
    return
  }
  await useMods.getState().load()
}

// One mod of a multi-item install that landed; the batch reports them in a single toast.
export interface Landed {
  key: string
  updated: boolean
  mods: string[]
}

export function pushBatchSummary(game: string, profile: Profile, landed: Landed[], failed: number) {
  if (landed.length === 0) {
    return
  }
  const names = landed.flatMap((l) => l.mods)
  useToasts.getState().push({
    kind: failed > 0 ? 'warning' : 'success',
    title: plural(names.length, {
      one: `Added # mod to ${profile.name}`,
      other: `Added # mods to ${profile.name}`,
    }),
    ...(failed > 0 ? { body: i18n._(msg`${failed} could not be added`) } : {}),
    detail: names.join('\n'),
    action: {
      label: i18n._(msg`Undo all`),
      run: () => undoBatchInstall(game, profile.id, landed),
      profileId: profile.id,
      live: () => {
        const current = useProfiles.getState().profiles.find((p) => p.id === profile.id)
        for (const l of landed) {
          const state = changeStillLatest(current, l.key, l.mods)
          if (state.disabled) {
            return state
          }
        }
        return { disabled: false }
      },
    },
  })
}
