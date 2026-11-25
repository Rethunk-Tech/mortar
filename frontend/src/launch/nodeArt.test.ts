import { expect, test } from 'bun:test'
import { nodeArt } from './nodeArt.ts'

test('a node whose icon failed draws the hero with the mark over it', () => {
  expect(nodeArt(['/icon', '/hero'], 0, true)).toEqual({ src: '/icon', mark: false })
  expect(nodeArt(['/icon', '/hero'], 1, true)).toEqual({ src: '/hero', mark: true })
  expect(nodeArt(['', '/hero'], 0, true)).toEqual({ src: '/hero', mark: false })
  expect(nodeArt(['/icon', '/hero'], 2, true)).toEqual({ src: undefined, mark: false })
  expect(nodeArt(['/a', '/b'], 1, false).mark).toBe(false)
})
