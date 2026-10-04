import { cmpText } from '../mods/cmpText.ts'

export interface OrderableProfile {
  id: string
  name: string
}

export function orderProfiles<T extends OrderableProfile>(
  profiles: T[],
  order: string,
  lastPlayedProfile = '',
): T[] {
  if (order === 'name') {
    return profiles.toSorted((a, b) => cmpText(a.name, b.name))
  }
  if (order === 'lastPlayed') {
    return profiles.toSorted((a, b) => {
      if (a.id === lastPlayedProfile && b.id !== lastPlayedProfile) {
        return -1
      }
      if (b.id === lastPlayedProfile && a.id !== lastPlayedProfile) {
        return 1
      }
      return 0
    })
  }
  return profiles
}
