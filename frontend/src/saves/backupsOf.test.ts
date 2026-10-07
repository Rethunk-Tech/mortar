import { expect, test } from 'bun:test'
import { backupsOf } from './backupsOf.ts'

const backup = (name: string, folders: string[] | null) => ({
  name,
  at: 0,
  size: 0,
  profile: '',
  kind: 'manual',
  pinned: false,
  saves: folders === null ? null : folders.map((folder) => ({ folder, farm: '' })),
})

test('backupsOf keeps the backups holding the folder, in order', () => {
  const all = [
    backup('a', ['Farm_1']),
    backup('b', ['Other']),
    backup('c', ['Other', 'Farm_1']),
    backup('d', null),
  ]
  expect(backupsOf(all, 'Farm_1').map((b) => b.name)).toEqual(['a', 'c'])
  expect(backupsOf(all, 'Missing')).toEqual([])
})
