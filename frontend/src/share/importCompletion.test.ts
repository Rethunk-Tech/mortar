import { expect, test } from 'bun:test'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { settledImportCounts } from './importCompletion.ts'

test('counts a settled import by queue outcome', () => {
  const got = settledImportCounts(
    [
      { batchId: 'batch-1', state: 'done' },
      { batchId: 'batch-1', state: 'failed' },
      { batchId: 'batch-1', state: 'cancelled' },
    ] as Item[],
    'batch-1',
    3,
  )
  expect(got).toEqual({ installed: 1, failed: 1, skipped: 1 })
})
