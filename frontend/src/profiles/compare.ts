import type {
  Entry,
  Profile,
  Source,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { cmpText } from '../mods/cmpText.ts'
import { idKey } from '../mods/dependents.ts'
import { sameId } from '../mods/lookup.ts'
import { userModEntries } from './count.ts'

interface CompareSide {
  uniqueId: string
  name: string
  version: string
  enabled: boolean
  key: string
  source: Source
}

interface ComparePair {
  uniqueId: string
  name: string
  a: CompareSide
  b: CompareSide
}

interface ProfileCompare {
  onlyA: CompareSide[]
  onlyB: CompareSide[]
  differentVersion: ComparePair[]
  differentEnabled: ComparePair[]
  identical: ComparePair[]
}

function enabledOf(entry: Entry, uniqueId: string): boolean {
  return !(entry.disabled ?? []).some((id) => sameId(id, uniqueId))
}

function indexUserMods(profile: Profile): Map<string, CompareSide> {
  const out = new Map<string, CompareSide>()
  for (const e of userModEntries(profile.entries)) {
    for (const m of e.mods ?? []) {
      const k = idKey(m.uniqueId)
      if (!out.has(k)) {
        out.set(k, {
          uniqueId: m.uniqueId,
          name: m.name,
          version: m.version,
          enabled: enabledOf(e, m.uniqueId),
          key: e.key,
          source: e.source,
        })
      }
    }
  }
  return out
}

function sortSides(sides: CompareSide[]): CompareSide[] {
  return [...sides].sort((a, b) => {
    const n = cmpText(a.name, b.name)
    if (n !== 0) {
      return n
    }
    return cmpText(a.uniqueId, b.uniqueId)
  })
}

function sortPairs(pairs: ComparePair[]): ComparePair[] {
  return [...pairs].sort((a, b) => {
    const n = cmpText(a.name, b.name)
    if (n !== 0) {
      return n
    }
    return cmpText(a.uniqueId, b.uniqueId)
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
    identical: ComparePair[]
  },
) {
  const name = pairName(side, other)
  const pair: ComparePair = { uniqueId: side.uniqueId, name, a: side, b: other }
  const versionDiff = side.version !== other.version
  const enabledDiff = side.enabled !== other.enabled
  if (versionDiff) {
    buckets.differentVersion.push(pair)
  }
  if (enabledDiff) {
    buckets.differentEnabled.push(pair)
  }
  if (!(versionDiff || enabledDiff)) {
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
  const identical: ComparePair[] = []

  for (const [k, side] of left) {
    const other = right.get(k)
    if (other) {
      classifyPair(side, other, { differentVersion, differentEnabled, identical })
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
    identical: sortPairs(identical),
  }
}

function sideLabel(side: CompareSide, enabled: string, disabled: string): string {
  const state = side.enabled ? enabled : disabled
  return `${side.name} · ${side.version} · ${state}`
}

interface SectionShared {
  enabled: string
  disabled: string
  pending: boolean
  lockedReason: (profile: Profile) => string
}

export type { ComparePair, CompareSide, ProfileCompare, SectionShared }
export { compareProfiles, sideLabel }
