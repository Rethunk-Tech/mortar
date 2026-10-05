import type { File } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Item } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import type { Want } from '../../queue/actions.ts'
import { pendingFor } from '../../queue/totals.ts'
import { sameId } from '../lookup.ts'
import { isFull, loadDetails, useNexusDetails } from '../nexusDetails.ts'
import { optionalUpdateWants, useOptionalSkips } from '../optionalFiles.ts'

/** The queue request for an update. A Nexus update of a mod with a GitHub repo tries that repo's release at the same
 * version first, since a free Nexus account must click for every file. */
export const updateWant = (u: Update): Want => ({
  kind: 'update',
  ...(u.package
    ? { package: u.package }
    : u.githubRepo
      ? { repo: u.githubRepo }
      : {
          modId: u.nexusId,
          ...(u.githubFallback ? { fallbackRepo: u.githubFallback, fallbackId: u.id } : {}),
        }),
  name: u.name,
  version: u.version,
  currentKey: u.key,
})

/** The update's request, then newer versions of the optional files laid over the entry unless they are left out;
 * the queue installs those after the main file. A GitHub update leaves them, since they wait on the Nexus file. */
export const withOptional = (
  u: Update,
  profile: Pick<Profile, 'entries'> | null | undefined,
  files: readonly File[],
  skipped: boolean,
): Want[] => [
  updateWant(u),
  ...(skipped || u.githubRepo ? [] : optionalUpdateWants(profile, u, files)),
]

/** withOptional for an update that no review row decided, such as auto-update before Play or a run error's Update:
 * reads the mod's Nexus files only when the profile holds optional files laid over the entry. */
export async function withOptionalLoaded(
  u: Update,
  profile: Pick<Profile, 'entries'> | null | undefined,
): Promise<Want[]> {
  const skipped = useOptionalSkips.getState().skipped[u.key] === true
  const hasOptional = (profile?.entries ?? []).some((e) => e.overlayOf === u.key)
  if (!hasOptional || skipped || u.githubRepo) {
    return [updateWant(u)]
  }
  if (!isFull(useNexusDetails.getState().byId[u.nexusId])) {
    await loadDetails(u.nexusId)
  }
  const files = useNexusDetails.getState().byId[u.nexusId]?.details?.files ?? []
  return withOptional(u, profile, files, false)
}

export const installedCaution = (mods: Mod[], u: Update): string => {
  const mod = mods.find((m) => m.key === u.key && sameId(m.id, u.id))
  return mod?.updateCautionMessage?.trim() ?? ''
}

export const pendingUpdate = (items: Item[], profileId: string, u: Update) =>
  u.githubRepo
    ? pendingFor(items, profileId, 0, u.githubRepo)
    : pendingFor(items, profileId, u.nexusId)
