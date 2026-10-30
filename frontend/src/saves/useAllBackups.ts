import { useLoaded } from '@rethunk/react-load-hooks'
import { Events } from '@wailsio/runtime'
import { useEffect, useRef } from 'react'
import type { Backup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/backup/models.ts'
import { ListBackups } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Every backup of the game's saves, read once for the whole page so each save's button can count its own; null until
// the first read. `stamp` changes when the saves do, which is when a backup may have been made: that reads again but
// keeps the counts shown, where another game starts over.
export function useAllBackups(game: string, stamp: string) {
  const { data: all, reload } = useLoaded<Backup[] | null>(
    () => ListBackups(game, '').then((rows) => rows ?? []),
    [game],
    null,
    reportUnexpected,
  )
  const seen = useRef({ game, stamp })
  useEffect(() => {
    const was = seen.current
    seen.current = { game, stamp }
    if (was.game === game && was.stamp !== stamp) {
      reload()
    }
  }, [game, stamp, reload])
  // A scheduled pass or `mortar backups create` makes backups without changing the saves.
  useEffect(() => Events.On('saves:scheduled', reload), [reload])
  return { all, reload }
}
