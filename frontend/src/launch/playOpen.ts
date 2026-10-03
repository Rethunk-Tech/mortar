import { System } from '@wailsio/runtime'
import { create } from 'zustand'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { ForcesSMAPI } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { useLoader } from '../loader/store.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useLaunch } from './store.ts'

const linuxVanillaDirectKey = 'mortar.linuxVanillaDirect'

function vanillaBusy(game: string): boolean {
  const { status, starting } = useLaunch.getState()
  const launching = status?.game === game && status.state === State.Launching
  return launching || starting || useLoader.getState().installing
}

export function linuxVanillaDirectAgreed(): boolean {
  try {
    return localStorage.getItem(linuxVanillaDirectKey) === '1'
  } catch {
    return false
  }
}

export function rememberLinuxVanillaDirect() {
  try {
    localStorage.setItem(linuxVanillaDirectKey, '1')
  } catch {
    // Private mode can refuse localStorage; the next Play without mods asks again.
  }
}

export const useVanillaPrompt = create<{
  smapiWarn: boolean
  linuxDirect: boolean
  setSmapiWarn: (on: boolean) => void
  setLinuxDirect: (on: boolean) => void
}>((set) => ({
  smapiWarn: false,
  linuxDirect: false,
  setSmapiWarn: (smapiWarn) => set({ smapiWarn }),
  setLinuxDirect: (linuxDirect) => set({ linuxDirect }),
}))

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
  const { start } = useLaunch.getState()
  if (!game || openId === '') {
    return
  }
  if (vanillaBusy(game)) {
    return
  }
  start(game, openId, false)
}

export function playVanillaOpen() {
  const game = routeGame(useNav.getState().route)
  if (!game || vanillaBusy(game)) {
    return
  }
  const { startVanilla } = useLaunch.getState()
  const prompt = useVanillaPrompt.getState()
  if (!System.IsWindows()) {
    if (!linuxVanillaDirectAgreed()) {
      prompt.setLinuxDirect(true)
      return
    }
    startVanilla(game, true).catch(() => undefined)
    return
  }
  ForcesSMAPI(game)
    .then((forces) => {
      const next = windowsVanillaAfterForcesCheck(true, forces)
      if (next === 'warn') {
        prompt.setSmapiWarn(true)
        return
      }
      return startVanilla(game, false)
    })
    .catch((e: unknown) => {
      reportUnexpected(e)
    })
}
