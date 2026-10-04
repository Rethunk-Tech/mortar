import { expect, test } from 'bun:test'
import { partitionPreview, selectedFolders, toggled } from './libraryRows.ts'

const rows = [
  { name: 'A', status: 'ok', folder: '/m/A' },
  { name: 'B', status: 'skipped', reason: 'no manifest', folder: '/m/B' },
  { name: 'C', status: 'ok', folder: '/m/.C', disabled: true },
  { name: 'D', status: 'failed', reason: 'bad' },
  { name: 'E', status: 'ok' },
]

test('partitionPreview separates rows that can be picked', () => {
  const { pickable, blocked } = partitionPreview(rows)
  expect(pickable.map((m) => m.name)).toEqual(['A', 'C'])
  expect(blocked.map((m) => m.name)).toEqual(['B', 'D'])
  expect(partitionPreview(null)).toEqual({ pickable: [], blocked: [] })
})

test('selectedFolders drops unchecked rows and toggled flips one', () => {
  const { pickable } = partitionPreview(rows)
  expect(selectedFolders(pickable, new Set())).toEqual(['/m/A', '/m/.C'])
  const off = toggled(new Set(), '/m/A')
  expect(selectedFolders(pickable, off)).toEqual(['/m/.C'])
  expect([...toggled(off, '/m/A')]).toEqual([])
})
