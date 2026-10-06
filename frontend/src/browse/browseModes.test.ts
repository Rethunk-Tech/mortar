import { expect, test } from 'bun:test'
import { DEFAULT_MODES, formatModes, parseModes } from './browseModes.ts'

test('modes round-trip and ignore junk', () => {
  const m = parseModes('installed=gray obsolete=hide broken=bogus x=hide')
  expect(m).toEqual({ installed: 'gray', obsolete: 'hide', broken: 'hide' })
  expect(formatModes(m)).toBe('installed=gray obsolete=hide broken=hide')
  expect(parseModes('')).toEqual(DEFAULT_MODES)
  expect(DEFAULT_MODES).toEqual({ installed: 'hide', obsolete: 'hide', broken: 'hide' })
  // An explicit Off survives a round trip, so choosing it sticks.
  const off = parseModes(formatModes({ installed: 'off', obsolete: 'off', broken: 'off' }))
  expect(off).toEqual({ installed: 'off', obsolete: 'off', broken: 'off' })
})
