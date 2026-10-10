import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Played } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { Get } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { gameSetupNeeded } from '../firstrun/needed.ts'
import { useRefreshOnFocus } from '../firstrun/useRefreshOnFocus.ts'
import { useLoader } from '../loader/store.ts'
import { isGameId } from '../nav/store.ts'
import { useSettings } from '../settings/store.ts'
import { type InlineError, inlineError, reportError } from '../toasts/report.ts'
import { type GameStatus, loaderCaption, loadGameStatus } from './status.ts'

type Game = GameInfo

interface GameState {
  profiles: Profile[]
  lastPlayedId: string
  lastPlayedAt: string
  lastPlayed: Played | undefined
  playtimeMs: number
  setupNeeded: boolean
}

// The games, their profiles and last-played state: what a game tile shows, shared by Game Select and the switcher.
function useGameTiles() {
  const { t } = useLingui()
  const [status, setStatus] = useState<GameStatus | null>(null)
  const [states, setStates] = useState<Record<string, GameState>>({})
  const [loadError, setLoadError] = useState<InlineError | null>(null)
  const loaderStatus = useLoader((s) => s.status)
  const checkLoader = useLoader((s) => s.check)
  const lastGame = useSettings((s) => s.lastGame)
  const availableGames = status?.games.filter((g) => g.available) ?? []
  const loaderGame = (availableGames.find((g) => g.id === lastGame) ?? availableGames[0])?.id
  useEffect(() => {
    if (loaderGame) {
      checkLoader(loaderGame)
    }
  }, [checkLoader, loaderGame])
  const refresh = useCallback(() => {
    setLoadError(null)
    Promise.all([loadGameStatus(), Get()])
      .then(async ([s, settings]) => {
        const entries = await Promise.all(
          s.games
            .filter((g) => g.available)
            .map(async (g): Promise<[string, GameState]> => {
              const [listed, setupNeeded] = await Promise.all([List(g.id), gameSetupNeeded(g)])
              const profiles = listed ?? []
              const played = settings.lastPlayed?.[g.id]
              const still = Boolean(
                played?.profile && profiles.some((p) => p.id === played.profile),
              )
              return [
                g.id,
                {
                  profiles,
                  setupNeeded,
                  lastPlayedId: still && played ? played.profile : '',
                  lastPlayedAt: still && played ? played.at : '',
                  lastPlayed: still && played ? played : undefined,
                  playtimeMs: played?.playtimeMs ?? 0,
                },
              ]
            }),
        )
        setStatus(s)
        setStates(Object.fromEntries(entries))
      })
      .catch((err: unknown) => {
        reportError(t`Could not read your games`)(err)
        setLoadError(inlineError(err))
      })
  }, [t])
  useEffect(refresh, [refresh])
  useRefreshOnFocus(refresh)
  const noteFor = (g: Game) => {
    const st = states[g.id]
    if (!g.available) {
      return t`Coming in a later Mortar version`
    }
    if (!st || st.setupNeeded) {
      return t`Not set up`
    }
    // The profile cards already name every profile.
    return ''
  }
  const tileProps = (g: Game) => {
    const st = states[g.id]
    const lastId = st?.lastPlayedId ?? ''
    const showCards = g.available && g.installed && st && !st.setupNeeded && st.profiles.length > 0
    return {
      game: g,
      openable: g.available,
      setupNeeded: g.available && (!st || st.setupNeeded),
      note: noteFor(g),
      loader: loaderCaption(g.loader, g.id === loaderGame ? loaderStatus : null),
      lastPlayedName: st?.profiles.find((p) => p.id === lastId)?.name ?? '',
      lastPlayedAt: st?.lastPlayedAt ?? '',
      lastPlayedId: st?.setupNeeded ? '' : lastId,
      playtimeMs: st?.playtimeMs ?? 0,
      cards:
        showCards && isGameId(g.id)
          ? { gameId: g.id, profiles: st.profiles, lastPlayed: st.lastPlayed }
          : undefined,
    }
  }
  return { status, loadError, refresh, tileProps }
}

export { useGameTiles }
