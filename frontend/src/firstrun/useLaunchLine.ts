import { useEffect, useState } from 'react'
import { LoaderLaunchLine } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'

// The Steam launch-options line that starts the game's loader, empty until the backend answers.
export function useLaunchLine(game: string, installDir: string): string {
  const [line, setLine] = useState('')
  useEffect(() => {
    if (!installDir) {
      setLine('')
      return
    }
    LoaderLaunchLine(game, installDir).then(setLine, () => setLine(''))
  }, [game, installDir])
  return line
}
