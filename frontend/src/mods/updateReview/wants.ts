import { msg } from '@lingui/core/macro'
import type { File } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Item } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { i18n } from '../../i18n/index.ts'
import type { Want } from '../../queue/actions.ts'
import { pendingFor } from '../../queue/totals.ts'
import { isFull, loadDetails, nexusEntryOf, useNexusDetails } from '../nexusDetails.ts'
import { optionalUpdateWants, useOptionalSkips } from '../optionalFiles.ts'

const updateSource = (u: Update): Partial<Want> => {
  if (u.package) {
    return { package: u.package, ...(u.packageSource ? { source: u.packageSource } : {}) }
  }
  if (u.githubRepo) {
    return { repo: u.githubRepo }
  }
  return {
    modId: u.nexusId,
    ...(u.fileId ? { fileId: u.fileId } : {}),
    ...(u.githubFallback ? { fallbackRepo: u.githubFallback, fallbackId: u.id } : {}),
  }
}

/** The queue request for an update. A Modrinth update names its exact version, since a version number can repeat
 * across loaders. A Nexus update of a mod with a GitHub repo tries that repo's release at the same
 * version first, since a free Nexus account must click for every file. */
export const updateWant = (u: Update): Want => ({
  kind: 'update',
  ...updateSource(u),
  name: u.name,
  version: u.packageVersion || u.version,
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
  game: string,
  u: Update,
  profile: Pick<Profile, 'entries'> | null | undefined,
): Promise<Want[]> {
  const skipped = useOptionalSkips.getState().skipped[u.key] === true
  const hasOptional = (profile?.entries ?? []).some((e) => e.overlayOf === u.key)
  if (!hasOptional || skipped || u.githubRepo) {
    return [updateWant(u)]
  }
  if (!isFull(nexusEntryOf(useNexusDetails.getState().byKey, game, u.nexusId))) {
    await loadDetails(game, u.nexusId)
  }
  const files =
    nexusEntryOf(useNexusDetails.getState().byKey, game, u.nexusId)?.details?.files ?? []
  return withOptional(u, profile, files, false)
}

// What to accept before the update applies: the installed manifest's caution, and the game's known-broken list
// flagging the new version.
export const installedCaution = (u: Update): string =>
  [
    u.cautionMessage?.trim() ?? '',
    u.knownBroken
      ? i18n._(
          msg`Mortar's known-broken list still flags this version: ${{ reason: u.knownBroken.reason }}`,
        )
      : '',
  ]
    .filter((part) => part !== '')
    .join(' ')

export const pendingUpdate = (items: Item[], profileId: string, u: Update) =>
  u.githubRepo
    ? pendingFor(items, profileId, 0, u.githubRepo)
    : pendingFor(items, profileId, u.nexusId)
