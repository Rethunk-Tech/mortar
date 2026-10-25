// A line of decompiled code split into coloured runs. Decompiled C# and ILSpy's IL keep comments to one line, so each
// line is read on its own.

export type TokenKind =
  | 'plain'
  | 'comment'
  | 'string'
  | 'keyword'
  | 'number'
  | 'type'
  | 'label'
  | 'directive'

export interface Token {
  kind: TokenKind
  text: string
  // Where the run starts in the line: its key, since runs never overlap.
  start: number
}

const CS_KEYWORDS = new Set(
  (
    'abstract as base bool break byte case catch char checked class const continue decimal default delegate do ' +
    'double else enum event explicit extern false finally fixed float for foreach goto if implicit in int interface ' +
    'internal is lock long namespace new null object operator out override params private protected public readonly ' +
    'ref return sbyte sealed short sizeof stackalloc static string struct switch this throw true try typeof uint ' +
    'ulong unchecked unsafe ushort using var virtual void volatile while get set value yield async await nameof'
  ).split(' '),
)

const IL_KEYWORDS = new Set(
  (
    'public private family assembly static instance class valuetype void bool int8 int16 int32 int64 uint8 uint16 ' +
    'uint32 uint64 float32 float64 string object native int char hidebysig specialname rtspecialname virtual final ' +
    'newslot abstract sealed cil managed extends implements auto ansi beforefieldinit nested init initonly literal ' +
    'default'
  ).split(' '),
)

// One rule per run, tried in order at each position; the first that matches there wins.
const RULES: { kind: TokenKind; re: RegExp }[] = [
  { kind: 'comment', re: /\/\/.*/y },
  { kind: 'string', re: /@?"(?:[^"\\]|\\.)*"?/y },
  { kind: 'string', re: /'(?:[^'\\]|\\.)*'?/y },
  { kind: 'label', re: /IL_[0-9a-f]{4}:?/y },
  { kind: 'directive', re: /\.[a-z][a-z0-9_.]*/y },
  { kind: 'number', re: /(?:0x[0-9a-fA-F]+|\d+(?:\.\d+)?)[fFdDmMuUlL]*/y },
  { kind: 'plain', re: /[A-Za-z_@][A-Za-z0-9_`]*/y },
  { kind: 'plain', re: /\s+/y },
  { kind: 'plain', re: /./y },
]

const IDENT = /^[A-Za-z_@][A-Za-z0-9_`]*$/
const OPCODE = /^[a-z][a-z0-9]*(\.[a-z0-9]+)*$/
// Decompiled C# capitalises type and member names; its keywords are lower case.
const CAPITAL = /^[A-Z]/

/** Splits one line into runs: C#'s keywords and capitalised names (types), or IL's labels, opcodes and directives. */
export function tokenize(line: string, il: boolean): Token[] {
  const out: Token[] = []
  let afterLabel = false
  for (let at = 0; at < line.length; ) {
    for (const { kind, re } of RULES) {
      re.lastIndex = at
      const m = re.exec(line)
      if (!m || m[0] === '') {
        continue
      }
      const text = m[0]
      let k = kind
      if (kind === 'directive' && !il) {
        k = 'plain'
      } else if (kind === 'plain' && IDENT.test(text)) {
        k = wordKind(text, il, afterLabel)
      }
      if (kind === 'label') {
        afterLabel = true
      } else if (text.trim() !== '') {
        afterLabel = false
      }
      push(out, k, text, at)
      at += text.length
      break
    }
  }
  return out
}

function wordKind(word: string, il: boolean, afterLabel: boolean): TokenKind {
  if (il) {
    if (afterLabel && OPCODE.test(word)) {
      return 'keyword'
    }
    return IL_KEYWORDS.has(word) ? 'keyword' : 'plain'
  }
  if (CS_KEYWORDS.has(word)) {
    return 'keyword'
  }
  return CAPITAL.test(word) ? 'type' : 'plain'
}

// Neighbouring runs of one kind join, and spaces join the run before them, so a line is a handful of spans.
function push(out: Token[], kind: TokenKind, text: string, start: number) {
  const last = out.at(-1)
  if (last && (last.kind === kind || text.trim() === '')) {
    last.text += text
  } else {
    out.push({ kind, text, start })
  }
}
