import { expect, test } from 'bun:test'
import { cloneFolder, isRepoAddress } from './repo.ts'

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

test('a clone lands in the repository name below the projects folder', () => {
  expect(cloneFolder('/home/me/Projects/', 'https://github.com/o/shop.git')).toBe(
    '/home/me/Projects/shop',
  )
  expect(cloneFolder('C:\\Projects', 'git@github.com:o/shop')).toBe('C:\\Projects\\shop')
})
