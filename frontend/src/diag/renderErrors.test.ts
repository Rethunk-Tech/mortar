import { expect, mock, test } from 'bun:test'

const sent: string[] = []
mock.module('../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts', () => ({
  LogRenderError: async (m: string, s: string) => {
    sent.push(`${m}|${s}`)
  },
}))
const { logRenderError } = await import('./renderErrors.ts')

test('one line per distinct stack, and not faster than once a second', () => {
  const loop = new Error('Maximum update depth exceeded')
  expect(logRenderError(loop, { componentStack: '\n at Foo' }, 10_000)).toBe(true)
  expect(logRenderError(loop, { componentStack: '\n at Foo' }, 11_000)).toBe(false)
  expect(logRenderError(loop, { componentStack: '\n at Bar' }, 10_100)).toBe(false)
  expect(logRenderError(loop, { componentStack: '\n at Bar' }, 12_000)).toBe(true)
  expect(sent).toHaveLength(2)
})
