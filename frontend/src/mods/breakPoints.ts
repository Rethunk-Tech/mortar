const AFTER_SEPARATOR = /(?<=[.,/_-])/

// Pieces of a long id, each keyed by its offset: the id wraps after its dots and dashes, and only mid-word when one word alone overflows.
export function breakPoints(text: string): { at: number; part: string }[] {
  let at = 0
  return text.split(AFTER_SEPARATOR).map((part) => {
    const piece = { at, part }
    at += part.length
    return piece
  })
}
