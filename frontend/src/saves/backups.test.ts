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

mock.module('../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts', () => ({
  ListBackups: async () => listed,
  RestoreBackup: async (name: string, folders: string[] | null) => {
    restored = { name, folders }
  },
  OpenBackupsFolder: async () => undefined,
}))

const { getInitialState, useSaveBackups } = await import('./backups.ts')
const { formatBytes } = await import('./backupFormat.ts')

test('load fills backups newest as returned', async () => {
  useSaveBackups.setState(getInitialState(), true)
  await useSaveBackups.getState().load()
  expect(useSaveBackups.getState().items).toEqual(listed)
  expect(useSaveBackups.getState().status).toBe('ready')
})

test('restore asks for the chosen zip and folders then reloads', async () => {
  restored = { name: '', folders: null }
  useSaveBackups.setState({ ...getInitialState(), items: listed, status: 'ready' }, true)
  await useSaveBackups.getState().restore(firstBackup.name, ['Farm_1'])
  expect(restored).toEqual({ name: firstBackup.name, folders: ['Farm_1'] })
  expect(useSaveBackups.getState().items).toEqual(listed)
})

test('formatBytes uses KB past a kibibyte', () => {
  expect(formatBytes(500)).toBe('500 B')
  expect(formatBytes(2048)).toBe('2.0 KB')
})
