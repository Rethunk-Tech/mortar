import { alpha, createTheme, responsiveFontSizes, type Theme } from '@mui/material/styles'
import { compact } from '../game/compact.ts'
import { type AccentName, accents } from './accents.ts'

const THUMB_ALPHA = 0.45
const THUMB_HOVER_ALPHA = 0.7
const TRACK_ALPHA = 0.08
const FONT = '"Open Sans", sans-serif'
const HTML_FONT_SIZE = 18
const HTML_FONT_SIZE_COMPACT = 16
const TITLE_BAR_PX = 36
const TITLE_BAR_COMPACT_PX = 32
const BUTTON_HEIGHT = TITLE_BAR_PX
const BUTTON_HEIGHT_COMPACT = TITLE_BAR_COMPACT_PX
const FOCUS_OUTLINE_PX = 2
const MENU_RADIUS_PX = 8
const BUTTON_RADIUS_PX = 6
const BUTTON_FONT_PX = 13
const BUTTON_FONT_WEIGHT = 500
const BUTTON_LINE_HEIGHT = 1.4
const BUTTON_PAD_COMPACT = '0 10px'
const BUTTON_PAD = '0 12px'
const BUTTON_SMALL_HEIGHT = 28
const BUTTON_SMALL_PAD = '0 10px'
const BUTTON_LARGE_HEIGHT = 44
const BUTTON_LARGE_PAD = '0 20px'
const BUTTON_LARGE_FONT_PX = 15
const ICON_GAP_PX = 6
const BUTTON_CONTAINED_WEIGHT = 700
const SCROLL_EM = '0.5em'
const THUMB_RADIUS_EM = '0.25em'
const REDUCE_MOTION_MS = '0.01ms'
const UNDERLINE_OFFSET_EM = '0.15em'
const REDUCE_MOTION_ITERATIONS = '1'

function baselineCss(main: string, reduceMotion: boolean) {
  return {
    ':root': {
      '--title-bar': `${TITLE_BAR_PX}px`,
      [compact]: { '--title-bar': `${TITLE_BAR_COMPACT_PX}px` },
    },
    '*': {
      scrollbarWidth: 'thin',
      scrollbarColor: `${alpha(main, THUMB_ALPHA)} ${alpha(main, TRACK_ALPHA)}`,
      ...(reduceMotion
        ? {
            animationDuration: `${REDUCE_MOTION_MS} !important`,
            animationIterationCount: `${REDUCE_MOTION_ITERATIONS} !important`,
            transitionDuration: `${REDUCE_MOTION_MS} !important`,
          }
        : {}),
    },
    '*::-webkit-scrollbar': { width: SCROLL_EM, height: SCROLL_EM },
    '*::-webkit-scrollbar-track': { background: alpha(main, TRACK_ALPHA) },
    '*::-webkit-scrollbar-thumb': {
      background: alpha(main, THUMB_ALPHA),
      borderRadius: THUMB_RADIUS_EM,
    },
    '*::-webkit-scrollbar-thumb:hover': { background: alpha(main, THUMB_HOVER_ALPHA) },
    ':focus-visible': {
      outline: `${FOCUS_OUTLINE_PX}px solid ${main}`,
      outlineOffset: FOCUS_OUTLINE_PX,
    },
  }
}

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
        styleOverrides: baselineCss(main, reduceMotion),
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
            borderRadius: MENU_RADIUS_PX,
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
            borderRadius: BUTTON_RADIUS_PX,
            fontSize: BUTTON_FONT_PX,
            fontWeight: BUTTON_FONT_WEIGHT,
            lineHeight: BUTTON_LINE_HEIGHT,
            height: compactUi ? BUTTON_HEIGHT_COMPACT : BUTTON_HEIGHT,
            padding: compactUi ? BUTTON_PAD_COMPACT : BUTTON_PAD,
          },
          sizeSmall: { height: BUTTON_SMALL_HEIGHT, padding: BUTTON_SMALL_PAD },
          sizeLarge: {
            height: BUTTON_LARGE_HEIGHT,
            padding: BUTTON_LARGE_PAD,
            fontSize: BUTTON_LARGE_FONT_PX,
          },
          startIcon: { marginLeft: 0, marginRight: ICON_GAP_PX },
          contained: { fontWeight: BUTTON_CONTAINED_WEIGHT },
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
            textUnderlineOffset: UNDERLINE_OFFSET_EM,
            '&:hover, &:focus-visible': {
              color: 'rgba(255,255,255,0.90)',
              textDecoration: 'underline solid',
            },
          },
        },
      },
      MuiButtonBase: {
        styleOverrides: {
          root: {
            '&.Mui-focusVisible': {
              outline: `${FOCUS_OUTLINE_PX}px solid ${main}`,
              outlineOffset: FOCUS_OUTLINE_PX,
            },
          },
        },
      },
    },
  })
  return responsiveFontSizes(theme)
}
