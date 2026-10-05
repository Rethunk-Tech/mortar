import { expect, test } from 'bun:test'
import { formatModes, OFF, parseModes } from './browseModes.ts'

test('modes round-trip and ignore junk', () => {
  const m = parseModes('installed=gray obsolete=hide broken=bogus x=hide')
  expect(m).toEqual({ installed: 'gray', obsolete: 'hide', broken: 'off' })
  expect(formatModes(m)).toBe('installed=gray obsolete=hide')
  expect(parseModes('')).toEqual(OFF)
})
