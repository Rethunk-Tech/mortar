import { readFileSync } from 'node:fs'
import { expect, test } from 'bun:test'
import { steamArt } from './art.ts'
import { releases } from './changelog.ts'
import { rivalFacts } from './faq.ts'
import { REPO, currentVersion, releaseBase, repo } from './site.ts'
import { topMods } from './topmods.ts'

test('currentVersion is build/config.yml, which chrome and downloads share', () => {
  const fromFile =
    /^\s+version: "([^"]+)"/m.exec(readFileSync(repo('build/config.yml'), 'utf8'))?.[1] ?? ''
  expect(currentVersion()).toBe(fromFile)
  expect(currentVersion()).toMatch(/^\d+\.\d+\.\d+$/)
  const download = readFileSync(new URL('../components/Download.astro', import.meta.url), 'utf8')
  const page = readFileSync(new URL('../pages/download/index.astro', import.meta.url), 'utf8')
  const footer = readFileSync(new URL('../components/Footer.astro', import.meta.url), 'utf8')
  expect(download).toContain('currentVersion()')
  expect(download).toContain('releaseBase(')
  expect(page).toContain('currentVersion()')
  expect(page).toContain('releaseBase(')
  expect(footer).toContain('currentVersion()')
})

test('releaseBase is the GitHub release asset directory', () => {
  expect(releaseBase('1.2.3')).toBe(`${REPO}/releases/download/v1.2.3/`)
})

test('build-path readers return the same object on a second call', () => {
  expect(releases()).toBe(releases())
  expect(rivalFacts()).toBe(rivalFacts())
  expect(steamArt()).toBe(steamArt())
  expect(topMods()).toBe(topMods())
})
