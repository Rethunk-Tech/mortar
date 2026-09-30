import { createTheme, type Theme } from '@mui/material/styles'
import { type AccentName, accents } from '../theme/accents.ts'
import { createMortarTheme } from '../theme/theme.ts'

const OPAQUE_DEFAULT = 'rgb(25,25,30)'
const OPAQUE_PAPER = 'rgb(50,50,60)'

export function isAccent(value: string): value is AccentName {
  return value in accents
}

export function buildTheme(accent: AccentName, solid: boolean): Theme {
  const base = createMortarTheme(accent)
  if (!solid) {
    return base
  }
  return createTheme(base, {
    palette: { background: { default: OPAQUE_DEFAULT, paper: OPAQUE_PAPER } },
    components: {
      MuiCssBaseline: {
        styleOverrides: { 'html, body, #root': { background: OPAQUE_DEFAULT } },
      },
    },
  })
}
