import { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'

// The CHANGELOG a Thunderstore package publishes, read through Browse's details, which the browse service keeps for
// the session. Empty while it loads, when the package has none, or when the site cannot be reached, so the row that
// offers it stays hidden.
export function usePackageChangelog(game: string, pkg: string): string {
  const { data: changelog } = useLoaded(
    game && pkg ? () => Details(game, 'thunderstore', pkg).then((d) => d.changelog ?? '') : null,
    [game, pkg],
    '',
  )
  return changelog
}
