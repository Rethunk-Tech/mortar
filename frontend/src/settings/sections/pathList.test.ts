import { expect, test } from 'bun:test'
import { distinctTails } from './pathList.ts'

test('each folder shows the ending that tells it apart', () => {
  const base = '/usr/local/src/com.github/Rethunk-Tech/mortar-mods'
  expect(
    distinctTails([`${base}/Posters/bin`, `${base}/Boombox/bin`, '/srv/x/Television/bin']),
  ).toEqual(['…/Posters/bin', '…/Boombox/bin', '…/Television/bin'])
  expect(distinctTails([`${base}/a/bin/Release`, `${base}/b/bin/Release`])).toEqual([
    '…/a/bin/Release',
    '…/b/bin/Release',
  ])
  expect(distinctTails(['C:\\Mods\\Posters', 'C:\\Mods\\Boombox'])).toEqual([
    '…\\Mods\\Posters',
    '…\\Mods\\Boombox',
  ])
  expect(distinctTails(['/one'])).toEqual(['/one'])
})
