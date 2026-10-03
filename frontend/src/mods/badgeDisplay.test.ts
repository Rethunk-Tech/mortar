import { expect, test } from 'bun:test'
import { runBackgroundBadgeChecks, showProblemBadge, showUpdateBadge } from './badgeDisplay.ts'

test('problemsAndUpdates shows both badges', () => {
  expect(showUpdateBadge('problemsAndUpdates')).toBe(true)
  expect(showProblemBadge('problemsAndUpdates')).toBe(true)
})

test('problems hides update badges', () => {
  expect(showUpdateBadge('problems')).toBe(false)
  expect(showProblemBadge('problems')).toBe(true)
})

test('off hides badges and skips background checks', () => {
  expect(showUpdateBadge('off')).toBe(false)
  expect(showProblemBadge('off')).toBe(false)
  expect(runBackgroundBadgeChecks('off', true)).toBe(false)
  expect(runBackgroundBadgeChecks('problemsAndUpdates', true)).toBe(true)
  expect(runBackgroundBadgeChecks('problemsAndUpdates', false)).toBe(false)
})
