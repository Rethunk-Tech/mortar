import { alpha, createTheme, responsiveFontSizes, type Theme } from '@mui/material/styles'
import { compact } from '../game/compact.ts'
import { type AccentName, accents } from './accents.ts'
import { mortarPalette, surfaceCssVars, surfaces, type ThemeMode } from './palette.ts'

const THUMB_ALPHA = 0.45
const THUMB_HOVER_ALPHA = 0.7
const TRACK_ALPHA_DARK = 0.08
const TRACK_ALPHA_LIGHT = 0.16
const FONT = '"Open Sans", sans-serif'
const MONO = 'ui-monospace, "SFMono-Regular", Menlo, Monaco, Consolas, monospace'
// MUI converts px to rem against this root size, so a larger value renders smaller text: compact shrinks type.
const HTML_FONT_SIZE = 18
const HTML_FONT_SIZE_COMPACT = 20
const TITLE_BAR_PX = 36
const TITLE_BAR_COMPACT_PX = 32
const WINDOW_BUTTON_PX = 46
const WINDOW_BUTTON_COMPACT_PX = 40
const WINDOW_BUTTONS = 3
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
const BUTTON_LARGE_HEIGHT = 46
const BUTTON_LARGE_PAD = '0 20px'
const BUTTON_LARGE_FONT_PX = 16
const BUTTON_LARGE_FONT_WEIGHT = 700
const ICON_GAP_PX = 6
const BUTTON_CONTAINED_WEIGHT = 700
const SCROLL_EM = '0.5em'
const THUMB_RADIUS_EM = '0.25em'
const REDUCE_MOTION_MS = '0.01ms'
const UNDERLINE_OFFSET_EM = '0.15em'
const REDUCE_MOTION_ITERATIONS = '1'

function baselineCss(main: string, reduceMotion: boolean, mode: ThemeMode) {
  const trackAlpha = mode === 'light' ? TRACK_ALPHA_LIGHT : TRACK_ALPHA_DARK
  return {
    ':root': {
      '--title-bar': `${TITLE_BAR_PX}px`,
      '--window-button': `${WINDOW_BUTTON_PX}px`,
      '--window-controls': `${WINDOW_BUTTON_PX * WINDOW_BUTTONS}px`,
      ...surfaceCssVars(mode),
      colorScheme: mode,
      [compact]: {
        '--title-bar': `${TITLE_BAR_COMPACT_PX}px`,
        '--window-button': `${WINDOW_BUTTON_COMPACT_PX}px`,
        '--window-controls': `${WINDOW_BUTTON_COMPACT_PX * WINDOW_BUTTONS}px`,
      },
    },
    '*': {
      scrollbarWidth: 'thin',
      scrollbarColor: `${alpha(main, THUMB_ALPHA)} ${alpha(main, trackAlpha)}`,
      ...(reduceMotion
        ? {
            animationDuration: `${REDUCE_MOTION_MS} !important`,
            animationIterationCount: `${REDUCE_MOTION_ITERATIONS} !important`,
            transitionDuration: `${REDUCE_MOTION_MS} !important`,
          }
        : {}),
    },
    '*::-webkit-scrollbar': { width: SCROLL_EM, height: SCROLL_EM },
    '*::-webkit-scrollbar-track': { background: alpha(main, trackAlpha) },
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
  opts: { compact?: boolean; reduceMotion?: boolean; mode?: ThemeMode } = {},
): Theme {
  const main = accents[accent]
  const compactUi = opts.compact === true
  const reduceMotion = opts.reduceMotion === true
  const mode: ThemeMode = opts.mode === 'light' ? 'light' : 'dark'
  const pal = mortarPalette(mode, main)
  const s = surfaces(mode)
  const theme = createTheme({
    palette: {
      mode,
      background: pal.background,
      text: pal.text,
      primary: pal.primary,
      info: pal.info,
      success: pal.success,
      warning: pal.warning,
      error: pal.error,
    },
    typography: {
      fontFamily: FONT,
      htmlFontSize: compactUi ? HTML_FONT_SIZE_COMPACT : HTML_FONT_SIZE,
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: baselineCss(pal.primaryMain, reduceMotion, mode),
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
      MuiDialog: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: { paper: { backgroundColor: s.panelSolid } },
      },
      // MUI drops the content's top padding under a title, which clips the floating label of a leading
      // outlined field.
      MuiDialogContent: {
        styleOverrides: { root: { '.MuiDialogTitle-root + &': { paddingTop: 8 } } },
      },
      MuiBackdrop: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: {
          root: {
            variants: [{ props: { invisible: false }, style: { backgroundColor: s.overlay30 } }],
          },
        },
      },
      MuiMenu: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: {
          paper: {
            backgroundColor: s.menu,
            border: `1px solid ${s.hairline14}`,
            borderRadius: MENU_RADIUS_PX,
          },
        },
      },
      MuiPopover: { defaultProps: { transitionDuration: 0 } },
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
            fontWeight: BUTTON_LARGE_FONT_WEIGHT,
          },
          startIcon: { marginLeft: 0, marginRight: ICON_GAP_PX },
          contained: { fontWeight: BUTTON_CONTAINED_WEIGHT },
        },
        variants: [
          {
            props: { variant: 'outlined', color: 'primary' },
            style: {
              color: s.ink,
              borderColor: s.hairline22,
              '&:hover': {
                borderColor: s.hairline40,
                backgroundColor: s.hairlineFaint,
              },
            },
          },
        ],
      },
      MuiChip: { styleOverrides: { label: { whiteSpace: 'nowrap' } } },
      MuiLink: {
        styleOverrides: {
          root: {
            color: s.inkSec,
            textDecoration: 'underline dotted',
            textUnderlineOffset: UNDERLINE_OFFSET_EM,
            '&:hover, &:focus-visible': {
              color: s.ink90,
              textDecoration: 'underline solid',
            },
          },
        },
      },
      MuiButtonBase: {
        styleOverrides: {
          root: {
            '&.Mui-focusVisible': {
              outline: `${FOCUS_OUTLINE_PX}px solid ${pal.primaryMain}`,
              outlineOffset: FOCUS_OUTLINE_PX,
            },
          },
        },
      },
    },
  })
  return responsiveFontSizes(theme)
}

export { MONO }
