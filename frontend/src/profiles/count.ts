interface Counted {
  entries?: { source: { kind: string }; mods?: unknown[] | null }[] | null
}

// The bundled mods (SMAPI's and Mortar's bridge) sit in every profile and are never shown, so they are not counted.
export function userModEntries<T extends { source: { kind: string } }>(
  entries: T[] | null | undefined,
): T[] {
  return (entries ?? []).filter((e) => e.source.kind !== 'smapi' && e.source.kind !== 'mortar')
}

export function userModCount(profile: Counted): number {
  return userModEntries(profile.entries).reduce((n, e) => n + (e.mods ?? []).length, 0)
}
