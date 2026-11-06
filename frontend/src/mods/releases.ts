import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import {
  CachedReleaseChangelog,
  ReleaseChangelog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useLastGood } from '../shell/useLastGood.ts'
import type { InlineError } from '../toasts/report.ts'

type Loaded = { logs: Changelog[] } | { error: InlineError } | null

// GitHub releases come from the cached releases list; Nexus versions are already in the mod's cached details. The
// stored list shows first, the live read replaces it.
function useReleases(open: boolean, githubRepo: string) {
  const { data, error, reload } = useLastGood<Changelog[]>(
    githubRepo === '' ? null : `releases:${githubRepo}`,
    open && githubRepo !== ''
      ? () => ReleaseChangelog(githubRepo).then((logs) => logs ?? [])
      : null,
    [open],
    { seed: () => CachedReleaseChangelog(githubRepo) },
  )
  let state: Loaded = null
  if (data !== null) {
    state = { logs: data }
  } else if (error !== null) {
    state = { error }
  }
  return { state, retry: reload }
}

export { useReleases }
