import { type Block, parseBBCode } from './bbcode.ts'

// A README is Markdown. Its headings, list items, links and bold are rewritten as the BBCode Nexus descriptions use,
// so one parser reduces both to text runs and no markup from a mod page reaches the DOM as HTML.

const FENCE = /^\s*(```|~~~)/
const HEADING = /^\s{0,3}#{1,6}\s+(.*?)\s*#*\s*$/
const ITEM = /^\s*(?:[-*+]|\d+[.)])\s+(.*)$/
const RULE = /^\s{0,3}([-*_])(\s*\1){2,}\s*$/
const IMAGE = /!\[[^\]]*\]\([^)]*\)/g
const LINK = /\[([^\]]+)\]\(\s*(\S+?)(?:\s+"[^"]*")?\s*\)/g
const BOLD = /(\*\*|__)(.+?)\1/g
const CODE = /`([^`]*)`/g
const QUOTE = /^\s*>\s?/
const LINE = /\r?\n/

const inline = (line: string) =>
  line
    .replace(IMAGE, '')
    .replace(LINK, '[url=$2]$1[/url]')
    .replace(BOLD, '[b]$2[/b]')
    .replace(CODE, '$1')
    .replace(QUOTE, '')

function markdownToBBCode(md: string): string {
  let code = false
  const out: string[] = []
  for (const line of md.split(LINE)) {
    const heading = HEADING.exec(line)
    const item = ITEM.exec(line)
    if (FENCE.test(line)) {
      code = !code
    } else if (code) {
      out.push(line)
    } else if (heading) {
      out.push(`[heading]${inline(heading[1] ?? '')}[/heading]`)
    } else if (RULE.test(line)) {
      out.push('[hr]')
    } else if (item) {
      out.push(`[*]${inline(item[1] ?? '')}`)
    } else {
      out.push(inline(line))
    }
  }
  return out.join('<br />')
}

function parseMarkdown(md: string): Block[] {
  return parseBBCode(markdownToBBCode(md))
}

export { parseMarkdown }
