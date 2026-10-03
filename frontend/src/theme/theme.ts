import { alpha, createTheme, responsiveFontSizes, type Theme } from '@mui/material/styles'
import { compact } from '../game/compact.ts'
import { type AccentName, accents } from './accents.ts'

const THUMB_ALPHA = 0.45
const THUMB_HOVER_ALPHA = 0.7
const TRACK_ALPHA = 0.08
const FONT = '"Open Sans", sans-serif'
const HTML_FONT_SIZE = 18
const HTML_FONT_SIZE_COMPACT = 16
const BUTTON_HEIGHT = 36
const BUTTON_HEIGHT_COMPACT = 32

export function createMortarTheme(
  accent: AccentName,
  opts: { compact?: boolean; reduceMotion?: boolean } = {},
): Theme {
  const main = accents[accent]
  const compactUi = opts.compact === true
  const reduceMotion = opts.reduceMotion === true
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
    typography: {
      fontFamily: FONT,
      htmlFontSize: compactUi ? HTML_FONT_SIZE_COMPACT : HTML_FONT_SIZE,
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          ':root': { '--title-bar': '36px', [compact]: { '--title-bar': '32px' } },
          '*': {
            scrollbarWidth: 'thin',
            scrollbarColor: `${alpha(main, THUMB_ALPHA)} ${alpha(main, TRACK_ALPHA)}`,
            ...(reduceMotion
              ? {
                  animationDuration: '0.01ms !important',
                  animationIterationCount: '1 !important',
                  transitionDuration: '0.01ms !important',
                }
              : {}),
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
      MuiDialog: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: { paper: { backgroundColor: 'rgb(40,40,48)' } },
      },
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
      MuiMenu: {
        styleOverrides: {
          paper: {
            backgroundColor: 'rgba(28,28,34,0.99)',
            border: '1px solid rgba(255,255,255,0.14)',
            borderRadius: 8,
          },
        },
      },
      MuiTab: { styleOverrides: { root: { textTransform: 'none', whiteSpace: 'nowrap' } } },
      MuiToggleButton: {
        styleOverrides: { root: { textTransform: 'none', whiteSpace: 'nowrap' } },
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
            height: compactUi ? BUTTON_HEIGHT_COMPACT : BUTTON_HEIGHT,
            padding: compactUi ? '0 10px' : '0 12px',
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
      MuiLink: {
        styleOverrides: {
          root: {
            color: 'rgba(225,225,230,0.95)',
            textDecoration: 'underline dotted',
            textUnderlineOffset: '0.15em',
            '&:hover, &:focus-visible': {
              color: 'rgba(255,255,255,0.90)',
              textDecoration: 'underline solid',
            },
          },
        },
      },
      MuiButtonBase: {
        styleOverrides: {
          root: { '&.Mui-focusVisible': { outline: `2px solid ${main}`, outlineOffset: 2 } },
        },
      },
    },
  })
  return responsiveFontSizes(theme)
}
