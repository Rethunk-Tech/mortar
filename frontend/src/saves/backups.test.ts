import { afterEach, beforeEach, expect, mock, test } from 'bun:test'
import * as savessvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { stubBinding } from '../testing/stubBinding.ts'
import { causeLabel, useSaveBackups } from './backups.ts'

let restored: { name: string; folders: string[] | null } = { name: '', folders: null }
const firstBackup = {
  name: '2026-07-01T00-00-00.000.zip',
  at: Date.parse('2026-07-01T00:00:00.000Z'),
  size: 2048,
  profile: 'p1',
  kind: 'update',
  pinned: false,
  saves: [{ folder: 'Farm_1', farm: 'Sunny' }],
}
const listed = [firstBackup]
let listImpl: () => Promise<typeof listed> = async () => listed
let listedFor: string[] = []

beforeEach(() => {
  stubBinding(savessvc, 'ListBackups', (game: string, profile: string) => {
    listedFor = [game, profile]
    return listImpl()
  })
  stubBinding(
    savessvc,
    'RestoreBackup',
    (_game: string, _profile: string, name: string, folders: string[] | null) => {
      restored = { name, folders }
    },
  )
  stubBinding(savessvc, 'SetBackupPinned', () => undefined)
  stubBinding(savessvc, 'OpenBackupsFolder', () => undefined)
})
afterEach(() => {
  mock.restore()
})

test('load fills backups newest as returned', async () => {
  listImpl = async () => listed
  useSaveBackups.setState(useSaveBackups.getInitialState(), true)
  await useSaveBackups.getState().load('stardew', 'p1')
  expect(listedFor).toEqual(['stardew', 'p1'])
  expect(useSaveBackups.getState().items).toEqual(listed)
  expect(useSaveBackups.getState().status).toBe('ready')
})

test('restore asks for the chosen zip and folders then reloads', async () => {
  listImpl = async () => listed
  restored = { name: '', folders: null }
  useSaveBackups.setState(
    {
      ...useSaveBackups.getInitialState(),
      game: 'stardew',
      profile: 'p1',
      items: listed,
      status: 'ready',
    },
    true,
  )
  await useSaveBackups.getState().restore(firstBackup.name, ['Farm_1'])
  expect(restored).toEqual({ name: firstBackup.name, folders: ['Farm_1'] })
  expect(useSaveBackups.getState().items).toEqual(listed)
})

test('a slower first list does not overwrite a later load', async () => {
  let n = 0
  let release: () => void = () => undefined
  const gate = new Promise<void>((resolve) => {
    release = resolve
  })
  listImpl = async () => {
    n += 1
    if (n === 1) {
      await gate
      return [firstBackup]
    }
    return [{ ...firstBackup, name: 'newer.zip' }]
  }
  useSaveBackups.setState(useSaveBackups.getInitialState(), true)
  const first = useSaveBackups.getState().load('stardew', 'p1')
  const second = useSaveBackups.getState().load('stardew', 'p1')
  await second
  release()
  await first
  expect(useSaveBackups.getState().items.map((b) => b.name)).toEqual(['newer.zip'])
})

test('formatBytes uses KB past a kibibyte', () => {
  expect(formatBytes(500)).toBe('500 bytes')
  expect(formatBytes(2048)).toBe('2 kB')
})

test('opening for another game never shows the rows read before', async () => {
  let release: () => void = () => undefined
  listImpl = () =>
    new Promise((resolve) => {
      release = () => resolve([])
    })
  useSaveBackups.setState(
    {
      ...useSaveBackups.getInitialState(),
      game: 'stardew',
      profile: 'p1',
      items: listed,
      status: 'ready',
    },
    true,
  )
  const loading = useSaveBackups.getState().load('lethal-company', 'p9')
  expect(useSaveBackups.getState().items).toEqual([])
  release()
  await loading
  expect(listedFor).toEqual(['lethal-company', 'p9'])
  expect(useSaveBackups.getState().items).toEqual([])
})

test('an update backup names its profile, or says it was deleted', () => {
  expect(causeLabel(firstBackup, [{ id: 'p1', name: 'Main' }])).toBe('Before updating Main')
  expect(causeLabel(firstBackup, [])).toBe('Before updating a deleted profile')
})
