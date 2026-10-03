const HISTORY_KINDS = new Set([
  'added',
  'removed',
  'updated',
  'enabled',
  'disabled',
  'pinned',
  'imported',
  'restored',
  'reverted',
  'bulk',
  'good',
])

/** "+3 −1 ~2" style summary of mod adds, removes, and updates for one history event. */
export function historyChangeSummary(ev: {
  added?: number | null
  removed?: number | null
  updated?: number | null
}): string {
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

export function historyEventKind(ev: { kind: string; label: string }): string {
  if (HISTORY_KINDS.has(ev.label)) {
    return ev.label
  }
  if (HISTORY_KINDS.has(ev.kind)) {
    return ev.kind
  }
  return ''
}
