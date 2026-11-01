/** One frame of a .NET stack as a log prints it, with where its "Type.Method" sits in the line. */
export interface StackFrame {
  type: string
  method: string
  // How many arguments the frame lists, which tells overloads apart.
  params: number
  // The module version id a frame without symbols prints ("<mvid>:IL_002E"), '' when it printed a file instead.
  mvid: string
  start: number
  end: number
}

const NAME = '[\\w.<>`$+@\\[\\]-]+?'
// Mirrors internal/loader/dotnetstack: Harmony's wrapper first, as it names the method it stands in for.
const DMD = /DMD<([^>]+)::([^>]+)>/
const FORMATS = [
  // .NET and Mono: "  at Namespace.Type.Method (args) [0x00019] in <mvid>:0".
  new RegExp(`^(\\s+at (?:\\(wrapper [^)]*\\) )?)(${NAME})\\.([^.\\s(]+)(?:\\[[^\\]]*\\])? ?\\(`),
  // Unity after "Stack trace:": "Namespace.Type.Method (args) (at <mvid>:0)".
  new RegExp(`^()(${NAME})\\.([^.\\s(:]+)(?:\\[[^\\]]*\\])? ?\\(.*\\(at .*\\)\\s*$`),
  // Unity's own Debug.Log trace: "Namespace.Type:Method(args)".
  /^()([\w.<>`$+@[\]-]+):([^.\s(:]+) ?\(/,
]
const MVID = /<([0-9a-f]{32})>/

// Arguments in the first parenthesised list after at: commas outside generic brackets, plus one.
function countParams(line: string, at: number): number {
  const open = line.indexOf('(', at)
  let depth = 0
  let count = 0
  for (let i = open + 1; i < line.length; i++) {
    const c = line[i]
    if (c === ')' && depth === 0) {
      return count === 0 && line.slice(open + 1, i).trim() === '' ? 0 : count + 1
    }
    if (c === '<' || c === '[') {
      depth += 1
    } else if (c === '>' || c === ']') {
      depth -= 1
    } else if (c === ',' && depth === 0) {
      count += 1
    }
  }
  return 0
}

/** The frame a log line holds, or null for a line that is none. */
export function parseFrame(line: string): StackFrame | null {
  const mvid = MVID.exec(line)?.[1] ?? ''
  const dmd = DMD.exec(line)
  if (dmd) {
    const start = dmd.index + 'DMD<'.length
    return {
      type: dmd[1] ?? '',
      method: dmd[2] ?? '',
      params: countParams(line, dmd.index + dmd[0].length),
      mvid,
      start,
      end: start + (dmd[1] ?? '').length + 2 + (dmd[2] ?? '').length,
    }
  }
  for (const re of FORMATS) {
    const m = re.exec(line)
    if (m) {
      const [, lead = '', type = '', method = ''] = m
      const start = lead.length
      const end = start + type.length + 1 + method.length
      return { type, method, params: countParams(line, end), mvid, start, end }
    }
  }
  return null
}
