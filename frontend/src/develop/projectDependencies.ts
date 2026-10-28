import { useCallback, useEffect, useRef, useState } from 'react'
import type { Dependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  AddDependency,
  Dependencies,
  RemoveDependency,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sameValue } from '../shell/sameValue.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { startDependencies, useKnownPackages } from './dependencies.ts'

/** A dependency as the dialog edits it: unticked ones are dropped on save, new ones added. */
export interface DependencyRow extends Dependency {
  keep: boolean
  added: boolean
}

/** The project's dependencies as loaded, the rows being edited, and whether they differ. */
export function useDependencyRows(game: string, dir: string) {
  const [loaded, setLoaded] = useState<DependencyRow[]>([])
  const [rows, setRows] = useState<DependencyRow[]>([])
  const [error, setError] = useState<InlineError | null>(null)
  const [ready, setReady] = useState(false)
  const latest = useRef(0)
  const load = useCallback(() => {
    latest.current += 1
    const id = latest.current
    setError(null)
    Dependencies(game, dir)
      .then((deps) => {
        if (id !== latest.current) {
          return
        }
        const read = (deps ?? []).map((d) => ({ ...d, keep: true, added: false }))
        setLoaded(read)
        // Rows added before the read finished stay.
        setRows((now) => [
          ...read,
          ...now.filter((r) => r.added && !read.some((d) => d.package === r.package)),
        ])
        setReady(true)
      })
      .catch((e: unknown) => id === latest.current && setError(inlineError(e)))
  }, [game, dir])
  useEffect(() => {
    load()
    return () => {
      latest.current += 1
    }
  }, [load])
  return {
    loaded,
    rows,
    setRows,
    dirty: !sameValue(loaded, rows),
    loading: !ready && error === null,
    error,
    retry: load,
  }
}

/** Writes the edited dependencies into the project and installs the new ones into its profile. */
export async function saveDependencies(
  game: string,
  dir: string,
  loaded: DependencyRow[],
  rows: DependencyRow[],
) {
  for (const row of rows) {
    const was = loaded.find((l) => l.package === row.package)
    if (was && !row.keep) {
      await RemoveDependency(game, dir, row.package)
    } else if (was && row.keep && was.soft !== row.soft) {
      await AddDependency(game, dir, row.package, row.version, row.soft)
    }
  }
  const added = rows.filter((r) => r.added && r.keep)
  await startDependencies(
    game,
    dir,
    {
      ids: added.map((r) => r.package),
      optional: added.filter((r) => r.soft).map((r) => r.package),
    },
    useKnownPackages.getState(),
  )
}
