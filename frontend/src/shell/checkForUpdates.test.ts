import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'checkForUpdates.ts'), 'utf8')

test('checking toast is dismissed before the result toast is pushed', () => {
  expect(src).toContain('toasts.dismiss(checking)')
  expect(src.indexOf('toasts.dismiss(checking)')).toBeLessThan(src.lastIndexOf('toasts.push'))
})
