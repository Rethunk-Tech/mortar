const NOT_TYPED = /[^\p{L}\p{N}]/gu

/** A pause this long starts a new search instead of adding to the last one. */
export const TYPE_RESET_MS = 1000

// A name as someone types it: letters and digits only, lower case, accents dropped, so "R.E.P.O." is "repo".
export function typedKey(name: string): string {
  return name.normalize('NFD').replace(NOT_TYPED, '').toLowerCase()
}

// The first item whose name starts with what was typed, else the first whose name holds it.
export function typedMatch<T>(
  items: readonly T[],
  typed: string,
  nameOf: (item: T) => string,
): T | undefined {
  const want = typedKey(typed)
  if (!want) {
    return undefined
  }
  return (
    items.find((item) => typedKey(nameOf(item)).startsWith(want)) ??
    items.find((item) => typedKey(nameOf(item)).includes(want))
  )
}
