import { expect, test } from 'bun:test'
import { densityVars } from './density.ts'

test('compact is tighter than comfortable on every token', () => {
  const comfortable = densityVars('comfortable')
  const compact = densityVars('compact')
  for (const key of Object.keys(comfortable) as (keyof typeof comfortable)[]) {
    expect(Number.parseInt(compact[key], 10)).toBeLessThan(Number.parseInt(comfortable[key], 10))
  }
})
