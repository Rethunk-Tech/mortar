import { useLingui } from '@lingui/react/macro'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { SetLastGame } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { gameSetupNeeded } from '../firstrun/needed.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { reportError } from '../toasts/report.ts'

// Opens the game, or its setup when it has none yet, remembering it as the last one.
export function useOpenGame(game: GameInfo) {
  const { t } = useLingui()
  return () => {
    if (!isGameId(game.id)) {
      return
    }
    const { id } = game
    SetLastGame(id).catch(reportError(t`Could not save the last game`))
    gameSetupNeeded(game)
      .then((needed) =>
        needed ? useNav.getState().openGameSetup(id) : useNav.getState().openGame(id),
      )
      .catch(reportError(t`Could not read your games`))
  }
}
