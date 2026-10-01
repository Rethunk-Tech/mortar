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

  test('dialog paper is solid', () => {
    const paper = createMortarTheme('sand').components?.MuiDialog?.styleOverrides?.paper
    expect(paper).toEqual({ backgroundColor: 'rgb(40,40,48)' })
  })

  test('links are secondary text with a dotted underline', () => {
    const root = createMortarTheme('sand').components?.MuiLink?.styleOverrides?.root
    expect(root).toEqual({
      color: 'rgba(225,225,230,0.95)',
      textDecoration: 'underline dotted',
      textUnderlineOffset: '0.15em',
      '&:hover, &:focus-visible': {
        color: 'rgba(255,255,255,0.90)',
        textDecoration: 'underline solid',
      },
    })
  })
})
