import { expect, test } from 'bun:test'
import { configValueMessage } from './valueError.ts'

const refused = (detail: object) => new Error('[invalid] x', { cause: { kind: 'invalid', detail } })

test('a refused value is worded by its problem, any other error keeps its generic sentence', () => {
  const msg = (detail: object) => configValueMessage(refused(detail))
  expect(msg({ problem: 'whole_number' })).toBe('This needs to be a whole number.')
  expect(msg({ problem: 'range', min: 10, max: 300 })).toBe('This needs to be between 10 and 300.')
  expect(msg({ problem: 'range', min: -2_147_483_648, max: 2_147_483_647 })).toBe(
    'This needs to be between -2,147,483,648 and 2,147,483,647.',
  )
  expect(msg({ problem: 'range', max: 5 })).toBe('This can be at most 5.')
  expect(msg({ problem: 'range', min: 2 })).toBe('This can be at least 2.')
  for (const problem of ['line', 'list', 'empty', 'toml', 'color', 'bool', 'enum', 'number']) {
    expect(msg({ problem })).not.toBe('That request was not valid.')
  }
  expect(configValueMessage(new Error('[invalid] something else'))).toBe(
    'That request was not valid.',
  )
})
