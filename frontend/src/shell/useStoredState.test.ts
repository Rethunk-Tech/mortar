import { expect, test } from 'bun:test'
import { readStored } from './useStoredState.ts'

const isText = (v: unknown): v is string => typeof v === 'string'

test('a stored string reads back whether it was written as JSON or bare', () => {
  localStorage.setItem('stored-test.json', JSON.stringify('run-1'))
  localStorage.setItem('stored-test.bare', 'run-2')
  expect(readStored('stored-test.json', '', isText)).toBe('run-1')
  expect(readStored('stored-test.bare', '', isText)).toBe('run-2')
  expect(readStored('stored-test.missing', 'none', isText)).toBe('none')
  localStorage.removeItem('stored-test.json')
  localStorage.removeItem('stored-test.bare')
})
