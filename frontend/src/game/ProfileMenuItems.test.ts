import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, '../bundles/applied.ts'), 'utf8')

test('bundle apply toasts use plural for the added count', () => {
  expect(src).toContain("plural(result.added, { one: '# mod added', other: '# mods added' })")
  expect(src.includes('result.added} mods added')).toBe(false)
})
