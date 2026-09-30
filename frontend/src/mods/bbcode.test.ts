import { expect, test } from 'bun:test'
import { parseBBCode as parseWithIds, safeUrl } from './bbcode.ts'

// Ids are positions; the tests compare content.
const parseBBCode = (src: string) =>
  parseWithIds(src).map(({ kind, runs }) => ({ kind, runs: runs.map(({ id: _, ...r }) => r) }))

test('reads Nexus markup into headings, items and linked runs', () => {
  const src =
    'Press [font=Courier New]F1[/font] ([url=https://example.com/a]see[/url]).\n<br />\n<br />[size=6][b]Install[/b][/size][list=1]\n<br />[*][url]https://smapi.io[/url] first.\n<br />[*]Then &#39;[i]run[/i]&#39; &amp; play.\n<br />[/list]'
  expect(parseBBCode(src)).toEqual([
    {
      kind: 'paragraph',
      runs: [
        { text: 'Press F1 (' },
        { text: 'see', href: 'https://example.com/a' },
        { text: ').' },
      ],
    },
    { kind: 'heading', runs: [{ text: 'Install', bold: true }] },
    {
      kind: 'item',
      runs: [{ text: 'https://smapi.io', href: 'https://smapi.io' }, { text: ' first.' }],
    },
    {
      kind: 'item',
      runs: [{ text: "Then '" }, { text: 'run', italic: true }, { text: "' & play." }],
    },
  ])
})

test('never passes hostile markup or unsafe links through', () => {
  const src =
    '<script>alert(1)</script>[url=javascript:alert(2)]a[/url][url]javascript:alert(3)[/url][url=data:text/html,x]b[/url]<img src=x onerror=alert(4)>[img]https://x/y.png[/img][b][i][url=https://ok.test]c[/b]d[/url][/i]&lt;b&gt;'
  const runs = parseBBCode(src).flatMap((b) => b.runs)
  expect(runs.map((r) => r.text).join('')).toBe('alert(1)ajavascript:alert(3)bcd<b>')
  expect(runs.filter((r) => r.href).map((r) => r.href)).toEqual(['https://ok.test'])
  expect(runs.find((r) => r.text === 'c')).toEqual({
    text: 'c',
    bold: true,
    italic: true,
    href: 'https://ok.test',
  })
})

test('keeps unknown brackets as text and only web links as links', () => {
  expect(parseBBCode('[note] x [/b]')).toEqual([
    { kind: 'paragraph', runs: [{ text: '[note] x ' }] },
  ])
  expect(safeUrl(' https://a.b/c ')).toBe('https://a.b/c')
  expect(safeUrl('HTTP://a.b')).toBe('HTTP://a.b')
  expect(safeUrl('javascript:alert(1)')).toBeUndefined()
  expect(safeUrl('https://a.b/"onclick')).toBeUndefined()
})
