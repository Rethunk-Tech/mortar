import { expect, test } from 'bun:test'
import { pictureSrc } from './pictureSrc.ts'

test('pictureSrc routes a remote picture through the local cache, escaped', () => {
  expect(pictureSrc('https://cdn.modrinth.com/data/a b/icon.png?x=1')).toBe(
    '/mod-picture/?u=https%3A%2F%2Fcdn.modrinth.com%2Fdata%2Fa%20b%2Ficon.png%3Fx%3D1',
  )
})
