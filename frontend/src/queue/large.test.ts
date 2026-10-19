import { expect, test } from 'bun:test'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { largeAsk } from './large.ts'

const item = (id: string, state: string) => ({ id, state }) as Item

test('the modal asks about the first held download and counts the rest', () => {
  const items = [item('a', 'queued'), item('b', 'confirm-large'), item('c', 'confirm-large')]
  expect(largeAsk(items)).toEqual({ item: item('b', 'confirm-large'), more: 1 })
})

test('nothing held means no modal, and an answered item stops asking', () => {
  expect(largeAsk([item('a', 'downloading'), item('b', 'needs-confirm')])).toBeNull()
  expect(largeAsk([item('a', 'confirm-large')])?.more).toBe(0)
  expect(largeAsk([item('a', 'queued')])).toBeNull()
})
