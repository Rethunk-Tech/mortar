type SegmentId = 'profiles' | 'store' | 'cache' | 'backups' | 'trash' | 'other'

interface Segment {
  id: SegmentId
  size: number
}

interface UsageTotals {
  profiles?: { size: number }[] | null
  store: number
  cache: number
  backups: number
  trash: number
  total: number
}

// Profile folders, store, cache, backups and trash do not overlap, so with "other" (everything else in the data
// folder) they add up to the total; per-game sizes cut across them and are shown separately.
function storageSegments(usage: UsageTotals): Segment[] {
  const profiles = (usage.profiles ?? []).reduce((n, p) => n + p.size, 0)
  const named: Segment[] = [
    { id: 'profiles', size: profiles },
    { id: 'store', size: usage.store },
    { id: 'cache', size: usage.cache },
    { id: 'backups', size: usage.backups },
    { id: 'trash', size: usage.trash },
  ]
  const known = named.reduce((n, s) => n + s.size, 0)
  return [...named, { id: 'other', size: Math.max(0, usage.total - known) }]
}

export { type SegmentId, storageSegments }
