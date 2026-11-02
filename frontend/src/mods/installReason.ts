import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { idKey, requiredIdsOf } from './dependents.ts'

type PartMod = Pick<Mod, 'key' | 'id' | 'needs' | 'optional' | 'contentPackFor'> & { name?: string }

export const partKey = (m: Pick<Mod, 'key' | 'id'>) => `${m.key}\n${idKey(m.id)}`

const framework = (m: PartMod) => idKey(m.contentPackFor ?? '')

// The main mod's score: each part requiring it outweighs both tie-breaks (a pack for the primary framework, then a
// mod that is no pack), which cannot both hold.
const BY_DEPENDENTS = 4
const BY_PRIMARY = 2
const BY_CODE = 1

// The words a download's parts are tagged with, so "[CP] Luck Skill" and "Luck Skill" name the same mod.
const PART_TAG =
  /\[[^\]]*\]|\([^)]*\)|(^|[\s\-_:])(cp|c#|smapi|code|content patcher|component|core|dll)(?=$|[\s\-_:])/gi
const NOT_ALNUM = /[^a-z0-9]/gi
const letters = (name: string) => name.replace(NOT_ALNUM, '').toLowerCase()
const bare = (name: string) => letters(name.replace(PART_TAG, ' '))

// The framework most content packs in the list are written for (Content Patcher in Stardew), or none.
function primaryFramework(mods: readonly PartMod[]): string {
  const counts = new Map<string, number>()
  for (const m of mods) {
    if (framework(m) !== '') {
      counts.set(framework(m), (counts.get(framework(m)) ?? 0) + 1)
    }
  }
  return [...counts].toSorted((a, b) => b[1] - a[1])[0]?.[0] ?? ''
}

/**
 * The mods that came in a download beside its main mod, which read implicit whatever the download's reason; the main
 * mod takes the download's. The main mod is the base the other parts build on: one that requires no other mod of the
 * download, the one most of them require (Love of Cooking under its [CP] and [EN] packs). Where nothing in the download
 * requires another, a pack for the primary framework wins (a town's Content Patcher pack beside its Farm Type Manager
 * and Custom Companions packs), else a mod that is no content pack, else the first.
 */
export function supportParts(
  mods: readonly PartMod[],
  all: readonly PartMod[] = mods,
): Set<string> {
  const primary = primaryFramework(all)
  const byKey = Map.groupBy(mods, (m) => m.key)
  const needs = (a: PartMod, b: PartMod) => requiredIdsOf(a).some((id) => idKey(id) === idKey(b.id))
  const out = new Set<string>()
  for (const parts of byKey.values()) {
    if (parts.length < 2) {
      continue
    }
    const bases = parts.filter((b) => !parts.some((a) => a !== b && needs(b, a) && !needs(a, b)))
    const dependents = (b: PartMod) => parts.filter((a) => a !== b && needs(a, b)).length
    const score = (m: PartMod) =>
      dependents(m) * BY_DEPENDENTS +
      (primary !== '' && framework(m) === primary ? BY_PRIMARY : 0) +
      (framework(m) === '' ? BY_CODE : 0)
    const base = (bases.length > 0 ? bases : parts).reduce((best, m) =>
      score(m) > score(best) ? m : best,
    )
    // A tagged base ("[CP] Luck Skill") gives way to its untagged twin, the part named as the mod is.
    const name = bare(base.name ?? '')
    const untagged = (m: PartMod) => bare(m.name ?? '') === letters(m.name ?? '')
    const twin = parts.find(
      (m) => m !== base && name !== '' && bare(m.name ?? '') === name && untagged(m),
    )
    const main = twin && !untagged(base) ? twin : base
    for (const m of parts) {
      if (m !== main) {
        out.add(partKey(m))
      }
    }
  }
  return out
}

// Whether a mod reads implicit: its download is, or it came in that download beside the main mod.
export function readsImplicit(
  implicit: boolean | undefined,
  mod: PartMod,
  all: readonly PartMod[],
) {
  return (
    Boolean(implicit) ||
    supportParts(
      all.filter((m) => m.key === mod.key),
      all,
    ).has(partKey(mod))
  )
}
