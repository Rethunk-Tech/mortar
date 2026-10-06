import { expect, test } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'

const SRC = join(import.meta.dir, '..')

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

function walk(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((ent) => {
    const p = join(dir, ent.name)
    if (ent.isDirectory()) {
      return walk(p)
    }
    if (/\.tsx?$/.test(ent.name) && !/\.test\.tsx?$/.test(ent.name)) {
      return [p]
    }
    return []
  })
}

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

const files = walk(SRC).map((path) => ({ path, src: readFileSync(path, 'utf8') }))

test('Lingui t is not taken as a helper parameter or argument', () => {
  const hits = files.flatMap(({ path, src }) => linguiTHits(path, src))
  expect(hits, hits.join('\n')).toEqual([])
})

test('zustand selectors do not allocate arrays or objects', () => {
  const hits = files.flatMap(({ path, src }) => selectorHits(path, strip(src)))
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
