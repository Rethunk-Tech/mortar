import { useLingui } from '@lingui/react/macro'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { SetLastGame } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { gameSetupNeeded } from '../firstrun/needed.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { reportError } from '../toasts/report.ts'

interface OpenDeps {
  remember: (id: string) => Promise<void>
  setupNeeded: (game: GameInfo) => Promise<boolean>
  onError: (title: string) => (err: unknown) => void
}

const live = (onError: OpenDeps['onError']): OpenDeps => ({
  remember: SetLastGame,
  setupNeeded: gameSetupNeeded,
  onError,
})

// Opens the game, or its setup when it has none yet, remembering it as the last one.
export function openGame(
  game: GameInfo,
  messages: { last: string; read: string },
  deps: OpenDeps,
): Promise<void> {
  if (!isGameId(game.id)) {
    return Promise.resolve()
  }
  const { id } = game
  deps.remember(id).catch(deps.onError(messages.last))
  return deps
    .setupNeeded(game)
    .then((needed) =>
      needed ? useNav.getState().openGameSetup(id) : useNav.getState().openGame(id),
    )
    .catch(deps.onError(messages.read))
}

// A function that opens any game it is given, for a list of them.
export function useOpenGame(): (game: GameInfo) => void {
  const { t } = useLingui()
  const messages = { last: t`Could not save the last game`, read: t`Could not read your games` }
  return (game) => {
    openGame(game, messages, live(reportError)).catch(() => undefined)
  }
}
