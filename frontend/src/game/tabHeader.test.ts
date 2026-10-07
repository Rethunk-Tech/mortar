import { expect, test } from 'bun:test'
import { notesFirstLine, PAGE_ACTION_PX, pageActionsSx } from './tabHeader.ts'

test('notesFirstLine skips blank leading lines and trims', () => {
  expect(notesFirstLine('\n  \n  Co-op farm \nsecond')).toBe('Co-op farm')
  expect(notesFirstLine('')).toBe('')
})

test('page actions give buttons and icon buttons one height and icon buttons a square', () => {
  const button = pageActionsSx['& .MuiButton-root, & .MuiIconButton-root']
  expect(button.height).toBe(PAGE_ACTION_PX)
  expect(button.minHeight).toBe(PAGE_ACTION_PX)
  expect(pageActionsSx['& .MuiIconButton-root'].width).toBe(PAGE_ACTION_PX)
})
