import { useEffect, useState } from 'react'
import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { ReleaseChangelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { inlineError } from '../toasts/report.ts'

type Loaded = { logs: Changelog[] } | { error: ReturnType<typeof inlineError> } | null

// GitHub releases come from the cached releases list; Nexus versions are already in the mod's cached details.
function useReleases(open: boolean, githubRepo: string) {
  const [state, setState] = useState<Loaded>(null)
  const [attempt, setAttempt] = useState(0)
  useEffect(() => {
    if (!open || githubRepo === '' || attempt < 0) {
      return
    }
    let live = true
    setState(null)
    ReleaseChangelog(githubRepo).then(
      (logs) => live && setState({ logs: logs ?? [] }),
      (e: unknown) => live && setState({ error: inlineError(e) }),
    )
    return () => {
      live = false
    }
  }, [open, githubRepo, attempt])
  return { state, retry: () => setAttempt((n) => n + 1) }
}

export { useReleases }
