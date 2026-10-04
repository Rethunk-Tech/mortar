import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'ToolsManageDialog.tsx'), 'utf8')

test('deleting a tool asks for confirmation before remove', () => {
  expect(src).toContain('ConfirmDialog')
  expect(src).toContain('color="error"')
  expect(src).not.toContain('remove(game, tool.id).catch')
})
