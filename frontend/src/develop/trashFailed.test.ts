import { expect, test } from 'bun:test'
import { inUseHolders } from './trashFailed.ts'

test('the holders come from the in_use cause detail, and nothing else yields any', () => {
  const e = new Error('x', {
    cause: { kind: 'in_use', detail: { folder: 'C:\\m', holders: ['dotnet', 'Code'] } },
  })
  expect(inUseHolders(e)).toEqual(['dotnet', 'Code'])
  expect(
    inUseHolders(new Error('x', { cause: { kind: 'in_use', detail: { holders: [] } } })),
  ).toEqual([])
  expect(inUseHolders(new Error('plain'))).toEqual([])
  expect(inUseHolders('plain')).toEqual([])
})
