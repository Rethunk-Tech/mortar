import { alpha, createTheme, responsiveFontSizes, type Theme } from '@mui/material/styles'
import { type AccentName, accents } from './accents.ts'

const FONT = '"Open Sans", sans-serif'

export function createMortarTheme(accent: AccentName): Theme {
  const main = accents[accent]
  const theme = createTheme({
    palette: {
      mode: 'dark',
      background: { default: 'rgba(25,25,30,0.80)', paper: 'rgba(50,50,60,0.80)' },
      text: { primary: 'rgba(255,255,255,0.90)', secondary: 'rgba(225,225,230,0.95)' },
      primary: { main, contrastText: '#1b1a17' },
      info: { main: '#2B8BDA' },
      success: { main: '#0CDF64' },
      warning: { main: '#F3B416' },
      error: { main: '#C70A0A' },
    },
    typography: { fontFamily: FONT, htmlFontSize: 18 },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          'html, body, #root': { background: 'transparent' },
          '*': {
            scrollbarWidth: 'thin',
            scrollbarColor: `${alpha(main, 0.45)} ${alpha(main, 0.08)}`,
          },
          '*::-webkit-scrollbar': { width: '0.5em', height: '0.5em' },
          '*::-webkit-scrollbar-track': { background: alpha(main, 0.08) },
          '*::-webkit-scrollbar-thumb': {
            background: alpha(main, 0.45),
            borderRadius: '0.25em',
          },
          '*::-webkit-scrollbar-thumb:hover': { background: alpha(main, 0.7) },
          ':focus-visible': { outline: `2px solid ${main}`, outlineOffset: 2 },
        },
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
      MuiBackdrop: { styleOverrides: { root: { backgroundColor: 'rgba(0,0,0,0.75)' } } },
      MuiButton: { styleOverrides: { root: { whiteSpace: 'nowrap' } } },
      MuiChip: { styleOverrides: { label: { whiteSpace: 'nowrap' } } },
      MuiButtonBase: {
        styleOverrides: {
          root: { '&.Mui-focusVisible': { outline: `2px solid ${main}`, outlineOffset: 2 } },
        },
      },
    },
  })
  return responsiveFontSizes(theme)
}
