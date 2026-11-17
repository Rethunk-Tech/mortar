import { expect, test } from 'bun:test'
import '../scripts/vendor.ts'

const { shortAPIPath, shortAppLink, shortFromPath } = await import('../public/share/app.js')

test('parses hosted short share paths', () => {
  expect(shortFromPath('/stardew/s/AbCdEfGh12')).toEqual({
    game: 'stardew',
    code: 'AbCdEfGh12',
    version: 0,
  })
  expect(shortFromPath('/stardew/s/AbCdEfGh12/')).toEqual({
    game: 'stardew',
    code: 'AbCdEfGh12',
    version: 0,
  })
  expect(shortFromPath('/stardew/s/AbCdEfGh12/3')).toEqual({
    game: 'stardew',
    code: 'AbCdEfGh12',
    version: 3,
  })
  expect(shortFromPath('/stardew/p/')).toBeNull()
  expect(shortFromPath('/stardew/s/AbCdEfGh1')).toBeNull()
  expect(shortFromPath('/stardew/s/AbCd-EfGh1')).toBeNull()
})

test('builds the share API path and Open in Mortar handoff', () => {
  const found = { game: 'stardew', code: 'AbCdEfGh12', version: 0 }
  expect(shortAPIPath(found)).toBe('/api/shares/AbCdEfGh12')
  expect(shortAppLink(found)).toBe('mortar://stardew/s/AbCdEfGh12')
  const versioned = { game: 'stardew', code: 'AbCdEfGh12', version: 2 }
  expect(shortAPIPath(versioned)).toBe('/api/shares/AbCdEfGh12/2')
  expect(shortAppLink(versioned)).toBe('mortar://stardew/s/AbCdEfGh12/2')
})
