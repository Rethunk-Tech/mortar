import { describe, expect, test } from 'bun:test'
import { type AccentName, accents } from './accents.ts'
import { createMortarTheme } from './theme.ts'

describe('createMortarTheme', () => {
  test.each(Object.keys(accents) as AccentName[])('%s becomes primary', (name) => {
    expect(createMortarTheme(name).palette.primary.main).toBe(accents[name])
  })

  test('Paper has no elevation overlay image', () => {
    const root = createMortarTheme('sand').components?.MuiPaper?.styleOverrides?.root
    expect(root).toEqual({ backgroundImage: 'none' })
  })
})
