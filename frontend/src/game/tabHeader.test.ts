import { expect, test } from 'bun:test'
import { notesFirstLine } from './tabHeader.ts'

test('notesFirstLine skips blank leading lines and trims', () => {
  expect(notesFirstLine('\n  \n  Co-op farm \nsecond')).toBe('Co-op farm')
  expect(notesFirstLine('')).toBe('')
})
