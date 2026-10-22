import { expect, test } from 'bun:test'
import { crashCauseDetailLine } from './crashCause.ts'

test('crashCauseDetailLine keeps one line', () => {
  expect(crashCauseDetailLine('first\nsecond')).toBe('first')
})
