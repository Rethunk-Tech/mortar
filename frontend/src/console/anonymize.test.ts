import { expect, test } from 'bun:test'
import { anonymize } from './anonymize.ts'

test('a path and bare mentions are redacted', () => {
  expect(anonymize('at /home/ann/mods by ann.')).toBe('at ~/mods by <user>.')
})

test('regex metacharacters in the user name do not break or widen the match', () => {
  const out = anonymize('/home/a.b+c/.config opened by a.b+c; axb+c stays')
  expect(out).toBe('~/.config opened by <user>; axb+c stays')
})

test('a name that would be an invalid pattern is still redacted', () => {
  expect(anonymize('/home/(x/log (x')).toBe('~/log <user>')
})

test('a log without a home path is unchanged', () => {
  expect(anonymize('no path here')).toBe('no path here')
})
