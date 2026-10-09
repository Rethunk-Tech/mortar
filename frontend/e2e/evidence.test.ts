import { expect, test } from 'bun:test'
import { describeExits, shouldKeepEvidence } from './evidence.ts'

test('evidence is kept for a failed test or a server that stopped, and not for a clean run', () => {
  expect(shouldKeepEvidence({ failed: false, alive: true, listening: true })).toBe(false)
  expect(shouldKeepEvidence({ failed: true, alive: true, listening: true })).toBe(true)
  expect(shouldKeepEvidence({ failed: false, alive: false, listening: true })).toBe(true)
  expect(shouldKeepEvidence({ failed: false, alive: true, listening: false })).toBe(true)
})

test('exit records read as statuses and signals', () => {
  expect(describeExits('12 0\n13 -9\n\n14 2\n')).toEqual([
    'pid 12 exited with status 0',
    'pid 13 killed by signal 9',
    'pid 14 exited with status 2',
  ])
})
