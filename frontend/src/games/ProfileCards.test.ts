import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const { dir } = import.meta

test('profile cards have no per-card Play button', () => {
  const src = readFileSync(join(dir, 'ProfileCards.tsx'), 'utf8')
  expect(src).not.toContain('useLaunch')
  expect(src).not.toContain('playDirect')
  expect(src).not.toContain('PLAY_ICON')
  expect(src).not.toMatch(/t`Play`/)
})

test('Game Select renders profile cards inside the banner row', () => {
  const src = readFileSync(join(dir, 'GameSelect.tsx'), 'utf8')
  expect(src).toContain('[loaderLine, note]')
  expect(src.indexOf('<ProfileCards')).toBeGreaterThan(src.indexOf('[loaderLine, note]'))
  expect(src).not.toMatch(/<\/Row>[\s\S]{0,80}<ProfileCards/)
})
