import { describe, expect, test } from 'bun:test'
import { lastRunCrashToast } from './crashToast.ts'

const copy = { title: 't', body: 'b', action: 'a' }

describe('lastRunCrashToast', () => {
  test('is omitted when the last run did not crash', () => {
    expect(lastRunCrashToast(false, copy, () => undefined)).toBeUndefined()
  })

  test('is a warning with the report action when the last run crashed', () => {
    const report = () => undefined
    expect(lastRunCrashToast(true, copy, report)).toEqual({
      kind: 'warning',
      title: 't',
      body: 'b',
      action: { label: 'a', run: report },
    })
  })
})
