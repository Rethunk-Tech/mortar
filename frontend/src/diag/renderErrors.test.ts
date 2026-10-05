import { expect, test } from 'bun:test'
import { logRenderError } from './renderErrors.ts'

const sent: string[] = []
const send = async (m: string, s: string) => {
  sent.push(`${m}|${s}`)
}

test('one line per distinct stack, and not faster than once a second', () => {
  const loop = new Error('Maximum update depth exceeded')
  expect(logRenderError(loop, { componentStack: '\n at Foo' }, 10_000, send)).toBe(true)
  expect(logRenderError(loop, { componentStack: '\n at Foo' }, 11_000, send)).toBe(false)
  expect(logRenderError(loop, { componentStack: '\n at Bar' }, 10_100, send)).toBe(false)
  expect(logRenderError(loop, { componentStack: '\n at Bar' }, 12_000, send)).toBe(true)
  expect(sent).toHaveLength(2)
})
