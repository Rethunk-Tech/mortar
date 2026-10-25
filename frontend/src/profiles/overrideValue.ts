import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export function resolveOverride(
  key: string,
  gameValue: string,
  overrides?: Record<string, string> | null,
): string {
  if (overrides && Object.hasOwn(overrides, key)) {
    return overrides[key] ?? gameValue
  }
  return gameValue
}

export function foldedOverrides(profile: Pick<Profile, 'overrides'>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(profile.overrides ?? {})) {
    if (value !== undefined) {
      out[key] = value
    }
  }
  return out
}
