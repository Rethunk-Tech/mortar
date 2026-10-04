interface InstructionPart {
  // Offset in the line, a stable key for the part.
  at: number
  text: string
  url?: string
}

interface InstructionLine {
  // Offset in the whole text, a stable key for the line.
  at: number
  parts: InstructionPart[]
}

// [text](url) or a bare http(s) link; trailing sentence punctuation stays outside a bare link.
const LINK = /\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)|https?:\/\/[^\s<>[\]()]*[^\s<>[\]().,;:!?'"]/g

function parseLine(line: string): InstructionPart[] {
  const parts: InstructionPart[] = []
  let from = 0
  for (const match of line.matchAll(LINK)) {
    if (match.index > from) {
      parts.push({ at: from, text: line.slice(from, match.index) })
    }
    const [whole, label, target] = match
    parts.push(
      label === undefined
        ? { at: match.index, text: whole, url: whole }
        : { at: match.index, text: label, url: target ?? '' },
    )
    from = match.index + whole.length
  }
  if (from < line.length) {
    parts.push({ at: from, text: line.slice(from) })
  }
  return parts
}

/** The curator's text as lines of plain text and links; the app has no markdown renderer. */
export function parseInstructions(text: string): InstructionLine[] {
  let at = 0
  return text
    .replaceAll('\r\n', '\n')
    .trim()
    .split('\n')
    .map((line) => {
      const parsed = { at, parts: parseLine(line) }
      at += line.length + 1
      return parsed
    })
}

export type { InstructionLine, InstructionPart }
