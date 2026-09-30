import { describe, expect, test } from 'bun:test'
import { nxmOwnerName } from './nxmOwnerName.ts'

describe('nxmOwnerName', () => {
  test('names Mortar when Mortar handles the links', () => {
    expect(nxmOwnerName(true, 'Vortex')).toBe('Mortar')
  })

  test('names the other app when it owns the links', () => {
    expect(nxmOwnerName(false, 'Vortex')).toBe('Vortex')
  })

  test('is empty when nothing owns the links', () => {
    expect(nxmOwnerName(false, '')).toBe('')
  })
})
