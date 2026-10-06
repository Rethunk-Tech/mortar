import { expect, test } from 'bun:test'
import { parseMarkdown } from './markdown.ts'

test('a README becomes headings, items and links of plain text', () => {
  const blocks = parseMarkdown(
    '# Title\n\nSome **bold** and a [link](https://x.test/a).\n![img](https://x.test/i.png)\n- one\n- two\n```\n# not a heading\n```\n<script>alert(1)</script>',
  )
  expect(blocks.map((b) => [b.kind, b.runs.map((r) => r.text).join('')])).toEqual([
    ['heading', 'Title'],
    ['paragraph', 'Some bold and a link.'],
    ['item', 'one'],
    ['item', 'two'],
    ['paragraph', '# not a heading'],
    ['paragraph', 'alert(1)'],
  ])
  const link = blocks[1]?.runs.find((r) => r.href)
  expect(link).toMatchObject({ text: 'link', href: 'https://x.test/a' })
  expect(blocks[1]?.runs.find((r) => r.bold)?.text).toBe('bold')
})

test('a link to anything but the web stays text', () => {
  const [block] = parseMarkdown('[x](javascript:alert(1))')
  expect(block?.runs.every((r) => r.href === undefined)).toBe(true)
})
