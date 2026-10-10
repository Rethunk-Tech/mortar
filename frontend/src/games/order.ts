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
    .sort((a, b) => typedKey(a.name).localeCompare(typedKey(b.name)))
  return { recent, rest }
}

const NOT_TYPED = /[^\p{L}\p{N}]/gu
const DIGIT = /\p{N}/u

// A name as someone types it: letters and digits only, lower case, accents dropped, so "R.E.P.O." is "repo".
function typedKey(name: string): string {
  return name.normalize('NFD').replace(NOT_TYPED, '').toLowerCase()
}

// The letter a game is filed under; every name that starts with a digit shares "#".
function initialOf(name: string): string {
  const first = typedKey(name).charAt(0)
  return DIGIT.test(first) ? '#' : first.toUpperCase()
}

// The first game whose name starts with what was typed, else the first whose name holds it.
function typedMatch<G extends Listed>(games: G[], typed: string): G | undefined {
  const want = typedKey(typed)
  if (!want) {
    return undefined
  }
  return (
    games.find((g) => typedKey(g.name).startsWith(want)) ??
    games.find((g) => typedKey(g.name).includes(want))
  )
}

export { gameOrder, initialOf, typedKey, typedMatch }
