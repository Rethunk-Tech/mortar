import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('toastError puts the mapped sentence in body and raw text in detail', () => {
  const src = readFileSync(join(import.meta.dir, 'report.ts'), 'utf8')
  expect(src).toContain('export function toastError')
  expect(src).toContain('export const reportError')
  expect(src).toContain('body: errorMessage(e)')
  expect(src).toContain("...(details === '' ? {} : { detail: details })")
  expect(src).toContain('toastError(i18n._(msg`Something went wrong`), e)')
})
