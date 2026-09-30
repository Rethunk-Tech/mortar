import { alpha, createTheme, responsiveFontSizes, type Theme } from '@mui/material/styles'
import { type AccentName, accents } from './accents.ts'

const THUMB_ALPHA = 0.45
const THUMB_HOVER_ALPHA = 0.7
const TRACK_ALPHA = 0.08
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
            scrollbarColor: `${alpha(main, THUMB_ALPHA)} ${alpha(main, TRACK_ALPHA)}`,
          },
          '*::-webkit-scrollbar': { width: '0.5em', height: '0.5em' },
          '*::-webkit-scrollbar-track': { background: alpha(main, TRACK_ALPHA) },
          '*::-webkit-scrollbar-thumb': {
            background: alpha(main, THUMB_ALPHA),
            borderRadius: '0.25em',
          },
          '*::-webkit-scrollbar-thumb:hover': { background: alpha(main, THUMB_HOVER_ALPHA) },
          ':focus-visible': { outline: `2px solid ${main}`, outlineOffset: 2 },
        },
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
      // On WebKitGTK a fading or filtered full-window layer turns the translucent
      // window opaque; a static tint keeps it translucent.
      MuiBackdrop: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: {
          root: {
            variants: [
              { props: { invisible: false }, style: { backgroundColor: 'rgba(0,0,0,0.3)' } },
            ],
          },
        },
      },
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
