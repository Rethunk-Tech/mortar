import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Mod } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Item } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import type { Want } from '../../queue/actions.ts'
import { pendingFor } from '../../queue/totals.ts'
import { installableUpdate, sameId } from '../lookup.ts'

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

export const downloadable = (u: Update) => installableUpdate(u)

export const installedCaution = (mods: Mod[], u: Update): string => {
  const mod = mods.find((m) => m.key === u.key && sameId(m.uniqueId, u.uniqueId))
  return mod?.updateCautionMessage?.trim() ?? ''
}

export const pendingUpdate = (items: Item[], profileId: string, u: Update) =>
  u.githubRepo
    ? pendingFor(items, profileId, 0, u.githubRepo)
    : pendingFor(items, profileId, u.nexusId)
