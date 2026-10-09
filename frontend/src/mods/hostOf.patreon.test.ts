import { expect, test } from 'bun:test'
import { hostOf } from './modActions.ts'

test('a Patreon post page is named Patreon', () => {
  expect(hostOf('https://www.patreon.com/posts/cool-mod-4242')).toBe('patreon')
  expect(hostOf('https://patreon.com.evil.test/posts/1')).toBe('web')
})
