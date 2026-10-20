// Mod ids read "<format>:<local>". The local part is what a person knows a mod by, and what SMAPI's log and
// manifests call it.
export function localId(id: string): string {
  const colon = id.indexOf(':')
  return colon < 0 ? id : id.slice(colon + 1)
}

// SMAPI ids compare case-insensitively and ignore surrounding space.
export function idKey(id: string): string {
  return id.trim().toLowerCase()
}

export function requiredIdsOf(mod: {
  needs?: readonly string[] | null
  optional?: readonly string[] | null
  contentPackFor?: string | null
}): string[] {
  const optional = new Set((mod.optional ?? []).map(idKey))
  const ids = (mod.needs ?? []).filter((id) => !optional.has(idKey(id)))
  const pack = mod.contentPackFor?.trim()
  if (pack) {
    ids.push(pack)
  }
  return ids
}

export function dependentsOf<
  T extends {
    id: string
    name: string
    enabled?: boolean
    needs?: readonly string[] | null
    optional?: readonly string[] | null
    contentPackFor?: string | null
  },
>(mods: readonly T[], removing: readonly T[]): T[] {
  const gone = new Set(removing.map((m) => idKey(m.id)))
  const out: T[] = []
  const seen = new Set<string>()
  for (const mod of mods) {
    const id = idKey(mod.id)
    if (
      mod.enabled &&
      !gone.has(id) &&
      !seen.has(id) &&
      requiredIdsOf(mod).some((need) => gone.has(idKey(need)))
    ) {
      seen.add(id)
      out.push(mod)
    }
  }
  return out
}

// The entry keys of every mod the chosen ones need, followed through what those need in turn, leaving out the chosen.
export function requirementKeys<
  T extends {
    id: string
    key: string
    needs?: readonly string[] | null
    optional?: readonly string[] | null
    contentPackFor?: string | null
  },
>(all: readonly T[], chosen: readonly T[]): string[] {
  const byId = new Map(all.map((m) => [idKey(m.id), m]))
  const have = new Set(chosen.map((m) => m.key))
  const out: string[] = []
  const queue = [...chosen]
  for (let i = 0; i < queue.length; i++) {
    for (const id of requiredIdsOf(queue[i] ?? {})) {
      const dep = byId.get(idKey(id))
      if (dep && !have.has(dep.key)) {
        have.add(dep.key)
        out.push(dep.key)
        queue.push(dep)
      }
    }
  }
  return out
}
