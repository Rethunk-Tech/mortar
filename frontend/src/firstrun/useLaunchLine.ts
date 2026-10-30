import { useLoaded } from '@rethunk/react-load-hooks'
import { LoaderLaunchLine } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'

// The Steam launch-options line that starts the game's loader, empty until the backend answers.
export function useLaunchLine(game: string, installDir: string): string {
  const { data } = useLoaded(
    installDir ? () => LoaderLaunchLine(game, installDir).catch(() => '') : null,
    [game, installDir],
    '',
  )
  return data
}
