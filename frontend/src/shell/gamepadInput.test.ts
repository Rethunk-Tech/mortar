import { expect, test } from 'bun:test'
import { firedActions, heldActions, type PadAction, spatialNext } from './gamepadInput.ts'

const pad = (pressed: number[], axes: number[] = [0, 0]) => ({
  buttons: Array.from({ length: 17 }, (_, i) => ({
    pressed: pressed.includes(i),
    touched: false,
    value: 0,
  })),
  axes,
})

test('buttons and the left stick map to actions', () => {
  expect([...heldActions([pad([0, 5])])]).toEqual(['confirm', 'nextTab'])
  expect([...heldActions([null, pad([], [-0.9, 0.7])])]).toEqual(['left', 'down'])
  expect([...heldActions([pad([], [0.3, -0.3])])]).toEqual([])
})

test('a press fires once and a held direction repeats after a pause', () => {
  const due = new Map<PadAction, number>()
  const down = new Set<PadAction>(['down', 'confirm'])
  expect(firedActions(down, due, 0)).toEqual(['down', 'confirm'])
  expect(firedActions(down, due, 300)).toEqual([])
  expect(firedActions(down, due, 400)).toEqual(['down'])
  expect(firedActions(down, due, 450)).toEqual([])
  expect(firedActions(down, due, 520)).toEqual(['down'])
  expect(firedActions(new Set(), due, 600)).toEqual([])
  expect(firedActions(down, due, 610)).toEqual(['down', 'confirm'])
})

// A row of three 100px buttons over one wide button, and a control inside the wide one's right end.
const box = (left: number, top: number, width: number, height = 40) => ({
  left,
  top,
  right: left + width,
  bottom: top + height,
})
const rects = [
  box(0, 0, 100),
  box(120, 0, 100),
  box(240, 0, 100),
  box(0, 60, 340),
  box(290, 65, 40, 30),
]

test('directions pick the nearest control in line', () => {
  expect(spatialNext(rects[0] as (typeof rects)[0], rects, 'right', 0)).toBe(1)
  expect(spatialNext(rects[2] as (typeof rects)[0], rects, 'right', 2)).toBe(-1)
  expect(spatialNext(rects[1] as (typeof rects)[0], rects, 'down', 1)).toBe(3)
  expect(spatialNext(rects[3] as (typeof rects)[0], rects, 'up', 3)).toBe(1)
  expect(spatialNext(rects[3] as (typeof rects)[0], rects, 'right', 3)).toBe(4)
  expect(spatialNext(rects[4] as (typeof rects)[0], rects, 'up', 4)).toBe(2)
})
