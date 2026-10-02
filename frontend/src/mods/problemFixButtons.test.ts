import { expect, test } from 'bun:test'
import { assetFixButtonStyle } from './problemGroups.ts'

test('cosmetic asset fixes use outlined inherit buttons', () => {
  expect(assetFixButtonStyle(true)).toEqual({ variant: 'outlined', color: 'inherit' })
  expect(assetFixButtonStyle(false)).toEqual({ variant: 'contained', color: 'warning' })
})
