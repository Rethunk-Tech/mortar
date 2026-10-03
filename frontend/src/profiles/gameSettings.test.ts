import { expect, test } from 'bun:test'
import { formSettingsFromBackend } from './formSettingsFromBackend.ts'

test('formSettingsFromBackend keeps known window modes and numeric fields', () => {
  expect(formSettingsFromBackend({})).toBeNull()
  expect(
    formSettingsFromBackend({
      windowMode: 'borderless',
      displayIndex: 1,
      zoomLevel: 90,
      startMuted: true,
    }),
  ).toEqual({
    windowMode: 'borderless',
    displayIndex: 1,
    zoomLevel: 90,
    startMuted: true,
  })
})
