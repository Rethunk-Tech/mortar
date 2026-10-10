interface Listed {
  id: string
  name: string
}

const RECENT_GAMES = 3
// Sorts after every timestamp, so the open game leads the recent ones before its own play is recorded.
const NEWEST = '￿'

// The games used most recently, newest first, and the rest by name. A game counts as used once it has been opened
// (lastGame) or played.
function gameOrder<G extends Listed>(
  games: G[],
  lastGame: string,
  played: Record<string, { at?: string } | undefined> | null | undefined,
): { recent: G[]; rest: G[] } {
  const at = (id: string) => (id === lastGame ? NEWEST : (played?.[id]?.at ?? ''))
  const recent = games
    .filter((g) => at(g.id) !== '')
    .sort((a, b) => at(b.id).localeCompare(at(a.id)))
    .slice(0, RECENT_GAMES)
  const rest = games
    .filter((g) => !recent.includes(g))
    .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
  return { recent, rest }
}

export { gameOrder }
