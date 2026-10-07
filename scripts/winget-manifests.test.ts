import { expect, test } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'

const dir = new URL('../build/windows/winget/', import.meta.url)

// winget-pkgs rejects a manifest that is not valid YAML, and bump.sh copies these files as they are.
test('every winget manifest parses as YAML and names the package', () => {
  const files = readdirSync(dir).filter((f) => f.endsWith('.yaml'))
  expect(files.length).toBe(3)
  for (const f of files) {
    const doc = Bun.YAML.parse(readFileSync(new URL(f, dir), 'utf8')) as Record<string, unknown>
    expect(doc.PackageIdentifier).toBe('RethunkTech.Mortar')
    expect(typeof doc.PackageVersion).toBe('string')
  }
})
