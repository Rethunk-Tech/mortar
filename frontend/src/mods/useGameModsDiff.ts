import { useCallback, useEffect, useRef, useState } from 'react'
import type { GameModsDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { GameModsDiff as fetchDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useFolderEvent } from '../shell/useFolderEvent.ts'

/**
 * The game's Mods folder compared with this profile. Read again when the profile changes (a new `updated` stamp: an
 * update, a move, a rollback) or the folder does; an unreadable folder shows no diff.
 */
export function useGameModsDiff(game: string, profileId: string, updated: string) {
  const [diff, setDiff] = useState<GameModsDiff | null>(null)
  const gen = useRef(0)
  const reload = useCallback(() => {
    gen.current += 1
    const token = gen.current
    if (game === '' || profileId === '' || updated === '') {
      setDiff(null)
      return
    }
    fetchDiff(game, profileId)
      .then((d) => token === gen.current && setDiff(d))
      .catch(() => token === gen.current && setDiff(null))
  }, [game, profileId, updated])
  useEffect(reload, [reload])
  useFolderEvent('library:mods-folder', game, reload)
  return { diff, reload }
}
