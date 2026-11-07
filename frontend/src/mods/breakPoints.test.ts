import { expect, test } from 'bun:test'
import { breakPoints } from './breakPoints.ts'

test('an id breaks after each separator, never inside a word', () => {
  expect(breakPoints('herbivoor.custommenubackground')).toEqual([
    { at: 0, part: 'herbivoor.' },
    { at: 10, part: 'custommenubackground' },
  ])
})
