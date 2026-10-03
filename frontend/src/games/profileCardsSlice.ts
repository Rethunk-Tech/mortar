export const PROFILE_CARD_LIMIT = 6

export function profileCardsSlice<T>(profiles: T[], limit = PROFILE_CARD_LIMIT) {
  const visible = profiles.slice(0, limit)
  const more = Math.max(0, profiles.length - limit)
  return { visible, more }
}
