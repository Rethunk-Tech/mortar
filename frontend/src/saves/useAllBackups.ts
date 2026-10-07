import { useCallback, useEffect, useState } from 'react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import { ListBackups } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Every backup of the game's saves, read once for the whole page so each save's button can count its own; null until
// the first read. `stamp` changes when the saves do, which is when a backup may have been made.
export function useAllBackups(game: string, stamp: string) {
  const [all, setAll] = useState<Backup[] | null>(null)
  const read = useCallback(
    (g: string) =>
      ListBackups(g, '')
        .then((rows) => setAll(rows ?? []))
        .catch(reportUnexpected),
    [],
  )
  const key = `${game}\0${stamp}`
  useEffect(() => {
    read(key.split('\0')[0] ?? '')
  }, [key, read])
  const reload = useCallback(() => read(game), [read, game])
  return { all, reload }
}
