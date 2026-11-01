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

export const ESCAPE = /[.*+?^${}()|[\]\\]/g

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

// How many arguments an IL call to name passes, read from its parameter list (commas outside generic arguments), or -1
// when the line is no call to it (a field access, say).
export function ilArity(line: string, name: string): number {
  return arityAt(line, line.search(new RegExp(`::${name.replace(ESCAPE, '\\$&')}\\b`)))
}

// How many parameters a C# declaration of name lists, or -1 when the line declares no method called name.
export function csArity(line: string, name: string): number {
  return arityAt(
    line,
    line.search(new RegExp(`\\b${name.replace(ESCAPE, '\\$&')}\\s*(?:<[^>()]*>)?\\s*\\(`)),
  )
}

// The number of entries in the first parameter list after position at, counting commas outside nested <> and ().
function arityAt(line: string, at: number): number {
  const open = at < 0 ? -1 : line.indexOf('(', at)
  if (open < 0) {
    return -1
  }
  const list = line.slice(open + 1)
  if (list.trimStart().startsWith(')')) {
    return 0
  }
  let depth = 0
  let commas = 0
  for (const c of list) {
    if (c === '<' || c === '(') {
      depth += 1
    } else if (c === '>' || (c === ')' && depth > 0)) {
      depth -= 1
    } else if (c === ')') {
      break
    } else if (c === ',' && depth === 0) {
      commas += 1
    }
  }
  return commas + 1
}
