import { expect, test } from 'bun:test'
import {
  sameRectOr,
  TOUR_STEP_COUNT,
  tourEligible,
  tourOnLastStep,
  tourStepBack,
  tourStepNext,
} from './logic.ts'
import { tourClearSeen, tourMarkSeen, tourShouldRun } from './seen.ts'

test('tour step sequencing clamps at ends', () => {
  expect(tourStepNext(0)).toBe(1)
  expect(tourStepNext(TOUR_STEP_COUNT - 1)).toBe(TOUR_STEP_COUNT - 1)
  expect(tourStepBack(TOUR_STEP_COUNT - 1)).toBe(TOUR_STEP_COUNT - 2)
  expect(tourStepBack(0)).toBe(0)
  expect(tourOnLastStep(TOUR_STEP_COUNT - 2)).toBe(false)
  expect(tourOnLastStep(TOUR_STEP_COUNT - 1)).toBe(true)
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
  const base = { onGameWithProfile: true, unseen: true, dismissed: false, replay: false }
  expect(tourEligible(base)).toBe(true)
  expect(tourEligible({ ...base, dismissed: true })).toBe(false)
  expect(tourEligible({ ...base, unseen: false })).toBe(false)
  expect(tourEligible({ ...base, unseen: false, dismissed: true, replay: true })).toBe(true)
  expect(tourEligible({ ...base, onGameWithProfile: false, replay: true })).toBe(false)
})
