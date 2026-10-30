import { useLoaded } from '@rethunk/react-load-hooks'
import { ProtonBuilds } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// The Proton builds Steam has that the game's copy could run through; none unless it is a Windows copy outside Steam
// on Linux.
export function useProtonBuilds(game: string, read = true): string[] {
  return useLoaded<string[]>(
    game === '' || !read ? null : () => ProtonBuilds(game).then((b) => b ?? []),
    [game, read],
    [],
    reportUnexpected,
  ).data
}
