import { expect, test } from 'bun:test'
import { configValueMessage } from './valueError.ts'

test('a refused value is worded by its problem, any other error keeps its generic sentence', () => {
  const msg = (text: string) => configValueMessage(new Error(`[invalid] ${text}`))
  expect(msg('config_value:whole_number Speed expects a whole number')).toBe(
    'This needs to be a whole number.',
  )
  expect(msg('config_value:range:10:300 Delay must be between 10 and 300')).toBe(
    'This needs to be between 10 and 300.',
  )
  expect(msg('config_value:range:any:5 x')).toBe('This can be at most 5.')
  expect(msg('config_value:range:2:any x')).toBe('This can be at least 2.')
  expect(msg('something else')).toBe('That request was not valid.')
})
