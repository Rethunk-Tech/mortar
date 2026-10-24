import { ProtonBuilds } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'

// The Proton builds Steam has that the game's copy could run through; none unless it is a Windows copy outside Steam
// on Linux.
export function useProtonBuilds(game: string): string[] {
  return useLoaded<string[]>(
    game === '' ? null : () => ProtonBuilds(game).then((b) => b ?? []),
    [game],
    [],
    reportUnexpected,
  ).data
}
