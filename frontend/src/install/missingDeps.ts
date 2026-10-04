import type {
  Missing,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { sameId } from '../mods/lookup.ts'
import { refWant, type Want } from '../queue/actions.ts'

interface ProfileLike {
  entries?: Array<{ mods?: Array<{ uniqueId?: string }> | null }> | null
}

interface MissingOffer {
  dependentName: string
  missing: Missing[]
}

const text = (obj: Record<string, unknown> | undefined, key: string) => {
  const value = obj?.[key]
  return typeof value === 'string' ? value.trim() : ''
}

function requiredUniqueIds(manifest: Record<string, unknown>): string[] {
  const own = text(manifest, 'UniqueID')
  const ids: string[] = []
  const seen = new Set<string>()
  const add = (id: string) => {
    if (id === '' || (own !== '' && sameId(id, own))) {
      return
    }
    const key = id.toLowerCase()
    if (seen.has(key)) {
      return
    }
    seen.add(key)
    ids.push(id)
  }
  const deps = manifest.Dependencies
  if (Array.isArray(deps)) {
    for (const dep of deps) {
      if (dep !== null && typeof dep === 'object') {
        const rec = dep as Record<string, unknown>
        if (rec.IsRequired !== false) {
          add(text(rec, 'UniqueID'))
        }
      }
    }
  }
  const pack = manifest.ContentPackFor
  if (pack !== null && typeof pack === 'object') {
    add(text(pack as Record<string, unknown>, 'UniqueID'))
  }
  return ids
}

function installedUniqueIds(profile: ProfileLike): string[] {
  return (profile.entries ?? []).flatMap((e) =>
    (e.mods ?? []).map((m) => m.uniqueId?.trim() ?? '').filter((id) => id !== ''),
  )
}

function missingRequired(manifest: Record<string, unknown>, profile: ProfileLike): string[] {
  const have = installedUniqueIds(profile)
  return requiredUniqueIds(manifest).filter((id) => !have.some((h) => sameId(h, id)))
}

function andList(names: string[]): string {
  if (names.length <= 1) {
    return names[0] ?? ''
  }
  if (names.length === 2) {
    return `${names[0]} and ${names[1]}`
  }
  return `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`
}

function depName(missing: Missing): string {
  const named = missing.where?.pageName?.trim() ?? ''
  return named === '' ? missing.uniqueId : named
}

function offersFor(dependentIds: readonly string[], result: Result | null): MissingOffer[] {
  if (dependentIds.length === 0) {
    return []
  }
  const groups = new Map<string, MissingOffer>()
  for (const missing of result?.missing ?? []) {
    if (
      missing.reason === 'absent' &&
      !(missing.listed && missing.optional) &&
      dependentIds.some((id) => sameId(id, missing.dependentId))
    ) {
      const key = missing.dependentId.toLowerCase()
      const group = groups.get(key)
      if (group) {
        if (!group.missing.some((m) => sameId(m.uniqueId, missing.uniqueId))) {
          group.missing.push(missing)
        }
      } else {
        groups.set(key, { dependentName: missing.dependentName, missing: [missing] })
      }
    }
  }
  return [...groups.values()].filter((g) => g.missing.length > 0)
}

function stillMissing(offer: MissingOffer, result: Result | null): Missing[] {
  if (!result) {
    return offer.missing
  }
  return offer.missing.filter((m) =>
    (result.missing ?? []).some(
      (r) =>
        r.reason === 'absent' &&
        sameId(r.dependentId, m.dependentId) &&
        sameId(r.uniqueId, m.uniqueId),
    ),
  )
}

function wantOf(missing: Missing): Want | null {
  const { where } = missing
  return where?.url ? refWant(where, 'dependency') : null
}

function wantsOf(missing: Missing[]): Want[] {
  const out: Want[] = []
  const seen = new Set<string>()
  for (const item of missing) {
    const want = wantOf(item)
    if (want) {
      const key = want.repo ? `gh:${want.repo}` : `nx:${want.modId}`
      if (!seen.has(key)) {
        seen.add(key)
        out.push(want)
      }
    }
  }
  return out
}

export type { MissingOffer }
export {
  andList,
  depName,
  missingRequired,
  offersFor,
  requiredUniqueIds,
  stillMissing,
  wantOf,
  wantsOf,
}
