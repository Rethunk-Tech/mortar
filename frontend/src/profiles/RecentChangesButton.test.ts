import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('recent-change Undo is disabled while another restore is in flight', () => {
  const src = readFileSync(join(import.meta.dir, 'RecentChangesButton.tsx'), 'utf8')
  expect(src).toContain('disabled={thisBusy || otherBusy}')
})
