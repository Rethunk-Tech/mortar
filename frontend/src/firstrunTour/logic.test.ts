import { expect, test } from 'bun:test'
import { sameRectOr, tourEligible, tourOnLastStep, tourStepBack, tourStepNext } from './logic.ts'
import { tourClearSeen, tourMarkSeen, tourShouldRun } from './seen.ts'

test('tour steps pass over the ones left out and stop at the ends', () => {
  // Steps 6 and 7 (indexes 5 and 6) are left out, as with no mod rows.
  const shown = [0, 1, 2, 3, 4, 7, 8]
  expect(tourStepNext(0, shown)).toBe(1)
  expect(tourStepNext(4, shown)).toBe(7)
  expect(tourStepBack(7, shown)).toBe(4)
  expect(tourStepNext(8, shown)).toBe(8)
  expect(tourStepBack(0, shown)).toBe(0)
  expect(tourOnLastStep(7, shown)).toBe(false)
  expect(tourOnLastStep(8, shown)).toBe(true)
})

test('tour seen state uses the tour tip id', () => {
  expect(tourShouldRun(undefined)).toBe(true)
  expect(tourShouldRun([])).toBe(true)
  expect(tourShouldRun(['mods'])).toBe(true)
  expect(tourMarkSeen(['mods'])).toEqual(['mods', 'tour'])
  expect(tourShouldRun(['mods', 'tour'])).toBe(false)
  expect(tourClearSeen(['mods', 'tour'])).toEqual(['mods'])
})

test('anchor rect keeps identity until it moves', () => {
  const a = { top: 1, left: 2, width: 3, height: 4 }
  expect(sameRectOr(a, { ...a })).toBe(a)
  expect(sameRectOr(a, { ...a, top: 5 })).toEqual({ ...a, top: 5 })
  expect(sameRectOr(a, null)).toBeNull()
})

test('Skip keeps the tour closed even before the seen flag is saved; Show again reopens it', () => {
  const base = {
    onGameWithProfile: true,
    unseen: true,
    dismissed: false,
    replay: false,
    modalOpen: false,
  }
  expect(tourEligible(base)).toBe(true)
  expect(tourEligible({ ...base, dismissed: true })).toBe(false)
  expect(tourEligible({ ...base, unseen: false })).toBe(false)
  expect(tourEligible({ ...base, unseen: false, dismissed: true, replay: true })).toBe(true)
  expect(tourEligible({ ...base, onGameWithProfile: false, replay: true })).toBe(false)
})

test('the tour waits while a modal dialog is open and returns when it closes', () => {
  const base = {
    onGameWithProfile: true,
    unseen: true,
    dismissed: false,
    replay: false,
    modalOpen: false,
  }
  expect(tourEligible({ ...base, modalOpen: true })).toBe(false)
  expect(tourEligible({ ...base, unseen: false, replay: true, modalOpen: true })).toBe(false)
  expect(tourEligible(base)).toBe(true)
})
