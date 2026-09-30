import { describe, expect, test } from 'bun:test'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'

describe('shouldLeavePageOnEscape', () => {
  test('leaves the page on unhandled Escape with no modal', () => {
    expect(shouldLeavePageOnEscape({ key: 'Escape', defaultPrevented: false }, false)).toBe(true)
  })

  test('stays when Escape was already handled', () => {
    expect(shouldLeavePageOnEscape({ key: 'Escape', defaultPrevented: true }, false)).toBe(false)
  })

  test('stays when a modal is open', () => {
    expect(shouldLeavePageOnEscape({ key: 'Escape', defaultPrevented: false }, true)).toBe(false)
  })

  test('ignores other keys', () => {
    expect(shouldLeavePageOnEscape({ key: 'Enter', defaultPrevented: false }, false)).toBe(false)
  })
})
