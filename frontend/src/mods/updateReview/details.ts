import type { File } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Details } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'

/** Size in KB of the file that carries the new version, and the first line of its changelog; both only from details already loaded. */
export function rowDetails(update: Update, details: Details | undefined) {
  const files: File[] = details?.files ?? []
  const [file] = files
    .filter((f) => f.modVersion === update.version || f.version === update.version)
    .sort((a, b) => b.uploaded.localeCompare(a.uploaded))
  const log = (details?.changelogs ?? []).find((c) => c.version === update.version)
  const first = [log?.body ?? '', ...(log?.notes ?? [])]
    .flatMap((s) => s.split('\n'))
    .map((s) => s.trim())
    .find((s) => s !== '')
  return { sizeKb: file?.sizeKb ?? 0, changelog: first ?? '' }
}

/** The dependency names the update adds and drops; both empty when the source gave no dependency data. */
export function dependencyChanges(update: Update) {
  return { added: update.addedDeps ?? [], removed: update.removedDeps ?? [] }
}
