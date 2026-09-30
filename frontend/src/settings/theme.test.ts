import { describe, expect, test } from 'bun:test'
import { buildTheme, isAccent } from './theme.ts'

describe('buildTheme', () => {
  test('translucent keeps the theme rgba surfaces', () => {
    const { background } = buildTheme('sand', true).palette
    expect(background.default).toBe('rgba(25,25,30,0.80)')
    expect(background.paper).toBe('rgba(50,50,60,0.80)')
  })

  test('opaque uses the same RGB at alpha 1', () => {
    const { background } = buildTheme('moss', false).palette
    expect(background.default).toBe('rgb(25,25,30)')
    expect(background.paper).toBe('rgb(50,50,60)')
  })

  test('accent survives the wrapper', () => {
    expect(buildTheme('sky', false).palette.primary.main).toBe('#79AEDC')
  })

  test('isAccent rejects unknown names', () => {
    expect(isAccent('copper')).toBe(true)
    expect(isAccent('neon')).toBe(false)
  })
})
