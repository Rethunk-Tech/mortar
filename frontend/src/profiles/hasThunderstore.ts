/** Whether a game's catalog entry lists Thunderstore among its mod sources. */
export function hasThunderstore(game: { sources: string[] | null } | null | undefined): boolean {
  return game?.sources?.includes('thunderstore') === true
}
