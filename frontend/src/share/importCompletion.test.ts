import { expect, test } from 'bun:test'
import { watchBatch } from './importCompletion.ts'

test('counts a settled batch by queue outcome', () => {
  const watch = watchBatch(['a', 'b', 'c'])
  expect(
    watch([
      { id: 'a', state: 'done' },
      { id: 'b', state: 'failed' },
      { id: 'c', state: 'cancelled' },
    ]),
  ).toEqual({ installed: 1, failed: 1, skipped: 1 })
})

test('waits for items the queue has not listed yet', () => {
  const watch = watchBatch(['a', 'b'])
  expect(watch([{ id: 'a', state: 'done' }])).toBeUndefined()
})

test('an item the queue trimmed after it finished keeps its outcome', () => {
  const watch = watchBatch(['a', 'b', 'c'])
  expect(
    watch([
      { id: 'a', state: 'done' },
      { id: 'b', state: 'downloading' },
      { id: 'c', state: 'queued' },
    ]),
  ).toBeUndefined()
  expect(
    watch([
      { id: 'b', state: 'done' },
      { id: 'c', state: 'downloading' },
    ]),
  ).toBeUndefined()
  // c was removed unfinished; a and b were trimmed as done.
  expect(watch([])).toEqual({ installed: 2, failed: 0, skipped: 1 })
})

test('an item joined from an earlier batch counts once', () => {
  const watch = watchBatch(['a', 'a'])
  expect(watch([{ id: 'a', state: 'done' }])).toEqual({ installed: 1, failed: 0, skipped: 0 })
})
