import type { PaletteOptions } from '@mui/material/styles'

type ThemeMode = 'light' | 'dark'

const DARK_BASE = 'rgb(25,25,30)'
const MIN_CONTRAST = 4.5

interface Surfaces {
  hairline: string
  hairlineMuted: string
  hairlineFaint: string
  hairlineGhost: string
  hairline12: string
  hairline14: string
  hairline15: string
  hairline16: string
  hairline18: string
  hairline20: string
  hairline22: string
  hairline25: string
  hairline35: string
  hairline40: string
  ink: string
  ink90: string
  ink88: string
  ink85: string
  ink72: string
  inkSec: string
  inkSoft: string
  inkDim: string
  inkDim92: string
  inkDim60: string
  overlay20: string
  // Dims the whole window behind a drop target or the tour spotlight; the overlay ramp is for recessed fills.
  scrim: string
  // The window's title bar over the wallpaper.
  titleBar: string
  overlay24: string
  overlay25: string
  overlay28: string
  overlay30: string
  overlay35: string
  overlay40: string
  overlay45: string
  overlay50: string
  overlay55: string
  overlay80: string
  overlay85: string
  overlay90: string
  panel: string
  panel85: string
  panel92: string
  panelSolid: string
  raised: string
  raised60: string
  menu: string
  menu95: string
  toast: string
  nav: string
  paper78: string
  console: string
  console90: string
  hero: string
  tint: string
  cardHover: string
  gameDim: string
}

const darkSurfaces: Surfaces = {
  hairline: 'rgba(255,255,255,0.1)',
  hairlineMuted: 'rgba(255,255,255,0.08)',
  hairlineFaint: 'rgba(255,255,255,0.06)',
  hairlineGhost: 'rgba(255,255,255,0.03)',
  hairline12: 'rgba(255,255,255,0.12)',
  hairline14: 'rgba(255,255,255,0.14)',
  hairline15: 'rgba(255,255,255,0.15)',
  hairline16: 'rgba(255,255,255,0.16)',
  hairline18: 'rgba(255,255,255,0.18)',
  hairline20: 'rgba(255,255,255,0.2)',
  hairline22: 'rgba(255,255,255,0.22)',
  hairline25: 'rgba(255,255,255,0.25)',
  hairline35: 'rgba(255,255,255,0.35)',
  hairline40: 'rgba(255,255,255,0.4)',
  ink: '#ffffff',
  ink90: 'rgba(255,255,255,0.90)',
  ink88: 'rgba(255,255,255,0.88)',
  ink85: 'rgba(255,255,255,0.85)',
  ink72: 'rgba(255,255,255,0.72)',
  inkSec: 'rgba(225,225,230,0.95)',
  inkSoft: 'rgba(235,235,240,0.95)',
  inkDim: 'rgba(210,210,215,0.85)',
  inkDim92: 'rgba(210,210,215,0.92)',
  inkDim60: 'rgba(210,210,215,0.6)',
  scrim: 'rgba(0,0,0,0.3)',
  titleBar: 'rgba(15,15,18,0.55)',
  overlay20: 'rgba(0,0,0,0.2)',
  overlay24: 'rgba(0,0,0,0.24)',
  overlay25: 'rgba(0,0,0,0.25)',
  overlay28: 'rgba(0,0,0,0.28)',
  overlay30: 'rgba(0,0,0,0.3)',
  overlay35: 'rgba(0,0,0,0.35)',
  overlay40: 'rgba(0,0,0,0.4)',
  overlay45: 'rgba(0,0,0,0.45)',
  overlay50: 'rgba(0,0,0,0.5)',
  overlay55: 'rgba(0,0,0,0.55)',
  overlay80: 'rgba(0,0,0,0.80)',
  overlay85: 'rgba(0,0,0,0.85)',
  overlay90: 'rgba(0,0,0,0.9)',
  panel: 'rgba(40,40,48,0.78)',
  panel85: 'rgba(40,40,48,0.85)',
  panel92: 'rgba(40,40,48,0.92)',
  panelSolid: 'rgb(40,40,48)',
  raised: 'rgba(55,55,65,0.9)',
  raised60: 'rgba(55,55,65,0.6)',
  menu: 'rgba(28,28,34,0.99)',
  menu95: 'rgba(28,28,34,0.95)',
  toast: 'rgba(30,30,36,0.98)',
  nav: 'rgba(30,30,36,0.8)',
  paper78: 'rgba(50,50,60,0.78)',
  console: 'rgba(25,25,30,0.96)',
  console90: 'rgba(25,25,30,0.9)',
  hero: 'rgba(15,15,18,0.5)',
  tint: 'rgba(25,25,30,0.8)',
  cardHover: 'rgba(60,60,70,0.9)',
  gameDim: 'rgba(28,28,32,0.92)',
}

const lightSurfaces: Surfaces = {
  hairline: 'rgba(0,0,0,0.12)',
  hairlineMuted: 'rgba(0,0,0,0.08)',
  hairlineFaint: 'rgba(0,0,0,0.06)',
  hairlineGhost: 'rgba(0,0,0,0.03)',
  hairline12: 'rgba(0,0,0,0.12)',
  hairline14: 'rgba(0,0,0,0.14)',
  hairline15: 'rgba(0,0,0,0.16)',
  hairline16: 'rgba(0,0,0,0.16)',
  hairline18: 'rgba(0,0,0,0.18)',
  hairline20: 'rgba(0,0,0,0.2)',
  hairline22: 'rgba(0,0,0,0.22)',
  hairline25: 'rgba(0,0,0,0.25)',
  hairline35: 'rgba(0,0,0,0.35)',
  hairline40: 'rgba(0,0,0,0.4)',
  ink: '#1a1a1e',
  ink90: 'rgba(26,26,30,0.92)',
  ink88: 'rgba(26,26,30,0.88)',
  ink85: 'rgba(26,26,30,0.85)',
  ink72: 'rgba(26,26,30,0.72)',
  inkSec: 'rgba(40,40,48,0.78)',
  inkSoft: 'rgba(32,32,38,0.88)',
  inkDim: 'rgba(50,50,58,0.7)',
  inkDim92: 'rgba(50,50,58,0.78)',
  inkDim60: 'rgba(50,50,58,0.55)',
  scrim: 'rgba(0,0,0,0.28)',
  titleBar: 'rgba(244,244,247,0.88)',
  // Recessed fills (inputs, toggles, chips, panels) over a light surface want a faint tint, not the dark ramp's veil.
  overlay20: 'rgba(0,0,0,0.04)',
  overlay24: 'rgba(0,0,0,0.045)',
  overlay25: 'rgba(0,0,0,0.05)',
  overlay28: 'rgba(0,0,0,0.055)',
  overlay30: 'rgba(0,0,0,0.06)',
  overlay35: 'rgba(0,0,0,0.07)',
  overlay40: 'rgba(0,0,0,0.08)',
  overlay45: 'rgba(0,0,0,0.09)',
  overlay50: 'rgba(0,0,0,0.1)',
  overlay55: 'rgba(0,0,0,0.12)',
  overlay80: 'rgba(0,0,0,0.55)',
  overlay85: 'rgba(0,0,0,0.5)',
  overlay90: 'rgba(0,0,0,0.45)',
  panel: '#ffffff',
  panel85: '#ffffff',
  panel92: '#f7f7f9',
  panelSolid: '#ffffff',
  raised: '#ececf1',
  raised60: '#f0f0f3',
  menu: '#ffffff',
  menu95: '#ffffff',
  toast: '#ffffff',
  nav: '#f3f3f6',
  paper78: '#ffffff',
  console: '#f4f4f7',
  console90: '#f4f4f7',
  hero: 'rgba(255,255,255,0.72)',
  tint: 'rgba(244,245,247,0.88)',
  cardHover: '#e8e8ee',
  gameDim: 'rgba(255,255,255,0.92)',
}

function surfaces(mode: ThemeMode): Surfaces {
  return mode === 'light' ? lightSurfaces : darkSurfaces
}

const CAMEL_TO_DASH = /[A-Z]/g
const LETTER_THEN_DIGIT = /([a-z])(\d)/g
const HEX_COLOUR = /^#([\da-f]{3}|[\da-f]{6})$/i
const RGB_COLOUR = /^rgba?\(\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)(?:\s*,\s*([\d.]+))?\s*\)$/
const SHORT_HEX_LEN = 3
const HEX_PAIR = 2
const HEX_RADIX = 16
const BYTE_MAX = 255
const SRGB_CUTOFF = 0.040_45
const LINEAR_DIVISOR = 12.92
const GAMMA_OFFSET = 0.055
const GAMMA_SCALE = 1.055
const GAMMA = 2.4
const LUMA_R = 0.2126
const LUMA_G = 0.7152
const LUMA_B = 0.0722
const CONTRAST_PAD = 0.05
const MIX_STEP = 0.08
const MIX_TRIES = 48

function surfaceCssVars(mode: ThemeMode): Record<string, string> {
  const s = surfaces(mode)
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(s)) {
    const kebab = key
      .replace(CAMEL_TO_DASH, (ch) => `-${ch.toLowerCase()}`)
      .replace(LETTER_THEN_DIGIT, '$1-$2')
    out[`--mortar-${kebab}`] = value
  }
  return out
}

interface Rgb {
  r: number
  g: number
  b: number
  a: number
}

function parseColor(input: string): Rgb {
  const hex = HEX_COLOUR.exec(input)
  if (hex) {
    const [, captured] = hex
    if (captured === undefined) {
      throw new Error(`unparseable colour ${input}`)
    }
    let h = captured
    if (h.length === SHORT_HEX_LEN) {
      h = [...h].map((c) => c + c).join('')
    }
    const [r = 0, g = 0, b = 0] = (h.match(/../g) ?? []).map((pair) =>
      Number.parseInt(pair, HEX_RADIX),
    )
    return { r, g, b, a: 1 }
  }
  const rgb = RGB_COLOUR.exec(input)
  if (!rgb) {
    throw new Error(`unparseable colour ${input}`)
  }
  return {
    r: Number(rgb[1]),
    g: Number(rgb[2]),
    b: Number(rgb[3]),
    a: rgb[4] === undefined ? 1 : Number(rgb[4]),
  }
}

function channel(value: number): number {
  const s = value / BYTE_MAX
  return s <= SRGB_CUTOFF ? s / LINEAR_DIVISOR : ((s + GAMMA_OFFSET) / GAMMA_SCALE) ** GAMMA
}

function relativeLuminance(color: string): number {
  const { r, g, b } = parseColor(color)
  return LUMA_R * channel(r) + LUMA_G * channel(g) + LUMA_B * channel(b)
}

function contrastRatio(a: string, b: string): number {
  const l1 = relativeLuminance(a)
  const l2 = relativeLuminance(b)
  const [hi, lo] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (hi + CONTRAST_PAD) / (lo + CONTRAST_PAD)
}

function composite(fg: string, bg: string): string {
  const f = parseColor(fg)
  const b = parseColor(bg)
  const { a } = f
  const r = Math.round(f.r * a + b.r * (1 - a))
  const g = Math.round(f.g * a + b.g * (1 - a))
  const bch = Math.round(f.b * a + b.b * (1 - a))
  return `rgb(${r},${g},${bch})`
}

function toHex({ r, g, b }: Rgb): string {
  const h = (n: number) =>
    Math.max(0, Math.min(BYTE_MAX, Math.round(n)))
      .toString(HEX_RADIX)
      .padStart(HEX_PAIR, '0')
  return `#${h(r)}${h(g)}${h(b)}`
}

function mixToward(color: string, target: Rgb, amount: number): string {
  const c = parseColor(color)
  return toHex({
    r: c.r + (target.r - c.r) * amount,
    g: c.g + (target.g - c.g) * amount,
    b: c.b + (target.b - c.b) * amount,
    a: 1,
  })
}

function ensureContrast(fg: string, bg: string, min = MIN_CONTRAST): string {
  let cur = toHex(parseColor(fg))
  const solidBg = composite(bg, bg)
  if (contrastRatio(cur, solidBg) >= min) {
    return cur
  }
  const towardBlack = relativeLuminance(cur) < relativeLuminance(solidBg)
  const target = towardBlack
    ? { r: 0, g: 0, b: 0, a: 1 }
    : { r: BYTE_MAX, g: BYTE_MAX, b: BYTE_MAX, a: 1 }
  for (let i = 0; i < MIX_TRIES; i++) {
    cur = mixToward(cur, target, MIX_STEP)
    if (contrastRatio(cur, solidBg) >= min) {
      return cur
    }
  }
  return cur
}

function contrastText(bg: string): string {
  return contrastRatio('#ffffff', bg) >= MIN_CONTRAST ? '#ffffff' : '#1b1a17'
}

function paperForContrast(paper: string, mode: ThemeMode): string {
  return composite(paper, mode === 'dark' ? DARK_BASE : paper)
}

function mortarPalette(
  mode: ThemeMode,
  accentHex: string,
): PaletteOptions & {
  primaryMain: string
} {
  const dark = mode === 'dark'
  const paper = dark ? 'rgba(50,50,60,0.80)' : '#f4f4f7'
  const backgroundDefault = dark ? 'rgba(25,25,30,0.80)' : '#ececf1'
  const textPrimary = dark ? 'rgba(255,255,255,0.90)' : 'rgba(22,22,26,0.92)'
  const textSecondary = dark ? 'rgba(225,225,230,0.95)' : 'rgba(40,40,48,0.78)'
  const paperSolid = paperForContrast(paper, mode)
  // The accent stays the colour the user picked in both themes; contrastText keeps text on it readable.
  const primaryMain = accentHex
  return {
    mode,
    primaryMain,
    background: { default: backgroundDefault, paper },
    text: { primary: textPrimary, secondary: textSecondary },
    primary: { main: primaryMain, contrastText: contrastText(primaryMain) },
    info: { main: dark ? '#2B8BDA' : ensureContrast('#2B8BDA', paperSolid) },
    success: { main: dark ? '#0CDF64' : ensureContrast('#0CDF64', paperSolid) },
    warning: { main: dark ? '#F3B416' : ensureContrast('#F3B416', paperSolid) },
    error: { main: dark ? '#C70A0A' : ensureContrast('#C70A0A', paperSolid) },
  }
}

function honourTheme(setting: string, osLight: boolean): ThemeMode {
  if (setting === 'light') {
    return 'light'
  }
  if (setting === 'system') {
    return osLight ? 'light' : 'dark'
  }
  return 'dark'
}

export type { ThemeMode }
export {
  composite,
  contrastRatio,
  honourTheme,
  mortarPalette,
  paperForContrast,
  surfaceCssVars,
  surfaces,
}
