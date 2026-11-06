export const PROFILE_CARD_LIMIT = 6

export function profileCardsSlice<T>(profiles: T[]) {
  const visible = profiles.slice(0, PROFILE_CARD_LIMIT)
  const more = Math.max(0, profiles.length - PROFILE_CARD_LIMIT)
  return { visible, more }
}
