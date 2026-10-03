import { requiredIdsOf } from './dependents.ts'

function fold(id: string): string {
  return id.trim().toLowerCase()
}

export interface RequirementSeed {
  uniqueId: string
  name: string
  enabled?: boolean
  needs?: readonly string[] | null
  optional?: readonly string[] | null
  contentPackFor?: string | null
}

export function pendingRequired<T extends RequirementSeed>(
  mods: readonly T[],
  enabling: readonly T[],
): T[] {
  const byId = new Map<string, T>()
  for (const mod of mods) {
    byId.set(fold(mod.uniqueId), mod)
  }
  const seen = new Set(enabling.map((m) => fold(m.uniqueId)))
  const out: T[] = []
  const queue = enabling.flatMap((m) => requiredIdsOf(m))
  while (queue.length > 0) {
    const id = queue.shift()
    if (id) {
      const low = fold(id)
      if (!seen.has(low)) {
        seen.add(low)
        const dep = byId.get(low)
        if (dep && !dep.enabled) {
          out.push(dep)
          queue.push(...requiredIdsOf(dep))
        }
      }
    }
  }
  return out
}

export function enableRequirementsDecision(
  mode: string,
  pendingCount: number,
): 'enable' | 'ask' | 'skip' {
  if (pendingCount === 0 || mode === 'never') {
    return 'skip'
  }
  if (mode === 'ask') {
    return 'ask'
  }
  return 'enable'
}
