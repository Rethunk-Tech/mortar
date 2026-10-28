import { expect, test } from 'bun:test'
import { isCreditList } from '../../../scripts/creditList.ts'
import credits from './generated/credits.json' with { type: 'json' }

test('credits list has name, licence, and project url on every entry', () => {
  expect(isCreditList(credits)).toBe(true)
  const names = new Set(credits.map((e) => e.name))
  for (const n of [
    'Fedora 44 default wallpaper (f44-01-night)',
    '@fontsource/open-sans',
    'lucide-react',
    '@mui/material',
    'react',
    'zustand',
    '@lingui/core',
    '@wailsio/runtime',
    'github.com/wailsapp/wails/v3',
  ]) {
    expect(names.has(n)).toBe(true)
  }
  expect(credits.find((e) => e.name === '@fontsource/open-sans')?.licence).toBe('OFL-1.1')
  expect(credits.find((e) => e.name === 'lucide-react')?.licence).toBe('ISC')
})
