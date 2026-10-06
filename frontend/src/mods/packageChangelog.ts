import { useEffect, useState } from 'react'
import { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'

// The CHANGELOG a Thunderstore package publishes, read through Browse's details, which the browse service keeps for
// the session. Empty while it loads, when the package has none, or when the site cannot be reached, so the row that
// offers it stays hidden.
export function usePackageChangelog(game: string, pkg: string): string {
  const [changelog, setChangelog] = useState('')
  useEffect(() => {
    setChangelog('')
    if (!(game && pkg)) {
      return
    }
    let live = true
    Details(game, 'thunderstore', pkg).then(
      (d) => live && setChangelog(d.changelog ?? ''),
      () => undefined,
    )
    return () => {
      live = false
    }
  }, [game, pkg])
  return changelog
}
