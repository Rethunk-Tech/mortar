export interface RedundantItem {
  kind: string
  key: string
  id: string
  name: string
  by: { key: string; name: string }[] | null
  detail?: string
  covered?: boolean
}

// sameJobGroups joins every similar-job pair into the set of mods that all do that job, in first-seen order.
export function sameJobGroups(items: RedundantItem[]): { keys: string[]; detail: string }[] {
  const parent = new Map<string, string>()
  const find = (k: string): string => {
    const p = parent.get(k) ?? k
    if (p === k) {
      return k
    }
    const root = find(p)
    parent.set(k, root)
    return root
  }
  const order: string[] = []
  const detail = new Map<string, string>()
  for (const item of items) {
    for (const k of [item.key, ...(item.by ?? []).map((b) => b.key)]) {
      if (!parent.has(k)) {
        parent.set(k, k)
        order.push(k)
      }
    }
    for (const b of item.by ?? []) {
      parent.set(find(b.key), find(item.key))
    }
  }
  for (const item of items) {
    const root = find(item.key)
    if (!detail.has(root) && item.detail) {
      detail.set(root, item.detail)
    }
  }
  const groups = new Map<string, string[]>()
  for (const k of order) {
    const root = find(k)
    groups.set(root, [...(groups.get(root) ?? []), k])
  }
  return [...groups.entries()].map(([root, keys]) => ({ keys, detail: detail.get(root) ?? '' }))
}

// How many rows the Redundant tab lists: every item but a same-job one, plus one row per group of mods doing the same job.
export function redundantRowCount(items: RedundantItem[]): number {
  const alone = items.filter((item) => item.kind !== 'sameJob' || item.covered).length
  return (
    alone + sameJobGroups(items.filter((item) => item.kind === 'sameJob' && !item.covered)).length
  )
}
