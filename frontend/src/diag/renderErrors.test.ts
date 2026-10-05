import { afterAll, expect, spyOn, test } from 'bun:test'
import { logRenderError } from './renderErrors.ts'

const reported = spyOn(globalThis, 'reportError').mockImplementation(() => undefined)
afterAll(() => reported.mockRestore())
const sent: string[] = []
const send = async (m: string, s: string) => {
  sent.push(`${m}|${s}`)
}

test('one line per distinct stack, each logged however close together, and reportError still sees every one', () => {
  const loop = new Error('Maximum update depth exceeded')
  expect(logRenderError(loop, { componentStack: '\n at Foo' }, send)).toBe(true)
  expect(logRenderError(loop, { componentStack: '\n at Foo' }, send)).toBe(false)
  expect(logRenderError(loop, { componentStack: '\n at Bar' }, send)).toBe(true)
  expect(sent).toHaveLength(2)
  expect(reported).toHaveBeenCalledTimes(3)
})

test('twenty stacks a session at most', () => {
  for (let i = 0; i < 30; i++) {
    logRenderError(new Error('x'), { componentStack: `\n at C${i}` }, send)
  }
  expect(sent.length).toBe(20)
})
