import type { Changelog } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'
import { CachedDetails } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { isNewer } from './nexusFormat.ts'

// Versions newer than installed and not newer than latest (latest included, installed excluded).
export const changelogsBetween = (
  logs: Changelog[] | null | undefined,
  installed: string,
  latest: string,
): Changelog[] =>
  (logs ?? []).filter((c) => isNewer(c.version, installed) && !isNewer(c.version, latest))

// Fills the details store from the on-disk Nexus cache only, so Update review never hits the network.
export async function mergeCachedDetails(ids: number[]): Promise<void> {
  const want = [...new Set(ids)].filter((id) => id > 0)
  const { byId } = useNexusDetails.getState()
  const unknown = want.filter((id) => !byId[id]?.details)
  if (unknown.length === 0) {
    return
  }
  const cached = (await CachedDetails(unknown)) ?? {}
  useNexusDetails.setState((s) => {
    const next = { ...s.byId }
    for (const id of unknown) {
      const details = cached[`${id}`]
      if (details && !next[id]?.details) {
        next[id] = { details }
      }
    }
    return { byId: next }
  })
}
