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

  test.each(['MuiDialog', 'MuiMenu', 'MuiPopover', 'MuiDrawer', 'MuiBackdrop'] as const)(
    '%s closes without an exit transition that leaves its backdrop catching clicks',
    (name) => {
      const props = createMortarTheme('sand').components?.[name]?.defaultProps
      expect(props).toEqual({ transitionDuration: 0 })
    },
  )

  test('dialog paper is solid', () => {
    const paper = createMortarTheme('sand').components?.MuiDialog?.styleOverrides?.paper
    expect(paper).toEqual({ backgroundColor: 'rgb(40,40,48)' })
  })

  test.each(['dark', 'light'] as const)('%s tooltips use the opaque menu surface', (mode) => {
    const tip = createMortarTheme('sand', { mode }).components?.MuiTooltip?.styleOverrides?.tooltip
    expect(tip).toMatchObject({
      backgroundColor: mode === 'dark' ? 'rgba(28,28,34,0.99)' : '#ffffff',
    })
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

  test('reduced motion zeroes MUI transitions and the global CSS durations', () => {
    const theme = createMortarTheme('sand', { reduceMotion: true })
    expect(theme.transitions.duration.standard).toBe(0)
    expect(theme.transitions.create('opacity')).toBe('none')
    const css = theme.components?.MuiCssBaseline?.styleOverrides as Record<string, unknown>
    expect(css['*']).toMatchObject({ animationDuration: '0.01ms !important' })
    expect(createMortarTheme('sand').transitions.duration.standard).toBeGreaterThan(0)
  })
})
