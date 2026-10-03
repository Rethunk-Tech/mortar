import { expect, test } from 'bun:test'
import type { Row } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loadorder/models.ts'
import { formatLoadOrderCopy, loadOrderEmptyKind } from './loadOrderText.ts'

const row = (over: Partial<Row> = {}): Row => ({
  position: 1,
  uniqueId: 'A.Mod',
  name: 'Alpha',
  required: [],
  optional: [],
  dependents: [],
  missingRequired: [],
  cycle: false,
  ...over,
})

test('a failed load is an error empty-state, not No enabled mods', () => {
  expect(loadOrderEmptyKind(true, 0)).toBe('error')
  expect(loadOrderEmptyKind(false, 0)).toBe('empty')
  expect(loadOrderEmptyKind(false, 1)).toBe('list')
})

test('copy load order is numbered plain text', () => {
  expect(
    formatLoadOrderCopy([
      row({ position: 1, name: 'Alpha' }),
      row({ position: 2, uniqueId: 'B.Mod', name: '' }),
    ]),
  ).toBe('1. Alpha\n2. B.Mod')
})
