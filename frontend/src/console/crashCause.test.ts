import { expect, test } from 'bun:test'
import { crashCauseDetailLine, crashCauseKind } from './crashCause.ts'

test('crashCauseKind maps SMAPI reasons and crashCauseDetailLine keeps one line', () => {
  expect(crashCauseKind('missing-file')).toBe('missing-file')
  expect(crashCauseKind('asset-load')).toBe('asset-load')
  expect(crashCauseKind('mod-exception')).toBe('mod-exception')
  expect(crashCauseKind('other')).toBe('')
  expect(crashCauseDetailLine('first\nsecond')).toBe('first')
})
