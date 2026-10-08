import { Events } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import type { GameModsProgress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

/** The latest progress the backend reports for an import or move in this game while `running`; null before the first report. */
export function useGameModsProgress(game: string, running: boolean) {
  const [progress, setProgress] = useState<GameModsProgress | null>(null)
  useEffect(() => {
    setProgress(null)
    if (!running || typeof Events.On !== 'function') {
      return
    }
    return Events.On('profile:game-mods-progress', (e) => {
      if (e.data.game === game) {
        setProgress(e.data)
      }
    })
  }, [game, running])
  return progress
}
