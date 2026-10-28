import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { relative } from 'node:path'
import { SRC, srcFiles } from '../../test/srcFiles.ts'

const SKIP_HOOKS = new Set([
  'useCallback',
  'useContext',
  'useDebugValue',
  'useDeferredValue',
  'useEffect',
  'useId',
  'useImperativeHandle',
  'useInsertionEffect',
  'useLayoutEffect',
  'useLingui',
  'useMemo',
  'useReducer',
  'useRef',
  'useState',
  'useSyncExternalStore',
  'useTransition',
  'useMediaQuery',
])

function rel(path: string): string {
  return relative(SRC, path).replaceAll('\\', '/')
}

function lineOf(src: string, index: number): number {
  return src.slice(0, index).split('\n').length
}

function skipString(src: string, start: number): number {
  const q = src[start]
  let i = start + 1
  while (i < src.length) {
    if (src[i] === '\\') {
      i += 2
    } else if (src[i] === q) {
      return i + 1
    } else {
      i += 1
    }
  }
  return i
}

function skipLineComment(src: string, i: number): number {
  const nl = src.indexOf('\n', i)
  return nl === -1 ? src.length : nl + 1
}

function skipBlockComment(src: string, i: number): number {
  const end = src.indexOf('*/', i + 2)
  return end === -1 ? src.length : end + 2
}

function strip(src: string): string {
  let out = ''
  let i = 0
  while (i < src.length) {
    if (src.startsWith('//', i)) {
      i = skipLineComment(src, i)
      out += '\n'
    } else if (src.startsWith('/*', i)) {
      const next = skipBlockComment(src, i)
      out += src.slice(i, next).replace(/[^\n]/g, ' ')
      i = next
    } else if (src[i] === '"' || src[i] === "'" || src[i] === '`') {
      const next = skipString(src, i)
      out += ' '.repeat(next - i)
      i = next
    } else {
      out += src[i]
      i += 1
    }
  }
  return out
}

function matching(src: string, openAt: number, open: string, close: string): number {
  let depth = 0
  for (let i = openAt; i < src.length; i += 1) {
    const c = src[i]
    if (c === open) {
      depth += 1
    } else if (c === close) {
      depth -= 1
      if (depth === 0) {
        return i
      }
    }
  }
  return -1
}

function linguiTHits(path: string, src: string): string[] {
  const hits: string[] = []
  const file = rel(path)
  const typeRe = /ReturnType\s*<\s*typeof\s+useLingui\s*>\s*\[\s*['"]t['"]\s*\]/g
  for (const m of src.matchAll(typeRe)) {
    hits.push(`${file}:${lineOf(src, m.index ?? 0)}`)
  }
  const callRe = /\b([A-Za-z_$][\w$]*)\s*\(/g
  for (const m of src.matchAll(callRe)) {
    const [, name] = m
    const skip =
      name === undefined ||
      name === 't' ||
      name === 'msg' ||
      name === 'if' ||
      name === 'for' ||
      name === 'while' ||
      name === 'switch' ||
      name === 'catch'
    const open = m.index ?? 0
    const dotted = open > 0 && src[open - 1] === '.'
    if (!(skip || dotted)) {
      const paren = src.indexOf('(', open)
      const close = matching(src, paren, '(', ')')
      if (close !== -1) {
        const args = src.slice(paren + 1, close)
        if (splitArgs(args).some((a) => a.trim() === 't')) {
          hits.push(`${file}:${lineOf(src, open)}`)
        }
      }
    }
  }
  return hits
}

function splitArgs(args: string): string[] {
  const parts: string[] = []
  let depth = 0
  let start = 0
  for (let i = 0; i < args.length; i += 1) {
    const c = args[i]
    if (c === '(' || c === '{' || c === '[') {
      depth += 1
    } else if (c === ')' || c === '}' || c === ']') {
      depth -= 1
    } else if (c === ',' && depth === 0) {
      parts.push(args.slice(start, i))
      start = i + 1
    }
  }
  parts.push(args.slice(start))
  return parts
}

function selectorHits(path: string, src: string): string[] {
  const hits: string[] = []
  const file = rel(path)
  const re = /\b(use[A-Z]\w*)\s*\(\s*\(\s*s\s*\)\s*=>/g
  for (const m of src.matchAll(re)) {
    const [, hook] = m
    if (hook !== undefined && !SKIP_HOOKS.has(hook)) {
      const arrow = src.indexOf('=>', m.index ?? 0)
      let i = arrow + 2
      while (src[i] === ' ' || src[i] === '\n' || src[i] === '\t' || src[i] === '\r') {
        i += 1
      }
      let body = ''
      if (src[i] === '{') {
        const end = matching(src, i, '{', '}')
        if (end !== -1) {
          body = src.slice(i, end + 1)
        }
      } else {
        const callOpen = src.indexOf('(', m.index ?? 0)
        const callClose = matching(src, callOpen, '(', ')')
        if (callClose !== -1) {
          body = src.slice(i, callClose)
        }
      }
      if (body !== '' && unstableSelector(body)) {
        hits.push(`${file}:${lineOf(src, m.index ?? 0)}`)
      }
    }
  }
  return hits
}

function unstableSelector(body: string): boolean {
  if (/\.\s*(map|filter|slice|concat|sort)\s*\(/.test(body)) {
    return true
  }
  if (/\?\?\s*(?:\[\]|\{\})/.test(body)) {
    return true
  }
  const trimmed = body.trim()
  if (trimmed.startsWith('({') || trimmed.startsWith('[')) {
    return true
  }
  return /\breturn\s*(\{|\[)/.test(body)
}

const files = srcFiles().map((path) => ({ path, src: readFileSync(path, 'utf8') }))

test('Lingui t is not taken as a helper parameter or argument', () => {
  const hits = files.flatMap(({ path, src }) => linguiTHits(path, src))
  expect(hits, hits.join('\n')).toEqual([])
})

// MUI's Tooltip renders itself and its child twice on mount; shell/Tooltip.tsx mounts it only while it shows.
test('tooltips come from shell/Tooltip.tsx', () => {
  const hits = files
    .filter(({ path }) => rel(path) !== 'shell/Tooltip.tsx')
    .filter(({ src }) => /import \{[^}]*\bTooltip\b[^}]*\} from '@mui\/material'/.test(src))
    .map(({ path }) => rel(path))
  expect(hits, hits.join('\n')).toEqual([])
})

test('zustand selectors do not allocate arrays or objects', () => {
  const hits = files.flatMap(({ path, src }) => selectorHits(path, strip(src)))
  expect(hits, hits.join('\n')).toEqual([])
})

// A store hook called without a selector re-renders its component on every change anywhere in that store.
test('zustand store hooks take a selector', () => {
  const stores = new Set(
    files.flatMap(({ src }) =>
      [...src.matchAll(/\b(use[A-Z]\w*)\s*=\s*create\b/g)].map((m) => m[1]),
    ),
  )
  const hits = files.flatMap(({ path, src }) => {
    const code = strip(src)
    return [...code.matchAll(/\b(use[A-Z]\w*)\(\s*\)/g)]
      .filter((m) => stores.has(m[1]))
      .map((m) => `${rel(path)}:${lineOf(code, m.index ?? 0)}`)
  })
  expect(hits, hits.join('\n')).toEqual([])
})

// Plurals and lists go through plural() and listNames(), so a translation can follow its own grammar.
function copyHits(path: string, src: string): string[] {
  const hits: string[] = []
  for (const m of src.matchAll(/\b(?:t|msg)`((?:[^`\\]|\\.)*)`/g)) {
    const body = m[1] ?? ''
    const counted = /\$\{[^}]*(?:\.length|[cC]ount|\bn)\}\s+[a-z]+s\b/.test(body)
    if (counted || body.includes('.join(')) {
      hits.push(`${rel(path)}:${lineOf(src, m.index ?? 0)}`)
    }
  }
  return hits
}

test('messages do not pre-render plurals or join lists in English', () => {
  const hits = files.flatMap(({ path, src }) => copyHits(path, src))
  expect(hits, hits.join('\n')).toEqual([])
})

// Tooltip names its child with aria-label, which a bare span may not carry; describeChild describes it instead.
test('a Tooltip around a bare span describes it rather than naming it', () => {
  const hits = files.flatMap(({ path, src }) =>
    [...src.matchAll(/<Tooltip\b((?:[^>]|=>)*?)>\s*<span>/g)]
      .filter((m) => !(m[1] ?? '').includes('describeChild'))
      .map((m) => `${rel(path)}:${lineOf(src, m.index ?? 0)}`),
  )
  expect(hits, hits.join('\n')).toEqual([])
})

const hitsOf = (pick: (file: string, src: string) => number[]) =>
  files.flatMap(({ path, src }) =>
    pick(rel(path), src).map((at) => `${rel(path)}:${lineOf(src, at)}`),
  )

// AppDialog, ConfirmationDialog, FormDialog and BareDialog carry the dialogs' frame, so a dialog never builds its own.
test('dialogs are built on shell/dialog', () => {
  const hits = hitsOf((file, src) =>
    file.startsWith('shell/dialog/')
      ? []
      : [...src.matchAll(/import \{[^}]*\bDialog\b[^}]*\} from '@mui\/material'/g)].map(
          (m) => m.index ?? 0,
        ),
  )
  expect(hits, hits.join('\n')).toEqual([])
})

// A menu's rows are MenuAction, MenuCheck and MenuNote, its headings MenuHeading and its dividers MenuRule, so every
// menu lines up, sizes its icons alike and reads the same to a screen reader.
test('menus are built from the shared menu rows', () => {
  const raw = /<(?:MenuItem|ListItemIcon|ListItemText|ListSubheader|Divider|Checkbox)\b/g
  const hits = hitsOf((file, src) => {
    if (file.startsWith('shell/')) {
      return []
    }
    const code = strip(src)
    return [...code.matchAll(/<Menu\b/g)].flatMap((open) => {
      const start = open.index ?? 0
      const end = code.indexOf('</Menu>', start)
      const body = code.slice(start, end === -1 ? undefined : end)
      return [...body.matchAll(raw)].map((m) => start + (m.index ?? 0))
    })
  })
  expect(hits, hits.join('\n')).toEqual([])
})

// Legibility over art comes from a solid layer with alpha; an image's edge may fade only through an alpha mask.
test('no gradients', () => {
  const hits = hitsOf((_, src) =>
    [...src.matchAll(/\b(?:linear|radial|conic)-gradient\(/g)]
      .filter((m) => !/mask/i.test(src.slice(Math.max(0, (m.index ?? 0) - 40), m.index)))
      .map((m) => m.index ?? 0),
  )
  expect(hits, hits.join('\n')).toEqual([])
})

// Icons are Lucide; only the brand's own marks draw SVG by hand.
test('icons come from Lucide', () => {
  const hits = hitsOf((file, src) => [
    ...[
      ...src.matchAll(
        /from '(?:@mui\/icons-material|react-icons|@tabler\/icons-react|@heroicons\/react)[^']*'/g,
      ),
    ].map((m) => m.index ?? 0),
    ...(file.startsWith('brand/')
      ? []
      : [...strip(src).matchAll(/<svg\b/g)].map((m) => m.index ?? 0)),
  ])
  expect(hits, hits.join('\n')).toEqual([])
})

// The compact Alert's size and colour come from the variant; its sx places it and nothing more.
test('a compact Alert takes its size and colour from the theme', () => {
  const styled =
    /\b(?:fontSize|color|bgcolor|backgroundColor|height|minHeight|p|px|py|pt|pb|pl|pr|padding|border)\s*:/
  const hits = hitsOf((_, src) =>
    [...src.matchAll(/<Alert\b(?:[^>]|=>)*?variant="compact"(?:[^>]|=>)*?>/g)]
      .filter((m) => {
        const sx = m[0].match(/\bsx=\{\{([\s\S]*?)\}\}/)
        return sx !== null && styled.test(sx[1] ?? '')
      })
      .map((m) => m.index ?? 0),
  )
  expect(hits, hits.join('\n')).toEqual([])
})

// A MUI unit is 8px, so 1 is the Comfortable gap token and 2 the pad and gutter tokens.
const LITERAL_SPACING =
  /\b(?:(?:gap|rowGap|columnGap|mt|mb): *(?:1|'8px')|(?:p|px|py|pt|pb|padding|paddingTop|paddingBottom): *(?:2|'16px'))(?![\d.\w])/g

test('layout spacing reads the density tokens, not 8px or 16px literals', () => {
  const hits: string[] = []
  for (const path of srcFiles().filter((p) => p.endsWith('.tsx') && !rel(p).startsWith('theme/'))) {
    const name = rel(path)
    const src = readFileSync(path, 'utf8')
    for (const m of src.matchAll(LITERAL_SPACING)) {
      hits.push(`${name}:${lineOf(src, m.index)} ${m[0]}`)
    }
  }
  expect(hits).toEqual([])
})

// "…" marks a control that opens further input; the dialog it opens is titled without it.
test('a title prop does not end in an ellipsis', () => {
  const hits = files.flatMap(({ path, src }) =>
    [...src.matchAll(/\btitle=\{t`[^`]*…`\}/g)].map((m) => `${rel(path)}:${lineOf(src, m.index)}`),
  )
  expect(hits, hits.join('\n')).toEqual([])
})
