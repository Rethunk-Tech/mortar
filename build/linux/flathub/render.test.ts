import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

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
