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

const ESCAPE = /[.*+?^${}()|[\]\\]/g

// The type an IL line names as the owner of name ("Namespace.Outer+Inner", as the member search names types), or ''.
// A call, field load or store writes its target as [Assembly]Namespace.Type::Member, a nested type with '/' and a
// generic one with its arguments in angle brackets.
export function ilOwner(line: string, name: string): string {
  const m = line.match(
    new RegExp(`([A-Za-z_][\\w.\`/]*)(?:<[^\\s]*>)?::${name.replace(ESCAPE, '\\$&')}\\b`),
  )
  return m?.[1]?.replaceAll('/', '+') ?? ''
}

// A name right-clicked in a code view: where, on which line (from 0) and that line's text, in C# or IL.
export interface SymbolClick {
  name: string
  x: number
  y: number
  line: number
  text: string
  il: boolean
}
