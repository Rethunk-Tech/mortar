// Nexus descriptions are BBCode with HTML line breaks. They are reduced to paragraphs, headings and list items of
// plain text runs, which React renders as text, so no markup from a mod page ever reaches the DOM as HTML.

const HEADING_SIZE = 5
const HEX = 16
const DECIMAL = 10
const MAX_CODE_POINT = 0x10_ff_ff
const ENTITY = /&(#x[0-9a-f]+|#\d+|[a-z]+);/gi
const NEWLINE = /\r?\n/g
const BREAK = /<br\s*\/?>/gi
const HTML_TAG = /<[^>]*>/g
const WEB_URL = /^https?:\/\/[^\s"'<>]+$/i
const TAG = /\[(\/?)([a-z]+|\*)(?:=([^\]]*))?\]/gi

const entities: Record<string, string> = {
  amp: '&',
  lt: '<',
  gt: '>',
  quot: '"',
  apos: "'",
  nbsp: ' ',
}

const decode = (s: string) =>
  s.replace(ENTITY, (whole, name: string) => {
    if (name.startsWith('#')) {
      const hex = name[1] === 'x' || name[1] === 'X'
      const code = Number.parseInt(name.slice(hex ? 2 : 1), hex ? HEX : DECIMAL)
      return code > 0 && code <= MAX_CODE_POINT ? String.fromCodePoint(code) : ''
    }
    return entities[name.toLowerCase()] ?? whole
  })

// Tags whose content is not text to show.
const dropped = new Set(['img', 'youtube', 'video', 'media'])
const known = new Set([
  'b',
  'i',
  'u',
  's',
  'url',
  'size',
  'heading',
  'list',
  '*',
  'color',
  'font',
  'center',
  'left',
  'right',
  'justify',
  'quote',
  'spoiler',
  'code',
  'line',
  'email',
  'indent',
  'hr',
  ...dropped,
])

interface Open {
  name: string
  href?: string | undefined
  // For [url]address[/url]: where the link's own text starts.
  from?: number
}

// Id is the position, fixed once parsed, for React keys.
interface Run {
  id: number
  text: string
  bold?: boolean
  italic?: boolean
  href?: string
}

export interface Block {
  id: number
  kind: 'paragraph' | 'heading' | 'item'
  runs: Run[]
}

// Only web links are kept; anything else (javascript:, data:, file:) leaves its text unlinked.
export const safeUrl = (raw: string): string | undefined => {
  const url = raw.trim()
  return WEB_URL.test(url) ? url : undefined
}

// Removing a tag can join the text around it into a new one ("<<b>b>"), so it repeats until none is left.
const stripTags = (raw: string): string => {
  let text = raw
  for (let next = text.replace(HTML_TAG, ''); next !== text; next = text.replace(HTML_TAG, '')) {
    text = next
  }
  return text
}

export function parseBBCode(source: string): Block[] {
  const text = decode(stripTags(source.replace(NEWLINE, '').replace(BREAK, '\n')))
  const blocks: Block[] = []
  let block: Block = { id: 0, kind: 'paragraph', runs: [] }
  const stack: Open[] = []
  const has = (name: string) => stack.some((o) => o.name === name)
  const end = (next: Block['kind'] = 'paragraph') => {
    const runs = block.runs.filter((r) => r.text.trim() !== '').map((r, id) => ({ ...r, id }))
    if (runs.length > 0) {
      blocks.push({ id: blocks.length, kind: block.kind, runs })
    }
    block = { id: 0, kind: next, runs: [] }
  }
  const emit = (chunk: string) => {
    if (chunk === '' || stack.some((o) => dropped.has(o.name))) {
      return
    }
    const run: Run = { id: 0, text: chunk }
    if (has('b')) {
      run.bold = true
    }
    if (has('i')) {
      run.italic = true
    }
    const link = stack.findLast((o) => o.name === 'url')?.href
    if (link) {
      run.href = link
    }
    const last = block.runs.at(-1)
    if (
      last &&
      last.bold === run.bold &&
      last.italic === run.italic &&
      last.href === run.href &&
      !has('url')
    ) {
      last.text += chunk
    } else {
      block.runs.push(run)
    }
  }
  const write = (chunk: string) => {
    chunk.split('\n').forEach((line, i) => {
      if (i > 0) {
        end(block.kind === 'heading' ? 'heading' : 'paragraph')
      }
      emit(line)
    })
  }
  const close = (name: string) => {
    const at = stack.findLastIndex((o) => o.name === name)
    if (at < 0) {
      return
    }
    const [open] = stack.splice(at)
    if (open?.name === 'url' && open.from !== undefined) {
      const runs = block.runs.slice(open.from)
      const href = safeUrl(runs.map((r) => r.text).join(''))
      for (const r of runs) {
        if (href) {
          r.href = href
        }
      }
    }
    if (name === 'list' || ((name === 'size' || name === 'heading') && block.kind === 'heading')) {
      end()
    }
  }
  let at = 0
  const tags = [...text.matchAll(TAG)].filter((m) => known.has((m[2] ?? '').toLowerCase()))
  for (const m of tags) {
    const [whole, slash, rawName = '', arg] = m
    const name = rawName.toLowerCase()
    write(text.slice(at, m.index))
    at = m.index + whole.length
    if (slash) {
      close(name)
    } else if (name === '*') {
      end('item')
    } else if (name === 'line' || name === 'hr') {
      end()
    } else if (name === 'url') {
      stack.push(
        arg === undefined ? { name, from: block.runs.length } : { name, href: safeUrl(arg) },
      )
    } else {
      if ((name === 'size' && Number(arg) >= HEADING_SIZE) || name === 'heading') {
        end('heading')
      }
      stack.push({ name })
    }
  }
  write(text.slice(at))
  end()
  return blocks
}
