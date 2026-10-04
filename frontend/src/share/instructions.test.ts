import { expect, test } from 'bun:test'
import { parseInstructions } from './instructions.ts'

test('markdown links and bare urls become links, the rest stays text', () => {
  const lines = parseInstructions(
    'Read [the guide](https://a.test/x) first.\r\nSee https://b.test/y, then go.',
  )
  expect(lines.map((l) => l.parts.map(({ text, url }) => [text, url]))).toEqual([
    [
      ['Read ', undefined],
      ['the guide', 'https://a.test/x'],
      [' first.', undefined],
    ],
    [
      ['See ', undefined],
      ['https://b.test/y', 'https://b.test/y'],
      [', then go.', undefined],
    ],
  ])
})

test('non-http targets are left as text and blank lines are kept', () => {
  const lines = parseInstructions('[x](javascript:alert(1))\n\nend')
  expect(lines.map((l) => l.parts.map((p) => p.text))).toEqual([
    ['[x](javascript:alert(1))'],
    [],
    ['end'],
  ])
  expect(lines.map((l) => l.at)).toEqual([0, 25, 26])
})
