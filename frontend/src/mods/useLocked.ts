import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { isLocked } from './locked.ts'

export const useLocked = () => {
  const status = useLaunch((s) => s.status)
  const startingProfile = useLaunch((s) => (s.starting ? s.startingProfile : ''))
  const openId = useProfiles((s) => s.openId)
  return isLocked(status, openId, startingProfile)
}
