import { expect, test } from 'bun:test'
import { splitDropped } from './dropped.ts'

test('a dropped .mortar file goes to Import and the rest to install', () => {
  expect(splitDropped(['/a/mod.zip', '/b/Cozy farm.MORTAR', '/c/x.7z', '/d/other.mortar'])).toEqual(
    {
      archives: ['/a/mod.zip', '/c/x.7z'],
      mortar: '/b/Cozy farm.MORTAR',
    },
  )
  expect(splitDropped(['/a/mod.zip'])).toEqual({ archives: ['/a/mod.zip'], mortar: '' })
  expect(splitDropped([])).toEqual({ archives: [], mortar: '' })
})
