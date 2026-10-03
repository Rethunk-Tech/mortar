import type { Played } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'

export function profileCardLastPlayedIso(
  profileId: string,
  played: Played | undefined,
  runStarted: Record<string, string>,
): string {
  if (played?.profile === profileId && played.at) {
    return played.at
  }
  return runStarted[profileId] ?? ''
}
