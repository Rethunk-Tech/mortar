import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'TipIconButton.tsx'), 'utf8')

test('disabled state still wraps in a span so the tooltip can attach', () => {
  expect(src).toContain('<Tooltip title={label}>')
  expect(src).toContain('<span>')
  expect(src).toContain('disabled={disabled}')
  expect(src).toContain('size="small"')
  expect(src).not.toContain('width: 32')
})
