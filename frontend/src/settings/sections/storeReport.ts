import type {
  Item as LeftoverItem,
  Preview,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import type {
  Item,
  Report as StoreReport,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/store/models.ts'

const SELECT_SEP = '\u0000'

export interface Sel {
  id: string
  game: string
  item: Item
}

type LeftoverKind = 'nexus' | 'github' | 'cache' | 'temp'

export interface LeftoverGroup {
  kind: LeftoverKind
  items: LeftoverItem[]
  size: number
}

function newestOf(group: Item[]): Item {
  return group.reduce((best, cur) =>
    Date.parse(String(cur.lastUsed)) > Date.parse(String(best.lastUsed)) ? cur : best,
  )
}

// Removable store items: everything no profile uses, plus every copy of a duplicate except those a profile uses (the newest when none is).
export function removable(rep: StoreReport): { unused: Sel[]; older: Sel[] } {
  const unused: Sel[] = []
  const older: Sel[] = []
  for (const [game, g] of Object.entries(rep ?? {})) {
    for (const item of g?.unused ?? []) {
      unused.push({ id: `${game}${SELECT_SEP}${item.key}`, game, item })
    }
    for (const items of (g?.duplicates ?? []).filter(
      (d): d is Item[] => d !== null && d.length > 1,
    )) {
      const used = items.filter((i) => i.inUse)
      const keep = used.length > 0 ? used : [newestOf(items)]
      for (const item of items.filter((i) => !keep.includes(i))) {
        older.push({ id: `${game}${SELECT_SEP}${item.key}`, game, item })
      }
    }
  }
  return { unused, older }
}

function leftoverKind(it: LeftoverItem): LeftoverKind {
  if (it.kind === 'temp') {
    return 'temp'
  }
  if (it.rel.startsWith('cache/nexus/')) {
    return 'nexus'
  }
  return it.rel.startsWith('cache/github-') ? 'github' : 'cache'
}

export function groupLeftovers(preview: Preview | null): LeftoverGroup[] {
  const groups = new Map<LeftoverKind, LeftoverGroup>()
  for (const it of (preview?.items ?? []).filter((x) => x.kind !== 'store')) {
    const kind = leftoverKind(it)
    const g = groups.get(kind) ?? { kind, items: [], size: 0 }
    g.items.push(it)
    g.size += it.size
    groups.set(kind, g)
  }
  return [...groups.values()].toSorted((a, b) => b.size - a.size)
}
