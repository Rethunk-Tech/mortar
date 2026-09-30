import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useLoader } from '../loader/store.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLaunch } from './store.ts'

export function windowsVanillaAfterForcesCheck(
  ok: boolean,
  forces: boolean,
): 'warn' | 'start' | 'abort' {
  if (!ok) {
    return 'abort'
  }
  return forces ? 'warn' : 'start'
}

export function playOpenProfile() {
  const game = routeGame(useNav.getState().route)
  const { openId } = useProfiles.getState()
  const { status, starting, start } = useLaunch.getState()
  if (!game || openId === '') {
    return
  }
  const launching = status?.game === game && status.state === State.Launching
  const busy = starting || useLoader.getState().installing
  if (launching || busy) {
    return
  }
  start(game, openId, false)
}
