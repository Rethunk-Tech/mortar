import { expect, test } from 'bun:test'
import { reportError } from '../toasts/report.ts'
import { fail } from './storeView.ts'

test('fail is the shared reportError helper', () => {
  expect(fail).toBe(reportError)
})
