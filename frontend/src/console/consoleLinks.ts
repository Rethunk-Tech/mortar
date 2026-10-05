import { localId } from '../mods/dependents.ts'

const WORD_CHAR = /[A-Za-z0-9_]/
const TRAILING_PUNCT = /[.,;:)\]]$/
const ROOT_TRAILING_SEP = /[/\\]+$/

type ConsoleLink =
  | { kind: 'mod'; start: number; end: number; id: string }
  | { kind: 'path'; start: number; end: number; path: string }

function overlap(aStart: number, aEnd: number, bStart: number, bEnd: number): boolean {
  return aStart < bEnd && bStart < aEnd
}

function insertNonOverlapping(into: ConsoleLink[], link: ConsoleLink): void {
  const len = link.end - link.start
  const overlapping = into.filter((e) => overlap(e.start, e.end, link.start, link.end))
  const longest = overlapping.reduce((n, e) => Math.max(n, e.end - e.start), 0)
  if (overlapping.length > 0 && len <= longest) {
    return
  }
  const kept = into.filter((e) => !overlap(e.start, e.end, link.start, link.end))
  into.length = 0
  into.push(...kept, link)
}

function isTokenBoundary(text: string, start: number, end: number): boolean {
  const before = start === 0 ? '' : text[start - 1]
  const after = end >= text.length ? '' : text[end]
  if (before !== undefined && before !== '' && WORD_CHAR.test(before)) {
    return false
  }
  if (after !== undefined && after !== '' && WORD_CHAR.test(after)) {
    return false
  }
  return true
}

function needlesFor(root: string): string[] {
  const trimmed = root.replace(ROOT_TRAILING_SEP, '')
  if (trimmed.length === 0) {
    return []
  }
  const posix = trimmed.replaceAll('\\', '/')
  const win = posix.replaceAll('/', '\\')
  return posix === win ? [posix] : [posix, win]
}

function pathSegmentChar(ch: string): boolean {
  return (
    ch !== '"' &&
    ch !== "'" &&
    ch !== '<' &&
    ch !== '>' &&
    ch !== '|' &&
    ch !== ' ' &&
    ch !== '\n' &&
    ch !== '\r'
  )
}

function extendPath(text: string, minEnd: number): number {
  let end = minEnd
  for (;;) {
    const sep = text[end]
    if (sep !== '/' && sep !== '\\') {
      break
    }
    let i = end + 1
    while (i < text.length && pathSegmentChar(text[i] ?? '')) {
      i += 1
    }
    if (i === end + 1) {
      break
    }
    end = i
  }
  while (end > minEnd && TRAILING_PUNCT.test(text[end - 1] ?? '')) {
    end -= 1
  }
  return end
}

interface InstalledMod {
  name: string
  id: string
}

interface ConsoleLinkRoots {
  modsDir: string
  gameDir: string
}

function linksInText(
  text: string,
  mods: readonly InstalledMod[],
  roots: ConsoleLinkRoots,
): ConsoleLink[] {
  const found: ConsoleLink[] = []
  const names = mods
    .flatMap((m) => [
      { key: localId(m.id), id: m.id },
      { key: m.name, id: m.id },
    ])
    .filter((m) => m.key.length > 0)
    .sort((a, b) => b.key.length - a.key.length)

  const seen = new Set<string>()
  for (const n of names) {
    const id = `${n.id}\0${n.key}`
    if (!seen.has(id)) {
      seen.add(id)
      let from = 0
      for (;;) {
        const i = text.indexOf(n.key, from)
        if (i < 0) {
          break
        }
        const end = i + n.key.length
        if (isTokenBoundary(text, i, end)) {
          insertNonOverlapping(found, {
            kind: 'mod',
            start: i,
            end,
            id: n.id,
          })
        }
        from = i + 1
      }
    }
  }

  for (const root of [roots.modsDir, roots.gameDir]) {
    for (const needle of needlesFor(root)) {
      let from = 0
      for (;;) {
        const i = text.indexOf(needle, from)
        if (i < 0) {
          break
        }
        const end = extendPath(text, i + needle.length)
        insertNonOverlapping(found, {
          kind: 'path',
          start: i,
          end,
          path: text.slice(i, end),
        })
        from = i + 1
      }
    }
  }

  found.sort((a, b) => a.start - b.start)
  return found
}

function linksForModColumn(modColumn: string, mods: readonly InstalledMod[]): ConsoleLink[] {
  const key = modColumn.trim()
  if (key.length === 0) {
    return []
  }
  const match = mods.find((m) => m.name === key || localId(m.id) === key)
  if (match === undefined) {
    return []
  }
  const start = modColumn.indexOf(key)
  return [{ kind: 'mod', start, end: start + key.length, id: match.id }]
}

export type { ConsoleLink, ConsoleLinkRoots, InstalledMod }
export { linksForModColumn, linksInText }
