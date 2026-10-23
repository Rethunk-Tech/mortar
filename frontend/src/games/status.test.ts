import { expect, test } from 'bun:test'
import type { Status } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import { installedVersion, loaderCaption } from './status.ts'

const installed = (version: string): Status => ({
  installed: true,
  broken: false,
  version,
  gameVersion: '1.6.15',
  latest: version,
  updateAvailable: false,
  shared: false,
  linkedFrom: '',
  perProfile: false,
})

test('game select names the installed SMAPI version, and SMAPI alone when it is missing', () => {
  expect(loaderCaption('SMAPI', installedVersion(installed('4.5.2')))).toBe('SMAPI 4.5.2')
  expect(loaderCaption('SMAPI', installedVersion(null))).toBe('SMAPI')
  expect(
    loaderCaption('SMAPI', installedVersion({ ...installed('4.5.2'), installed: false })),
  ).toBe('SMAPI')
})
