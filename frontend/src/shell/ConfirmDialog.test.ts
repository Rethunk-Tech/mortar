import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'ConfirmDialog.tsx'), 'utf8')

test('confirmDisabled turns off only the confirm button', () => {
  expect(src).toContain('confirmDisabled?: boolean | undefined')
  expect(src).toContain('disabled={busy || confirmDisabled}')
  expect(src).toContain('<Button onClick={onCancel} disabled={busy}')
})
