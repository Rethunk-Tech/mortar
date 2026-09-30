import { alpha, createTheme, responsiveFontSizes, type Theme } from '@mui/material/styles'
import { compact } from '../game/compact.ts'
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
          ':root': { '--title-bar': '36px', [compact]: { '--title-bar': '32px' } },
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
      MuiButton: {
        defaultProps: { disableElevation: true },
        styleOverrides: {
          root: {
            whiteSpace: 'nowrap',
            textTransform: 'none',
            borderRadius: 6,
            fontSize: 13,
            fontWeight: 500,
            lineHeight: 1.4,
            height: 36,
            padding: '0 12px',
          },
          sizeSmall: { height: 28, padding: '0 10px' },
          sizeLarge: { height: 44, padding: '0 20px', fontSize: 15 },
          startIcon: { marginLeft: 0, marginRight: 6 },
          contained: { fontWeight: 700 },
        },
        variants: [
          {
            props: { variant: 'outlined', color: 'primary' },
            style: {
              color: '#ffffff',
              borderColor: 'rgba(255,255,255,0.22)',
              '&:hover': {
                borderColor: 'rgba(255,255,255,0.4)',
                backgroundColor: 'rgba(255,255,255,0.06)',
              },
            },
          },
        ],
      },
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
