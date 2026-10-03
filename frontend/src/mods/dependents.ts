function fold(id: string): string {
  return id.trim().toLowerCase()
}

export function requiredIdsOf(mod: {
  needs?: readonly string[] | null
  optional?: readonly string[] | null
  contentPackFor?: string | null
}): string[] {
  const optional = new Set((mod.optional ?? []).map(fold))
  const ids = (mod.needs ?? []).filter((id) => !optional.has(fold(id)))
  const pack = mod.contentPackFor?.trim()
  if (pack) {
    ids.push(pack)
  }
  return ids
}

export function dependentsOf<
  T extends {
    uniqueId: string
    name: string
    enabled?: boolean
    needs?: readonly string[] | null
    optional?: readonly string[] | null
    contentPackFor?: string | null
  },
>(mods: readonly T[], removing: readonly T[]): T[] {
  const gone = new Set(removing.map((m) => fold(m.uniqueId)))
  const out: T[] = []
  const seen = new Set<string>()
  for (const mod of mods) {
    const id = fold(mod.uniqueId)
    if (
      mod.enabled &&
      !gone.has(id) &&
      !seen.has(id) &&
      requiredIdsOf(mod).some((need) => gone.has(fold(need)))
    ) {
      seen.add(id)
      out.push(mod)
    }
  }
  return out
}
