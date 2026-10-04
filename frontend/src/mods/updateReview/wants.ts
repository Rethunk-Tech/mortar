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
import { optionalUpdateWants } from '../optionalFiles.ts'

/** The queue request for an update. A Nexus update of a mod with a GitHub repo tries that repo's release at the same
 * version first, since a free Nexus account must click for every file. */
export const updateWant = (u: Update): Want => ({
  kind: 'update',
  ...(u.githubRepo
    ? { repo: u.githubRepo }
    : {
        modId: u.nexusId,
        ...(u.githubFallback ? { fallbackRepo: u.githubFallback, fallbackId: u.uniqueId } : {}),
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

export const installedCaution = (mods: Mod[], u: Update): string => {
  const mod = mods.find((m) => m.key === u.key && sameId(m.uniqueId, u.uniqueId))
  return mod?.updateCautionMessage?.trim() ?? ''
}

export const pendingUpdate = (items: Item[], profileId: string, u: Update) =>
  u.githubRepo
    ? pendingFor(items, profileId, 0, u.githubRepo)
    : pendingFor(items, profileId, u.nexusId)
