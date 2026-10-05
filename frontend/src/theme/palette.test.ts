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
