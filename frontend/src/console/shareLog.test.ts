import { expect, test } from 'bun:test'
import { shareLogText } from './shareLog.ts'

const read = {
  live: () => Promise.resolve('live'),
  run: (id: string) => Promise.resolve(`run ${id}`),
  runs: () => Promise.resolve([{ id: 'new' }, { id: 'old' }]),
}

test('a paste loader shares the viewed run, else its newest run, never the live SMAPI log', async () => {
  expect(await shareLogText('old', true, read)).toBe('run old')
  expect(await shareLogText('', true, read)).toBe('run new')
  expect(await shareLogText('', true, { ...read, runs: () => Promise.resolve(null) })).toBe('')
  expect(await shareLogText('', false, read)).toBe('live')
})
