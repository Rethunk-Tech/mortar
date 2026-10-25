import { expect, test } from 'bun:test'
import { sdkSatisfies } from './sdk.ts'

test('an installed SDK meets a global.json pin by its roll-forward rule', () => {
  const installed = ['8.0.131', '9.0.121', '10.0.112']
  expect(sdkSatisfies({ version: '10.0.100', rollForward: 'latestMajor' }, installed)).toBe(true)
  expect(sdkSatisfies({ version: '10.0.200', rollForward: 'latestFeature' }, installed)).toBe(false)
  expect(sdkSatisfies({ version: '9.0.100', rollForward: 'latestFeature' }, installed)).toBe(true)
  expect(sdkSatisfies({ version: '10.0.100', rollForward: '' }, installed)).toBe(true)
  expect(sdkSatisfies({ version: '10.0.113', rollForward: 'patch' }, installed)).toBe(false)
  expect(sdkSatisfies({ version: '11.0.100', rollForward: 'latestMajor' }, installed)).toBe(false)
  expect(sdkSatisfies({ version: '', rollForward: '' }, [])).toBe(false)
})
