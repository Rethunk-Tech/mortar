import type {
  Missing,
  Result,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { localId } from '../mods/dependents.ts'
import { sameId } from '../mods/lookup.ts'
import type { Want } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'

interface ProfileLike {
  entries?: Array<{ mods?: Array<{ id?: string }> | null }> | null
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
    (e.mods ?? []).map((m) => m.id?.trim() ?? '').filter((id) => id !== ''),
  )
}

function missingRequired(manifest: Record<string, unknown>, profile: ProfileLike): string[] {
  const have = installedUniqueIds(profile)
  return requiredUniqueIds(manifest).filter((id) => !have.some((h) => sameId(localId(h), id)))
}

function depName(missing: Missing): string {
  const named = missing.where?.pageName?.trim() ?? ''
  return named === '' ? localId(missing.id) : named
}

function offersFor(dependentIds: readonly string[], result: Result | null): MissingOffer[] {
  if (dependentIds.length === 0) {
    return []
  }
  const groups = new Map<string, MissingOffer>()
  for (const missing of result?.missing ?? []) {
    if (
      missing.reason === 'absent' &&
      !missing.external &&
      !(missing.listed && missing.optional) &&
      dependentIds.some((id) => sameId(id, missing.dependentId))
    ) {
      const key = missing.dependentId.toLowerCase()
      const group = groups.get(key)
      if (group) {
        if (!group.missing.some((m) => sameId(m.id, missing.id))) {
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
      (r) => r.reason === 'absent' && sameId(r.dependentId, m.dependentId) && sameId(r.id, m.id),
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
export { depName, missingRequired, offersFor, requiredUniqueIds, stillMissing, wantOf, wantsOf }
