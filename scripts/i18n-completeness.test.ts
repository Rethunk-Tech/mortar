import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { generate } from './i18n-completeness.ts'

test('the committed translation counts match the catalogs', async () => {
  const committed = readFileSync(
    join(import.meta.dir, '../frontend/src/locales/completeness.ts'),
    'utf8',
  )
  expect(committed).toBe(await generate())
})
