// Rewrites a yarn v1 lockfile that `bun install --yarn` wrote so each pattern an `overrides` entry forces reads
// `name@<override>`. Bun keeps the ranges the dependents declared (an optional wasm binding's exact
// `@emnapi/core@2.0.0-beta.2`) on the entry of the overriding version, which yarn then cannot satisfy offline.
// The Flathub build gets the same overrides as yarn `resolutions` from app-packages.ts. Usage: bun yarn-lock.ts LOCK.
import { readFileSync, writeFileSync } from 'node:fs'
import process from 'node:process'

const versionSuffix = /(?<=.)@[^@]*$/u

export const pinOverrides = (lock: string, overrides: Record<string, string>): string =>
  lock.replace(/^("[^\n]+"):$/gmu, (header, patterns: string) => {
    const names = new Set(
      patterns.split(', ').map((p) => p.slice(1, -1).replace(versionSuffix, '')),
    )
    const [name] = names
    const spec = name === undefined ? undefined : overrides[name]
    return names.size === 1 && spec !== undefined ? `"${name}@${spec}":` : header
  })

if (import.meta.main) {
  const lock = process.argv[2] ?? ''
  const root = JSON.parse(readFileSync('package.json', 'utf8')) as {
    overrides?: Record<string, string>
  }
  writeFileSync(lock, pinOverrides(readFileSync(lock, 'utf8'), root.overrides ?? {}))
}
