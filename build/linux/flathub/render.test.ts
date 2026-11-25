import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { pinOverrides } from './yarn-lock.ts'

// render.sh takes everything above the sideload manifest's `modules:` as the Flathub header and appends this
// directory's template from `sdk-extensions:` on, so the two manifests share one list of finish-args.
const topLevelKeys = (path: string): string[] =>
  [...readFileSync(path, 'utf8').matchAll(/^([a-z-]+):/gmu)].map((m) => m[1] ?? '')

test('the Flathub template carries no header of its own', () => {
  expect(topLevelKeys(join(import.meta.dir, 'tech.rethunk.Mortar.yml'))).toEqual([
    'sdk-extensions',
    'modules',
  ])
})

test('the sideload manifest holds the header and finish-args above its modules', () => {
  const keys = topLevelKeys(join(import.meta.dir, '..', 'flatpak', 'tech.rethunk.Mortar.yml'))
  expect(keys).toContain('finish-args')
  expect(keys).not.toContain('sdk-extensions')
  expect(keys.at(-1)).toBe('modules')
})

test('scripts the Flathub build runs with node use no Bun APIs', () => {
  const template = readFileSync(join(import.meta.dir, 'tech.rethunk.Mortar.yml'), 'utf8')
  const scripts = [...template.matchAll(/^\s*- node (\S+\.ts)$/gmu)].map((m) => m[1] ?? '')
  expect(scripts).not.toEqual([])
  for (const script of scripts) {
    expect(readFileSync(join(import.meta.dir, '..', '..', '..', script), 'utf8')).not.toMatch(
      /\bBun\.|bun:/u,
    )
  }
})

// yarn refuses a lock entry whose version does not satisfy one of its patterns, and `--offline` finds no other tarball.
const unsatisfied = (lock: string): string[] =>
  [...lock.matchAll(/^("[^\n]+"):\n {2}version "([^"]+)"/gmu)].flatMap((m) =>
    (m[1] ?? '').split(', ').flatMap((quoted) => {
      const pattern = quoted.slice(1, -1)
      const at = pattern.lastIndexOf('@')
      const range = pattern.slice(at + 1)
      return Bun.semver.satisfies(m[2] ?? '', range) ? [] : [`${pattern} -> ${m[2]}`]
    }),
  )

// The shape bun writes for an overridden package whose dependents ask for other versions.
const bunLock = `"@emnapi/core@2.0.0-beta.2", "@emnapi/core@^1.7.1 || 2.0.0-beta.2":
  version "1.11.3"
  resolved "https://registry.npmjs.org/@emnapi/core/-/core-1.11.3.tgz"

"@emnapi/wasi-threads@1.2.3":
  version "1.2.3"
`

test('an overridden package keeps no pattern its version fails', () => {
  expect(unsatisfied(bunLock)).not.toEqual([])
  expect(unsatisfied(pinOverrides(bunLock, { '@emnapi/core': '1.11.3' }))).toEqual([])
})

test('the rendered yarn.lock satisfies every pattern', () => {
  const rendered = process.env.MORTAR_FLATHUB_YARN_LOCK
  expect(rendered ? unsatisfied(readFileSync(rendered, 'utf8')) : []).toEqual([])
})
