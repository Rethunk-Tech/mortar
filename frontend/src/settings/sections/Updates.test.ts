import { describe, expect, test } from 'bun:test'
import { lastOpenedGame } from './lastOpenedGame.ts'

describe('lastOpenedGame', () => {
  test('returns the last game when Mortar knows it', () => {
    expect(lastOpenedGame('stardew')).toBe('stardew')
  })

  test('returns null when no last game is set', () => {
    expect(lastOpenedGame('')).toBeNull()
  })

  test('returns null for a last game this build cannot open', () => {
    expect(lastOpenedGame('lethal')).toBeNull()
  })
})
