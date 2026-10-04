import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('game select shows LoadingRow while waiting and Retry after a load failure', () => {
  const src = readFileSync(join(import.meta.dir, 'GameSelect.tsx'), 'utf8')
  expect(src).toContain('<LoadingRow>{t`Loading…`}</LoadingRow>')
  expect(src).toContain('setLoadError(errorMessage(err))')
  expect(src).toContain('<LoadErrorRow')
})
