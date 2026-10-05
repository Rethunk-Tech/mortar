import { CopyMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { FromSave } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { useSaves } from './store.ts'
import { wantFor } from './wantFor.ts'

// Switches on the recorded mods this profile has but disabled; returns how many.
export async function enableRecordedMods(
  game: string,
  profileId: string,
  fit: Fit,
): Promise<number> {
  const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
  const off = ((fit.lastMissing?.length ? fit.lastMissing : fit.missing) ?? []).filter(
    (m) => m.disabled,
  )
  if (!profile) {
    return 0
  }
  for (const lack of off) {
    await useSaves.getState().enable(game, profile, lack.id)
  }
  return off.length
}

// Adds the recorded mods the profile lacks, and with `enable` also switches on the ones it has disabled.
export async function addRecordedMods(
  game: string,
  profileId: string,
  fit: Fit,
  enable = true,
): Promise<void> {
  const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
  if (!profile) {
    return
  }
  const missing = (fit.lastMissing?.length ? fit.lastMissing : fit.missing) ?? []
  const fromId = fit.lastProfileExists ? fit.lastProfileId : ''
  const copyIds = missing.filter((m) => !m.disabled).map((m) => m.id)
  if (fromId && fromId !== profileId && copyIds.length > 0) {
    useProfiles.getState().replace(await CopyMods(game, fromId, profileId, copyIds))
  }
  if (enable) {
    await enableRecordedMods(game, profileId, fit)
  }
  const wants = missing
    .filter((m) => !m.disabled)
    .map(wantFor)
    .filter((w) => w !== null)
  if (wants.length > 0) {
    await download(wants)
  }
}

export async function newProfileFromSave(game: string, folder: string): Promise<void> {
  const got = await FromSave(game, folder)
  await useProfiles.getState().refresh()
  useProfiles.getState().open(got.profile.id)
}
