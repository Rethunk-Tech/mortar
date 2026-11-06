import { afterEach, beforeEach, expect, mock, spyOn, test } from 'bun:test'
import * as opener from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { openWeb } from './openWeb.ts'

let opened: unknown[][] = []

// Only window.open is stubbed: the DOM the preload installs stays for the tests that run after these.
beforeEach(() => {
  opened = []
  spyOn(globalThis.window, 'open').mockImplementation((...a: unknown[]) => {
    opened.push(a)
    return null
  })
})
afterEach(() => {
  mock.restore()
})

test('a server build opens the address in this tab, without an opener', () => {
  stubBinding(opener, 'OpenWeb', () => {
    throw new Error('mortar:open-in-page')
  })
  return openWeb('https://example.com/a').then(() => {
    expect(opened).toEqual([['https://example.com/a', '_blank', 'noopener,noreferrer']])
  })
})

test('any other failure is rethrown and opens nothing', async () => {
  stubBinding(opener, 'OpenWeb', () => {
    throw new Error('[invalid] not an http address')
  })
  await expect(openWeb('ftp://x')).rejects.toThrow('not an http address')
  expect(opened).toEqual([])
})

test('a handled open does not touch the page', async () => {
  const bound = stubBinding(opener, 'OpenWeb', () => undefined)
  await openWeb('https://example.com')
  expect(bound).toHaveBeenCalledTimes(1)
  expect(opened).toEqual([])
})
