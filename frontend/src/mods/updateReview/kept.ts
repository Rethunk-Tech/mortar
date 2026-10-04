import type {
  Update,
  UpdatesResult,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { entryOf } from '../lookup.ts'

// Updates the profile keeps back because the mod is pinned, one per mod.
function keptUpdates(result: UpdatesResult | null, profile: Profile): Update[] {
  const seen = new Set<string>()
  return (result?.updates ?? []).filter((u) => {
    if (!entryOf(profile, u.key)?.pinned || seen.has(u.key)) {
      return false
    }
    seen.add(u.key)
    return true
  })
}

export { keptUpdates }
