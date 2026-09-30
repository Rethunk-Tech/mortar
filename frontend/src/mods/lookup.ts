import type {
  Broken,
  Copy,
  Duplicate,
  Missing,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

export const siblingsOf = (mods: Mod[], mod: Mod) =>
  mods.filter((m) => m.key === mod.key && m.uniqueId !== mod.uniqueId)

export const sourceKind = (profile: Profile, mod: Mod) =>
  (profile.entries ?? []).find((e) => e.key === mod.key)?.source.kind ?? ''

// A mod is told apart by its entry too, since two entries can hold the same UniqueID.
export const modId = (mod: Pick<Mod, 'key' | 'uniqueId'>) => `${mod.key}/${mod.uniqueId}`

export type Problem =
  | { kind: 'broken'; broken: Broken }
  | { kind: 'missing'; missing: Missing }
  | { kind: 'duplicate'; duplicate: Duplicate }

export const problemsOf = (result: Result | null): Problem[] =>
  result
    ? [
        ...(result.duplicates ?? []).map(
          (duplicate): Problem => ({ kind: 'duplicate', duplicate }),
        ),
        ...(result.broken ?? []).map((broken): Problem => ({ kind: 'broken', broken })),
        ...(result.missing ?? []).map((missing): Problem => ({ kind: 'missing', missing })),
      ]
    : []

export const sameId = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

// A card is flagged for a broken mod, for each copy of a duplicate, and for the dependent of a missing dependency.
export function concerns(p: Problem, mod: Mod): boolean {
  if (p.kind === 'broken') {
    return p.broken.key === mod.key && sameId(p.broken.uniqueId, mod.uniqueId)
  }
  if (p.kind === 'duplicate') {
    return (
      sameId(p.duplicate.uniqueId, mod.uniqueId) &&
      (p.duplicate.copies ?? []).some((c) => c.key === mod.key)
    )
  }
  return sameId(p.missing.dependentId, mod.uniqueId)
}

// The copy to keep unless the user picks another: the newest, and among equals the Nexus-sourced one.
export function preselect(copies: Copy[]): string {
  const newest = copies.filter((c) => c.newest)
  const pool = newest.length > 0 ? newest : copies
  return (pool.find((c) => c.nexus) ?? pool[0])?.key ?? ''
}
