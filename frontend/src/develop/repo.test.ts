import { expect, test } from 'bun:test'
import { isRepoAddress } from './repo.ts'

test('a repository is owner/name or a github.com address', () => {
  for (const ok of [
    'o/shop',
    ' https://github.com/o/shop.git ',
    'git@github.com:o/shop',
    'o/a.b',
  ]) {
    expect(isRepoAddress(ok)).toBe(true)
  }
  for (const bad of ['', 'shop', 'a/b/c', 'https://gitlab.com/o/shop', '-o/shop']) {
    expect(isRepoAddress(bad)).toBe(false)
  }
})
