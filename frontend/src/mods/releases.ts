import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { ReleaseChangelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { inlineError } from '../toasts/report.ts'

type Loaded = { logs: Changelog[] } | { error: ReturnType<typeof inlineError> } | null

// GitHub releases come from the cached releases list; Nexus versions are already in the mod's cached details.
function useReleases(open: boolean, githubRepo: string) {
  const { data, error, reload } = useLoaded<Loaded>(
    open && githubRepo !== ''
      ? () => ReleaseChangelog(githubRepo).then((logs) => ({ logs: logs ?? [] }))
      : null,
    [open, githubRepo],
    null,
  )
  const state: Loaded = error === null ? data : { error: inlineError(error) }
  return { state, retry: reload }
}

export { useReleases }
