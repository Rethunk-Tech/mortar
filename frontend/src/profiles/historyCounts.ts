import type { HistoryEvent } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

/** "+3 −1 ~2" style summary of mod adds, removes, and updates for one history event. */
export function historyChangeSummary(ev: HistoryEvent): string {
  const parts: string[] = []
  const added = ev.added ?? 0
  const removed = ev.removed ?? 0
  const updated = ev.updated ?? 0
  if (added > 0) {
    parts.push(`+${added}`)
  }
  if (removed > 0) {
    parts.push(`−${removed}`)
  }
  if (updated > 0) {
    parts.push(`~${updated}`)
  }
  return parts.join(' ')
}
