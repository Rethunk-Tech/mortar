import { CopyMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { FromSave } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { useSaves } from './store.ts'
import { wantFor } from './wantFor.ts'

export async function addRecordedMods(game: string, profileId: string, fit: Fit): Promise<void> {
  const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
  if (!profile) {
    return
  }
  const missing = (fit.lastMissing?.length ? fit.lastMissing : fit.missing) ?? []
  const { enable } = useSaves.getState()
  const fromId = fit.lastProfileExists ? fit.lastProfileId : ''
  const copyIds = missing.filter((m) => !m.disabled).map((m) => m.uniqueId)
  if (fromId && fromId !== profileId && copyIds.length > 0) {
    useProfiles.getState().replace(await CopyMods(game, fromId, profileId, copyIds))
  }
  for (const lack of missing) {
    if (lack.disabled) {
      await enable(game, profile, lack.uniqueId)
    }
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
