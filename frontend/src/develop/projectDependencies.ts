import { useEffect, useState } from 'react'
import type { Dependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  AddDependency,
  Dependencies,
  RemoveDependency,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { startDependencies, useKnownPackages } from './dependencies.ts'

/** A dependency as the dialog edits it: unticked ones are dropped on save, new ones added. */
export interface DependencyRow extends Dependency {
  keep: boolean
  added: boolean
}

const same = (a: DependencyRow[], b: DependencyRow[]) => JSON.stringify(a) === JSON.stringify(b)

/** The project's dependencies as loaded, the rows being edited, and whether they differ. */
export function useDependencyRows(game: string, dir: string) {
  const [loaded, setLoaded] = useState<DependencyRow[]>([])
  const [rows, setRows] = useState<DependencyRow[]>([])
  useEffect(() => {
    Dependencies(game, dir)
      .then((deps) => {
        const read = (deps ?? []).map((d) => ({ ...d, keep: true, added: false }))
        setLoaded(read)
        setRows(read)
      })
      .catch(() => undefined)
  }, [game, dir])
  return { loaded, rows, setRows, dirty: !same(loaded, rows) }
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
