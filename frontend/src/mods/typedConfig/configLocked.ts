import { State } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { useLaunch } from '../../launch/store.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useTypedConfig } from './store.ts'

const useLaunchOfOpen = (state: State) => {
  const openId = useProfiles((s) => s.openId)
  return useLaunch(
    (s) =>
      (state === State.Launching && s.starting && s.startingProfile === openId) ||
      (s.status?.profile === openId && s.status.state === state),
  )
}

/** A launch placing the open profile's files refuses every write; the in-game menu's edits wait for the next start. */
export function useConfigLocked(): boolean {
  const starting = useLaunchOfOpen(State.Launching)
  const inGameMenu = useTypedConfig(
    (s) => s.files.find((f) => f.name === s.current)?.format === 'gmcm',
  )
  return starting && !inGameMenu
}

/** The open profile's game is up: its config files take edits, which most mods read only at their next start. */
export const useConfigGameRunning = () => useLaunchOfOpen(State.Running)
