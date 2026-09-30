interface Counted {
  entries?: { source: { kind: string }; mods?: unknown[] | null }[] | null
}

// SMAPI's bundled mods sit in every profile and are never shown, so they are not counted.
export function userModCount(profile: Counted): number {
  return (profile.entries ?? [])
    .filter((e) => e.source.kind !== 'smapi')
    .reduce((n, e) => n + (e.mods ?? []).length, 0)
}
