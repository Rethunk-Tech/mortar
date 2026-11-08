import { describe, expect, test } from 'bun:test'
import { type AccentName, accents } from './accents.ts'
import {
  composite,
  contrastRatio,
  mortarPalette,
  paperForContrast,
  type ThemeMode,
} from './palette.ts'
import { createMortarTheme } from './theme.ts'

function paletteColourStrings(mode: ThemeMode, accent: AccentName): Record<string, string> {
  const pal = mortarPalette(mode, accents[accent])
  const out: Record<string, string> = { primaryMain: pal.primaryMain }
  const groups = [
    pal.background,
    pal.text,
    pal.primary,
    pal.info,
    pal.success,
    pal.warning,
    pal.error,
  ]
  const names = ['background', 'text', 'primary', 'info', 'success', 'warning', 'error']
  for (const [i, group] of groups.entries()) {
    if (!group) {
      continue
    }
    for (const [key, value] of Object.entries(group)) {
      if (typeof value === 'string') {
        out[`${names[i]}.${key}`] = value
      }
    }
  }
  return out
}

const expectedPaletteColours: Record<string, Record<string, string>> = {
  'light:sand': {
    primaryMain: '#D6B17A',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#D6B17A',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:moss': {
    primaryMain: '#93B86A',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#93B86A',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:copper': {
    primaryMain: '#D98C5F',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#D98C5F',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:sky': {
    primaryMain: '#79AEDC',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#79AEDC',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:rose': {
    primaryMain: '#E08A9B',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#E08A9B',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:lavender': {
    primaryMain: '#A893DE',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#A893DE',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:teal': {
    primaryMain: '#5DB8AE',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#5DB8AE',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'light:slate': {
    primaryMain: '#94A7BC',
    'background.default': '#ececf1',
    'background.paper': '#f4f4f7',
    'text.primary': 'rgba(22,22,26,0.92)',
    'text.secondary': 'rgba(40,40,48,0.78)',
    'primary.main': '#94A7BC',
    'primary.contrastText': '#1b1a17',
    'info.main': '#226daa',
    'success.main': '#067c38',
    'warning.main': '#88650d',
    'error.main': '#c70a0a',
  },
  'dark:sand': {
    primaryMain: '#D6B17A',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#D6B17A',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:moss': {
    primaryMain: '#93B86A',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#93B86A',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:copper': {
    primaryMain: '#D98C5F',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#D98C5F',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:sky': {
    primaryMain: '#79AEDC',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#79AEDC',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:rose': {
    primaryMain: '#E08A9B',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#E08A9B',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:lavender': {
    primaryMain: '#A893DE',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#A893DE',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:teal': {
    primaryMain: '#5DB8AE',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#5DB8AE',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
  'dark:slate': {
    primaryMain: '#94A7BC',
    'background.default': 'rgba(25,25,30,0.80)',
    'background.paper': 'rgba(50,50,60,0.80)',
    'text.primary': 'rgba(255,255,255,0.90)',
    'text.secondary': 'rgba(225,225,230,0.95)',
    'primary.main': '#94A7BC',
    'primary.contrastText': '#1b1a17',
    'info.main': '#2B8BDA',
    'success.main': '#0CDF64',
    'warning.main': '#F3B416',
    'error.main': '#e07777',
  },
}

describe('mortarPalette colour strings', () => {
  test('light and dark palettes stay byte-identical for every built-in accent', () => {
    const recorded: Record<string, Record<string, string>> = {}
    for (const mode of ['light', 'dark'] as ThemeMode[]) {
      for (const accent of Object.keys(accents) as AccentName[]) {
        recorded[`${mode}:${accent}`] = paletteColourStrings(mode, accent)
      }
    }
    expect(recorded).toEqual(expectedPaletteColours)
  })
})

describe('mortarPalette contrast', () => {
  const cases = (['light', 'dark'] as ThemeMode[]).flatMap((mode) =>
    (Object.keys(accents) as AccentName[]).map((accent) => [mode, accent] as const),
  )

  test.each(cases)('%s %s text and primary on paper are at least 4.5:1', (mode, accent) => {
    const pal = mortarPalette(mode, accents[accent])
    const paper = pal.background?.paper
    const text = pal.text?.primary
    const primary = pal.primaryMain
    if (typeof paper !== 'string' || typeof text !== 'string' || typeof primary !== 'string') {
      throw new Error('palette colours missing')
    }
    const solid = paperForContrast(paper, mode)
    const textOnPaper = text.includes('rgba') ? composite(text, solid) : text
    expect(contrastRatio(textOnPaper, solid)).toBeGreaterThanOrEqual(4.5)
    expect(primary).toBe(accents[accent])
    const theme = createMortarTheme(accent, { mode })
    expect(theme.palette.primary.main).toBe(primary)
    expect(contrastRatio(theme.palette.primary.contrastText, primary)).toBeGreaterThanOrEqual(4.5)
  })
})
