import { expect, test } from 'bun:test'
import { listArchives } from './downloadsDialog.ts'

test('newest first, filtered by name', () => {
  const a = (name: string, mtime: number) => ({
    path: `/d/${name}`,
    name,
    size: 1,
    mtime,
    knownNexus: false,
    modId: 0,
  })
  const all = [a('Old.zip', 1), a('New.zip', 3), a('Mid-old.zip', 2)]
  expect(listArchives(all, '').map((x) => x.name)).toEqual(['New.zip', 'Mid-old.zip', 'Old.zip'])
  expect(listArchives(all, ' OLD').map((x) => x.name)).toEqual(['Mid-old.zip', 'Old.zip'])
})
