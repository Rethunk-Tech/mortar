import { expect, mock, test } from 'bun:test'

let restored: { name: string; folders: string[] | null } = { name: '', folders: null }
const firstBackup = {
  name: '2026-07-01T00-00-00.000.zip',
  at: Date.parse('2026-07-01T00:00:00.000Z'),
  size: 2048,
  profile: 'p1',
  kind: 'update',
  saves: [{ folder: 'Farm_1', farm: 'Sunny' }],
}
const listed = [firstBackup]
let listImpl: () => Promise<typeof listed> = async () => listed

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts', () => ({
  ListBackups: () => listImpl(),
  RestoreBackup: async (name: string, folders: string[] | null) => {
    restored = { name, folders }
  },
  OpenBackupsFolder: async () => undefined,
}))

const { getInitialState, useSaveBackups } = await import('./backups.ts')
const { formatBytes } = await import('./backupFormat.ts')

test('load fills backups newest as returned', async () => {
  listImpl = async () => listed
  useSaveBackups.setState(getInitialState(), true)
  await useSaveBackups.getState().load()
  expect(useSaveBackups.getState().items).toEqual(listed)
  expect(useSaveBackups.getState().status).toBe('ready')
})

test('restore asks for the chosen zip and folders then reloads', async () => {
  listImpl = async () => listed
  restored = { name: '', folders: null }
  useSaveBackups.setState({ ...getInitialState(), items: listed, status: 'ready' }, true)
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
  useSaveBackups.setState(getInitialState(), true)
  const first = useSaveBackups.getState().load()
  const second = useSaveBackups.getState().load()
  await second
  release()
  await first
  expect(useSaveBackups.getState().items.map((b) => b.name)).toEqual(['newer.zip'])
})

test('formatBytes uses KB past a kibibyte', () => {
  expect(formatBytes(500)).toBe('500 B')
  expect(formatBytes(2048)).toBe('2.0 KB')
})
