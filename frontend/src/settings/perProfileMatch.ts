const WHITESPACE = /\s+/

// The labels among `labels` that a settings search for `query` finds: the query's words each appear in the label,
// ignoring case, so "ui sc" finds "UI scale".
export function matchPerProfile(query: string, labels: readonly string[]): string[] {
  const words = query.toLowerCase().split(WHITESPACE).filter(Boolean)
  if (words.length === 0) {
    return []
  }
  return labels.filter((label) => {
    const text = label.toLowerCase()
    return words.every((w) => text.includes(w))
  })
}
