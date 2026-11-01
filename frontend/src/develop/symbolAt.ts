const IDENTIFIER = /[A-Za-z_][A-Za-z0-9_]*/g

// The identifier in text that spans offset, or '' when offset sits on punctuation or space. An offset just past a
// name's last letter still counts, as a click on the right half of that letter lands there.
export function identifierAt(text: string, offset: number): string {
  for (const m of text.matchAll(IDENTIFIER)) {
    if (offset >= m.index && offset <= m.index + m[0].length) {
      return m[0]
    }
  }
  return ''
}

// The identifier under a point on the page, read from the text node the browser places the caret in there.
export function symbolAt(x: number, y: number): string {
  const range = document.caretRangeFromPoint?.(x, y)
  const node = range?.startContainer
  if (!range || node?.nodeType !== Node.TEXT_NODE) {
    return ''
  }
  return identifierAt(node.textContent ?? '', range.startOffset)
}
