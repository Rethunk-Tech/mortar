import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

/** The remembered profile when it is still listed, else the game's first remaining one, else none. */
export function settledLastProfile(
  profiles: Pick<Profile, 'id'>[],
  last: string | undefined,
): string {
  return profiles.find((p) => p.id === last)?.id ?? profiles[0]?.id ?? ''
}
