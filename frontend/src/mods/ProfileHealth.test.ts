import { expect, test } from 'bun:test'
import {
  healthView,
  SIDEBAR_BADGES_ALL,
  SIDEBAR_BADGES_OFF,
  SIDEBAR_BADGES_PROBLEMS,
} from './badgeDisplay.ts'

const labels = {
  missing: (n: number) => (n === 1 ? '1 missing requirement' : `${n} missing requirements`),
  problems: (n: number) => (n === 1 ? '1 problem' : `${n} problems`),
  updates: (n: number) => (n === 1 ? '1 update' : `${n} updates`),
}

test('red when missing or problems are present, with only non-zero pluralised lines', () => {
  expect(healthView({ missing: 2, problems: 3, updates: 5 }, SIDEBAR_BADGES_ALL, labels)).toEqual({
    tone: 'red',
    tooltip: '2 missing requirements\n3 problems\n5 updates',
    value: 5,
  })
  expect(healthView({ missing: 1, problems: 0, updates: 0 }, SIDEBAR_BADGES_ALL, labels)).toEqual({
    tone: 'red',
    tooltip: '1 missing requirement',
    value: 1,
  })
})

test('amber when only updates are present', () => {
  expect(healthView({ missing: 0, problems: 0, updates: 5 }, SIDEBAR_BADGES_ALL, labels)).toEqual({
    tone: 'amber',
    tooltip: '5 updates',
    value: 5,
  })
})

test('nothing when all zero', () => {
  expect(healthView({ missing: 0, problems: 0, updates: 0 }, SIDEBAR_BADGES_ALL, labels)).toEqual({
    tone: 'none',
    tooltip: '',
    value: 0,
  })
  expect(healthView(undefined, SIDEBAR_BADGES_ALL, labels).tone).toBe('none')
})

test('setting hides the badge, and problems-only hides update-only', () => {
  expect(healthView({ missing: 2, problems: 3, updates: 5 }, SIDEBAR_BADGES_OFF, labels).tone).toBe(
    'none',
  )
  expect(
    healthView({ missing: 0, problems: 0, updates: 5 }, SIDEBAR_BADGES_PROBLEMS, labels),
  ).toEqual({ tone: 'none', tooltip: '', value: 0 })
  expect(
    healthView({ missing: 2, problems: 0, updates: 5 }, SIDEBAR_BADGES_PROBLEMS, labels),
  ).toEqual({
    tone: 'red',
    tooltip: '2 missing requirements',
    value: 2,
  })
})
