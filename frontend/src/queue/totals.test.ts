import { expect, test } from 'bun:test'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { megabytes, pendingFor, totals } from './totals.ts'

const item = (state: string, sizeKb = 0, modId = 1): Item => ({
  id: state + modId,
  kind: 'install',
  game: 'stardew',
  profileId: 'p',
  modId,
  fileId: 1,
  currentFileId: 0,
  name: 'n',
  fileName: 'f',
  version: '',
  sizeKb,
  state,
  progress: 0,
  speed: 0,
  error: '',
})

test('totals count each state and split the bar over what was asked for', () => {
  const t = totals([
    item('done'),
    item('downloading', 2048),
    item('failed', 1024),
    item('queued', 1024),
    item('skipped', 999),
    item('cancelled', 999),
  ])
  expect([t.done, t.active, t.failed, t.left, t.sizeKb]).toEqual([1, 1, 1, 3, 4096])
  expect([t.doneShare, t.activeShare, t.failedShare]).toEqual([25, 25, 25])
})

test('an empty queue has an empty bar', () => {
  expect(totals([]).doneShare).toBe(0)
})

test('a failed download can be queued again but a waiting one cannot', () => {
  expect(pendingFor([item('failed')], 'p', 1)).toBe(false)
  expect(pendingFor([item('waiting-click')], 'p', 1)).toBe(true)
  expect(pendingFor([item('queued')], 'q', 1)).toBe(false)
})

test('sizes show one decimal until 10 MB', () => {
  expect(megabytes(512)).toBe('0.5')
  expect(megabytes(20_480)).toBe('20')
})
