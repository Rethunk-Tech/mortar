import { useMemo } from 'react'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { isLocked } from './locked.ts'

type LaunchLocks = Pick<
  ReturnType<typeof useLaunch.getState>,
  'status' | 'starting' | 'startingProfile'
>

/** The game holds this profile's mods: launching or running it, or between the Play click and the first status. */
export const lockedIn = (s: LaunchLocks, profileId: string) =>
  isLocked(s.status, profileId, s.starting ? s.startingProfile : '')

/** The launch state lockedIn reads, for a list that tests many profiles; stable while those three are. */
export const useLaunchLocks = (): LaunchLocks => {
  const status = useLaunch((s) => s.status)
  const starting = useLaunch((s) => s.starting)
  const startingProfile = useLaunch((s) => s.startingProfile)
  return useMemo(() => ({ status, starting, startingProfile }), [status, starting, startingProfile])
}

export const useProfileLocked = (profileId: string) => useLaunch((s) => lockedIn(s, profileId))

export const profileLocked = (profileId: string) => lockedIn(useLaunch.getState(), profileId)

export const useLocked = () => useProfileLocked(useProfiles((s) => s.openId))
