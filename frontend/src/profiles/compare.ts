import type {
  Entry,
  Profile,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { cmpText } from '../mods/cmpText.ts'
import { idKey, localId } from '../mods/dependents.ts'
import { sameId } from '../mods/lookup.ts'
import { userModEntries } from './count.ts'

interface CompareSide {
  id: string
  name: string
  version: string
  enabled: boolean
  key: string
  source: Source
}

interface ComparePair {
  id: string
  name: string
  a: CompareSide
  b: CompareSide
}

interface ProfileCompare {
  onlyA: CompareSide[]
  onlyB: CompareSide[]
  differentVersion: ComparePair[]
  differentEnabled: ComparePair[]
  differentSource: ComparePair[]
  identical: ComparePair[]
}

function enabledOf(entry: Entry, id: string): boolean {
  return !(entry.disabled ?? []).some((disabled) => sameId(disabled, id))
}

function indexUserMods(profile: Profile): Map<string, CompareSide> {
  const out = new Map<string, CompareSide>()
  for (const e of userModEntries(profile.entries)) {
    for (const m of e.mods ?? []) {
      const k = idKey(m.id)
      if (!out.has(k)) {
        out.set(k, {
          id: m.id,
          name: m.name,
          version: m.version,
          enabled: enabledOf(e, m.id),
          key: e.key,
          source: e.source,
        })
      }
    }
  }
  return out
}

function sortSides(sides: CompareSide[]): CompareSide[] {
  return sides.toSorted((a, b) => {
    const n = cmpText(a.name, b.name)
    if (n !== 0) {
      return n
    }
    return cmpText(a.id, b.id)
  })
}

function sortPairs(pairs: ComparePair[]): ComparePair[] {
  return pairs.toSorted((a, b) => {
    const n = cmpText(a.name, b.name)
    if (n !== 0) {
      return n
    }
    return cmpText(a.id, b.id)
  })
}

function pairName(a: CompareSide, b: CompareSide): string {
  return a.name || b.name
}

function classifyPair(
  side: CompareSide,
  other: CompareSide,
  buckets: {
    differentVersion: ComparePair[]
    differentEnabled: ComparePair[]
    differentSource: ComparePair[]
    identical: ComparePair[]
  },
) {
  const name = pairName(side, other)
  const pair: ComparePair = { id: side.id, name, a: side, b: other }
  const versionDiff = side.version !== other.version
  const enabledDiff = side.enabled !== other.enabled
  if (versionDiff) {
    buckets.differentVersion.push(pair)
  }
  if (enabledDiff) {
    buckets.differentEnabled.push(pair)
  }
  const sourceDiff = side.source.kind !== other.source.kind
  if (sourceDiff) {
    buckets.differentSource.push(pair)
  }
  if (!(versionDiff || enabledDiff || sourceDiff)) {
    buckets.identical.push(pair)
  }
}

/** Lists user mods across two profiles, grouped for the compare UI. */
function compareProfiles(a: Profile, b: Profile): ProfileCompare {
  const left = indexUserMods(a)
  const right = indexUserMods(b)
  const onlyA: CompareSide[] = []
  const onlyB: CompareSide[] = []
  const differentVersion: ComparePair[] = []
  const differentEnabled: ComparePair[] = []
  const differentSource: ComparePair[] = []
  const identical: ComparePair[] = []

  for (const [k, side] of left) {
    const other = right.get(k)
    if (other) {
      classifyPair(side, other, { differentVersion, differentEnabled, differentSource, identical })
    } else {
      onlyA.push(side)
    }
  }
  for (const [k, side] of right) {
    if (!left.has(k)) {
      onlyB.push(side)
    }
  }

  return {
    onlyA: sortSides(onlyA),
    onlyB: sortSides(onlyB),
    differentVersion: sortPairs(differentVersion),
    differentEnabled: sortPairs(differentEnabled),
    differentSource: sortPairs(differentSource),
    identical: sortPairs(identical),
  }
}

type GroupKind = 'version' | 'onlyB' | 'onlyA' | 'enabled' | 'source' | 'identical'

interface CompareRow {
  id: string
  name: string
  kind: GroupKind
  a: CompareSide | null
  b: CompareSide | null
}

interface CompareGroup {
  kind: GroupKind
  rows: CompareRow[]
}

interface CompareView {
  groups: CompareGroup[]
  /** Distinct mods that differ and exist in both profiles or only in B: what a bare compare is about. */
  differences: number
  onlyA: number
  identical: number
}

function matchesNeedle(name: string, id: string, needle: string): boolean {
  return (
    !needle || name.toLowerCase().includes(needle) || localId(id).toLowerCase().includes(needle)
  )
}

function pairRows(pairs: ComparePair[], kind: GroupKind): CompareRow[] {
  return pairs.map((p) => ({ id: p.id, name: p.name, kind, a: p.a, b: p.b }))
}

/**
 * Groups a compare for the dialog: differences first; mods only in A (the profile being edited from) and identical
 * mods appear only when `showAll`, because they are usually the long tail. Counts always cover the whole filtered set.
 */
function compareView(diff: ProfileCompare, needle: string, showAll: boolean): CompareView {
  const keep = <T extends { name: string; id: string }>(rows: T[]) =>
    rows.filter((r) => matchesNeedle(r.name, r.id, needle))
  const onlyA = keep(diff.onlyA)
  const onlyB = keep(diff.onlyB)
  const version = keep(diff.differentVersion)
  const enabled = keep(diff.differentEnabled)
  const source = keep(diff.differentSource)
  const identical = keep(diff.identical)
  const side = (s: CompareSide, kind: GroupKind): CompareRow => ({
    id: s.id,
    name: s.name,
    kind,
    a: kind === 'onlyA' ? s : null,
    b: kind === 'onlyB' ? s : null,
  })
  const all: CompareGroup[] = [
    { kind: 'version', rows: pairRows(version, 'version') },
    { kind: 'onlyB', rows: onlyB.map((s) => side(s, 'onlyB')) },
    { kind: 'onlyA', rows: showAll ? onlyA.map((s) => side(s, 'onlyA')) : [] },
    { kind: 'enabled', rows: pairRows(enabled, 'enabled') },
    { kind: 'source', rows: pairRows(source, 'source') },
    { kind: 'identical', rows: showAll ? pairRows(identical, 'identical') : [] },
  ]
  const groups = all.filter((g) => g.rows.length > 0)
  const differing = new Set([...version, ...enabled, ...source, ...onlyB].map((r) => r.id))
  return { groups, differences: differing.size, onlyA: onlyA.length, identical: identical.length }
}

interface ApplyPlan {
  copy: string[]
  moves: { oldKey: string; newKey: string }[]
}

/**
 * What "make the target match the source" does for the selected rows: mods only in the source and enabled-state
 * differences are copied, version and source differences switch the target's entry to the source's. Mods only in the
 * target stay, since there is no removal here.
 */
function applyPlan(rows: CompareRow[], toB: boolean): ApplyPlan {
  const copy = new Set<string>()
  const moves = new Map<string, string>()
  for (const row of rows) {
    const from = toB ? row.a : row.b
    const to = toB ? row.b : row.a
    const copies =
      row.kind === 'enabled' || (row.kind === 'onlyA' && toB) || (row.kind === 'onlyB' && !toB)
    if (from && copies) {
      copy.add(row.id)
    } else if (from && to && (row.kind === 'version' || row.kind === 'source')) {
      moves.set(to.key, from.key)
    }
  }
  return {
    copy: [...copy],
    moves: [...moves].map(([oldKey, newKey]) => ({ oldKey, newKey })),
  }
}

export type { CompareGroup, CompareRow, CompareSide, CompareView, GroupKind }
export { applyPlan, compareProfiles, compareView }
