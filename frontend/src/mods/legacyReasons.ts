import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { idKey, localId } from './dependents.ts'

// The site categories that hold libraries: Thunderstore's Libraries and Nexus's Modding Tools.
const LIBRARY_CATEGORIES = new Set(['libraries', 'modding tools'])

type LegacyMod = Pick<
  Mod,
  'key' | 'id' | 'name' | 'enabled' | 'needs' | 'optional' | 'contentPackFor'
>

const isPack = (m: LegacyMod) =>
  Boolean(m.contentPackFor) && idKey(m.contentPackFor ?? '') !== idKey(m.id)

const ID_SEPARATOR = /[.-]/

// A shorter id or name ("PIF", "Core") turns up inside unrelated mods' names.
const MIN_NAMED = 4

// A mod id's own name, without its format and author: smapi:Bouhm.NPCMapLocations is npcmaplocations.
const idName = (id: string) => {
  const local = localId(idKey(id))
  const cut = local.search(ID_SEPARATOR)
  return cut < 0 ? local : local.slice(cut + 1)
}

// Whether mod a names mod b in its id or name, as an add-on does ("PortraiturePlus", "Schedule Viewer - SVE Add-on").
const names = (a: LegacyMod, b: LegacyMod) => {
  const id = idName(b.id)
  const name = b.name.trim().toLowerCase()
  return (
    (id.length >= MIN_NAMED && idKey(a.id).includes(id)) ||
    (name.length >= MIN_NAMED && a.name.toLowerCase().includes(name))
  )
}

// The entry keys a switched-on mod requires (its required dependencies and the framework of a content pack), each
// that another entry requires without leading back to it, so the members of a cycle nothing outside needs stay out.
// An entry required only by its own add-ons, each naming it and none required itself, is a mod the player chose.
function requiredKeys(mods: readonly LegacyMod[]): Set<string> {
  const keyOf = new Map(mods.map((m) => [idKey(m.id), m.key]))
  const requires = new Map<string, Set<string>>()
  for (const m of mods.filter((x) => x.enabled)) {
    const optional = new Set((m.optional ?? []).map(idKey))
    const wants = (m.needs ?? []).filter((n) => !optional.has(idKey(n)))
    for (const want of isPack(m) ? [...wants, m.contentPackFor ?? ''] : wants) {
      const key = keyOf.get(idKey(want))
      if (key !== undefined && key !== m.key) {
        requires.set(m.key, (requires.get(m.key) ?? new Set()).add(key))
      }
    }
  }
  const reaches = (from: string, to: string) => {
    const seen = new Set<string>()
    const stack = [...(requires.get(from) ?? [])]
    for (let n = stack.pop(); n !== undefined; n = stack.pop()) {
      if (n === to) {
        return true
      }
      if (!seen.has(n)) {
        seen.add(n)
        stack.push(...(requires.get(n) ?? []))
      }
    }
    return false
  }
  const out = new Set<string>()
  for (const [by, needs] of requires) {
    for (const key of needs) {
      if (!reaches(key, by)) {
        out.add(key)
      }
    }
  }
  const modsOf = (key: string) => mods.filter((m) => m.key === key)
  const namesAny = (by: string, key: string) =>
    modsOf(by).some((a) => modsOf(key).some((b) => names(a, b)))
  for (const key of [...out]) {
    const dependents = [...requires].filter(([, needs]) => needs.has(key)).map(([by]) => by)
    if (dependents.every((by) => !out.has(by) && namesAny(by, key))) {
      out.delete(key)
    }
  }
  return out
}

/**
 * The entries of a profile from before install reasons that its game's defaults mark implicit: what another mod
 * requires (a library, or the framework a content pack names), what sits in a library category, and the packs that
 * only supply tilesheets. Content packs and mods required only by their own add-ons stay explicit, as do entries
 * added since Mortar kept reasons.
 */
export type LegacyWhy = 'required' | 'category' | 'supporting'

export function legacyImplicit(
  { entries, intentSince }: Pick<Profile, 'entries' | 'intentSince'>,
  mods: readonly LegacyMod[],
  categoryOf: (entry: Entry) => string,
  supporting: readonly string[],
): { key: string; why: LegacyWhy }[] {
  const since = Date.parse(intentSince ?? '')
  const required = requiredKeys(mods)
  const out: { key: string; why: LegacyWhy }[] = []
  for (const e of entries ?? []) {
    const added = Date.parse(e.added ?? '')
    const legacy = Number.isNaN(since) || Number.isNaN(added) || added < since
    if (!legacy || e.implicit || e.source.kind === 'smapi' || e.source.kind === 'mortar') {
      continue
    }
    // A content pack is content even when another mod requires it (an expansion its add-ons need).
    const pack = mods.some((m) => m.key === e.key && isPack(m))
    if (supporting.includes(e.key)) {
      out.push({ key: e.key, why: 'supporting' })
    } else if (!pack && required.has(e.key)) {
      out.push({ key: e.key, why: 'required' })
    } else if (!pack && LIBRARY_CATEGORIES.has(categoryOf(e).trim().toLowerCase())) {
      out.push({ key: e.key, why: 'category' })
    }
  }
  return out
}
