import {
  alpha,
  type Components,
  createTheme,
  responsiveFontSizes,
  type Theme,
} from '@mui/material/styles'
import { compact } from '../game/compact.ts'
import { type AccentName, accents } from './accents.ts'
import { calloutFill, calloutLine } from './callout.ts'
import { densityVars, space } from './density.ts'
import {
  contrastText,
  mortarPalette,
  statusColors,
  surfaceCssVars,
  surfaces,
  type ThemeMode,
} from './palette.ts'

const THUMB_ALPHA = 0.45
const THUMB_HOVER_ALPHA = 0.7
const TRACK_ALPHA_DARK = 0.08
const TRACK_ALPHA_LIGHT = 0.16
const FONT = '"Open Sans", sans-serif'
const MONO = 'ui-monospace, "SFMono-Regular", Menlo, Monaco, Consolas, monospace'
// MUI converts px to rem against this root size, so a larger value renders smaller text: compact shrinks type.
const HTML_FONT_SIZE = 18
const HTML_FONT_SIZE_COMPACT = 20
const TITLE_BAR_PX = 48
const TITLE_BAR_COMPACT_PX = 40
const WINDOW_BUTTON_PX = 46
const WINDOW_BUTTON_COMPACT_PX = 40
const WINDOW_BUTTONS = 3
// The smallest touch target on the Steam Deck's screen (WCAG 2.5.5), for every control while touch or a pad leads.
const TARGET_ROOMY_PX = 44
const FOCUS_OUTLINE_PX = 2
// Focus a gamepad moves is script focus, which the browser rings only after keyboard use; shell/gamepad marks it here.
const PAD_FOCUS = 'html[data-input="gamepad"]'
const MENU_RADIUS_PX = 8
const BUTTON_RADIUS_PX = 6
const BUTTON_FONT_PX = 13
const BUTTON_FONT_WEIGHT = 500
const BUTTON_LINE_HEIGHT = 1.4
const BUTTON_PAD_COMPACT = '0 10px'
const BUTTON_PAD = '0 12px'
const BUTTON_SMALL_HEIGHT = 28
const BUTTON_SMALL_PAD = '0 10px'
// The compact Alert is a one-line bar above a list, as tall as the update bar, with small action buttons.
const ALERT_COMPACT_HEIGHT = 38
const ALERT_COMPACT_PAD_END = '6px'
const ALERT_COMPACT_RADIUS = '6px'
const ALERT_COMPACT_FONT_PX = 14
const ALERT_COMPACT_ICON_PX = 18
const ALERT_COMPACT_ICON_GAP_PX = 10
const ALERT_TONES = ['info', 'warning', 'success', 'error'] as const
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

const noDuration = {
  shortest: 0,
  shorter: 0,
  short: 0,
  standard: 0,
  complex: 0,
  enteringScreen: 0,
  leavingScreen: 0,
}

const roomyTargets = {
  'html .MuiButtonBase-root:not(.MuiSwitch-switchBase)': {
    minWidth: TARGET_ROOMY_PX,
    minHeight: TARGET_ROOMY_PX,
  },
  'html .MuiInputBase-root': { minHeight: TARGET_ROOMY_PX },
  // A resize handle stays a thin line; an invisible strip centred on it takes the touch.
  'html [role="separator"][tabindex="0"]::before': {
    content: '""',
    position: 'absolute',
    insetBlock: 0,
    left: '50%',
    width: TARGET_ROOMY_PX,
    transform: 'translateX(-50%)',
  },
}

function baselineCss(
  main: string,
  mode: ThemeMode,
  { reduceMotion, roomy, compactUi }: { reduceMotion: boolean; roomy: boolean; compactUi: boolean },
) {
  const trackAlpha = mode === 'light' ? TRACK_ALPHA_LIGHT : TRACK_ALPHA_DARK
  return {
    ':root': {
      '--title-bar': `${TITLE_BAR_PX}px`,
      '--window-button': `${WINDOW_BUTTON_PX}px`,
      '--window-controls': `${WINDOW_BUTTON_PX * WINDOW_BUTTONS}px`,
      ...surfaceCssVars(mode),
      ...densityVars(compactUi ? 'compact' : 'comfortable'),
      // Text and icons tinted with the accent; pastel accents are unreadable as ink on light surfaces.
      '--mortar-accent-ink': mode === 'light' ? 'var(--mortar-ink)' : main,
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
    [`:focus-visible, ${PAD_FOCUS} :focus`]: {
      outline: `${FOCUS_OUTLINE_PX}px solid ${main}`,
      outlineOffset: FOCUS_OUTLINE_PX,
    },
    ...(roomy ? roomyTargets : {}),
  }
}

// A checked mark is a glyph, so it reads as ink; the switch track is a fill, so it shows the full accent.
// The compact Alert: a one-line bar above a list, as tall as the update bar, with small action buttons.
function compactAlert(): Components<Theme>['MuiAlert'] {
  return {
    variants: [
      {
        props: { variant: 'compact' },
        style: ({ theme: th }) => ({
          minHeight: ALERT_COMPACT_HEIGHT,
          padding: `0 ${ALERT_COMPACT_PAD_END} 0 ${space.pad}`,
          alignItems: 'center',
          borderRadius: ALERT_COMPACT_RADIUS,
          fontSize: ALERT_COMPACT_FONT_PX,
          color: th.palette.text.primary,
          '& .MuiAlert-icon': {
            padding: 0,
            marginRight: ALERT_COMPACT_ICON_GAP_PX,
            fontSize: ALERT_COMPACT_ICON_PX,
            alignSelf: 'center',
          },
          '& .MuiAlert-message': { padding: '6px 0' },
          '& .MuiAlert-action': { padding: 0, marginRight: 0, alignItems: 'center' },
          '& .MuiAlert-action .MuiButton-root': {
            height: BUTTON_SMALL_HEIGHT,
            padding: BUTTON_SMALL_PAD,
          },
        }),
      },
      ...ALERT_TONES.map((tone) => ({
        props: { variant: 'compact' as const, severity: tone },
        style: ({ theme: th }: { theme: Theme }) => ({
          backgroundColor: calloutFill(tone)(th),
          border: `1px solid ${calloutLine(tone)(th)}`,
          '& .MuiAlert-icon': { color: th.palette[tone].main },
        }),
      })),
    ],
  }
}

function lightControls(ink: string): Components<Theme> {
  return {
    MuiIconButton: { styleOverrides: { colorPrimary: { color: ink } } },
    MuiRadio: { styleOverrides: { colorPrimary: { '&.Mui-checked': { color: ink } } } },
    MuiCheckbox: {
      styleOverrides: { colorPrimary: { '&.Mui-checked': { color: ink } } },
    },
    MuiSwitch: {
      styleOverrides: {
        switchBase: { '&.Mui-checked': { color: '#ffffff' } },
        track: { '.Mui-checked.Mui-checked + &': { opacity: 1 } },
        thumb: { boxShadow: '0 1px 2px rgba(0,0,0,0.3)' },
      },
    },
  }
}

// Menus, a select's list, Autocomplete and tooltips are popups on one solid surface.
function popupOverrides(s: ReturnType<typeof surfaces>): Components<Theme> {
  return {
    MuiMenuItem: {
      styleOverrides: { root: { paddingTop: space.menuY, paddingBottom: space.menuY } },
    },
    MuiMenu: { defaultProps: { transitionDuration: 0 } },
    // A heading inside a menu or a select's list reads like the menus' own section headings, on the menu's surface: it
    // stays put as the list scrolls, so it cannot be see-through.
    MuiListSubheader: {
      styleOverrides: {
        root: {
          backgroundColor: s.menu,
          color: 'var(--mortar-ink-sec)',
          fontSize: 11,
          fontWeight: 700,
          letterSpacing: '0.08em',
          textTransform: 'uppercase',
          lineHeight: '28px',
          paddingTop: 6,
        },
      },
    },
    // An Autocomplete's list is its own Paper, not a Menu, so without this it takes the translucent paper colour.
    MuiAutocomplete: {
      styleOverrides: {
        paper: {
          backgroundColor: s.menu,
          border: `1px solid ${s.hairline14}`,
          borderRadius: MENU_RADIUS_PX,
        },
      },
    },
    // MUI's tooltip is a translucent grey over whatever it covers; it takes the menu surface like every other popup.
    MuiTooltip: {
      styleOverrides: {
        tooltip: {
          backgroundColor: s.menu,
          color: s.ink,
          border: `1px solid ${s.hairline14}`,
        },
      },
    },
    // Every popover, a Menu's included, is a solid surface: the paper colour is translucent over the wallpaper.
    MuiPopover: {
      defaultProps: { transitionDuration: 0 },
      styleOverrides: {
        paper: {
          backgroundColor: s.menu,
          border: `1px solid ${s.hairline14}`,
          borderRadius: MENU_RADIUS_PX,
        },
      },
    },
  }
}

function buttonOverrides(
  s: ReturnType<typeof surfaces>,
  mode: ThemeMode,
  compactUi: boolean,
): Components<Theme>['MuiButton'] {
  return {
    defaultProps: { disableElevation: true },
    styleOverrides: {
      root: {
        whiteSpace: 'nowrap',
        textTransform: 'none',
        borderRadius: BUTTON_RADIUS_PX,
        fontSize: BUTTON_FONT_PX,
        fontWeight: BUTTON_FONT_WEIGHT,
        lineHeight: BUTTON_LINE_HEIGHT,
        height: space.control,
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
      // Light mode darkens status colours for text; a filled button keeps the colour as picked, with black or white text.
      ...(mode === 'light'
        ? (Object.keys(statusColors) as (keyof typeof statusColors)[]).map((color) => ({
            props: { variant: 'contained' as const, color },
            style: {
              backgroundColor: statusColors[color],
              color: contrastText(statusColors[color]),
              '&:hover': { backgroundColor: statusColors[color], filter: 'brightness(0.94)' },
            },
          }))
        : []),
      // Pastel accents are fills, not ink: on a light surface a text button in the accent colour is unreadable.
      ...(mode === 'light'
        ? [
            {
              props: { variant: 'text' as const, color: 'primary' as const },
              style: { color: s.ink },
            },
          ]
        : []),
    ],
  }
}

/** The bundled font behind an installed family the player chose, so a glyph the choice lacks still draws. */
function fontStack(family: string | undefined): string {
  const name = (family ?? '').replaceAll('"', '').trim()
  return name === '' ? FONT : `"${name}", ${FONT}`
}

function createMortarTheme(
  accent: AccentName,
  opts: {
    font?: string
    accentHex?: string
    compact?: boolean
    roomy?: boolean
    reduceMotion?: boolean
    mode?: ThemeMode
  } = {},
): Theme {
  const main = opts.accentHex ?? accents[accent]
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
    // The global rule below stops CSS transitions; MUI's own timers (Collapse, Fade, ripples) read these durations.
    ...(reduceMotion ? { transitions: { create: () => 'none', duration: noDuration } } : {}),
    typography: {
      fontFamily: fontStack(opts.font),
      htmlFontSize: compactUi ? HTML_FONT_SIZE_COMPACT : HTML_FONT_SIZE,
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: baselineCss(pal.primaryMain, mode, {
          reduceMotion,
          roomy: opts.roomy === true,
          compactUi,
        }),
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
      MuiAlert: compactAlert(),
      MuiDialog: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: { paper: { backgroundColor: s.panelSolid } },
      },
      // MUI drops the content's top padding under a title, which clips the floating label of a leading
      // outlined field.
      // The content sits as far below the title as the actions sit below the content; the title's line spacing already
      // adds room above its text, so it starts a little higher.
      MuiDialogContent: {
        styleOverrides: {
          root: {
            padding: space.pad,
            '.MuiDialogTitle-root + &': { paddingTop: 8 },
            '&:not(.MuiDialogContent-dividers):has(+ .MuiDialogActions-root)': { paddingBottom: 0 },
          },
        },
      },
      MuiDialogTitle: {
        styleOverrides: { root: { padding: `calc(${space.pad} - 4px) ${space.pad} 0` } },
      },
      MuiDialogActions: {
        styleOverrides: { root: { padding: `12px ${space.pad} ${space.pad}`, gap: space.gap } },
      },
      MuiBackdrop: {
        defaultProps: { transitionDuration: 0 },
        styleOverrides: {
          root: {
            variants: [{ props: { invisible: false }, style: { backgroundColor: s.overlay30 } }],
          },
        },
      },
      // A dropdown or field beside buttons takes the buttons' height.
      MuiInputBase: {
        styleOverrides: {
          root: {
            '&.MuiInputBase-sizeSmall:not(.MuiInputBase-multiline)': { height: space.control },
          },
        },
      },
      ...popupOverrides(s),
      MuiCollapse: { defaultProps: { timeout: 0 } },
      // A sliding exit leaves the modal's backdrop mounted for its duration, so the click right after a drawer closes
      // lands on the backdrop and does nothing.
      MuiDrawer: { defaultProps: { transitionDuration: 0 } },
      MuiTab: { styleOverrides: { root: { textTransform: 'none', whiteSpace: 'nowrap' } } },
      MuiToggleButton: {
        styleOverrides: { root: { textTransform: 'none', whiteSpace: 'nowrap' } },
      },
      MuiButton: buttonOverrides(s, mode, compactUi),
      ...(mode === 'light' ? lightControls(s.ink) : {}),
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

export {
  BUTTON_LARGE_HEIGHT,
  BUTTON_RADIUS_PX,
  BUTTON_SMALL_HEIGHT,
  createMortarTheme,
  MONO,
  PAD_FOCUS,
}

declare module '@mui/material/Alert' {
  interface AlertPropsVariantOverrides {
    compact: true
  }
}
