import { expect, test } from 'bun:test'
import { smapiAction } from './smapiAction.ts'

const LATEST = 'latest'
const installed = { installed: true, broken: false, version: '4.5.2', updateAvailable: false }

test('following latest updates or reinstalls without asking', () => {
  expect(smapiAction({ ...installed, updateAvailable: true }, LATEST, LATEST)).toEqual({
    kind: 'update',
    version: '',
    confirm: false,
  })
  expect(smapiAction(installed, LATEST, LATEST)).toEqual({
    kind: 'reinstall',
    version: '',
    confirm: false,
  })
})

test('a pinned other version asks first; the installed one just reinstalls', () => {
  expect(smapiAction(installed, '4.4.0', LATEST)).toEqual({
    kind: 'install',
    version: '4.4.0',
    confirm: true,
  })
  expect(smapiAction(installed, '4.5.2', LATEST)).toEqual({
    kind: 'reinstall',
    version: '4.5.2',
    confirm: false,
  })
})

test('nothing installed installs; a broken launcher reinstalls', () => {
  expect(smapiAction({ ...installed, installed: false }, LATEST, LATEST).kind).toBe('install')
  expect(smapiAction({ ...installed, broken: true }, LATEST, LATEST).kind).toBe('reinstall')
})
