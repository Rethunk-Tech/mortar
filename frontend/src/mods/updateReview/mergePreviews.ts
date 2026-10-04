import type { EverywherePreview } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export function mergePreviews(parts: EverywherePreview[]): EverywherePreview {
  const affected: EverywherePreview['affected'] = []
  const skipped: EverywherePreview['skipped'] = []
  const seen = new Set<string>()
  for (const part of parts) {
    for (const row of part.affected ?? []) {
      if (!seen.has(row.profileId)) {
        seen.add(row.profileId)
        affected.push(row)
      }
    }
    for (const row of part.skipped ?? []) {
      if (!seen.has(row.profileId)) {
        seen.add(row.profileId)
        skipped.push(row)
      }
    }
  }
  return { affected, skipped }
}
