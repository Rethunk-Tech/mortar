import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'General.tsx'), 'utf8')

test('LAN port commit rejects 0 and shows the allowed range', () => {
  expect(src).toContain('port >= 1 && port <= maxLanPort')
  expect(src).toContain('setPortError(true)')
  expect(src).not.toContain('port >= 0 && port <= maxLanPort')
})
