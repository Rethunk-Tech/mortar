import { expect, test } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const OPENERS =
  /Browser\.(OpenURL|OpenFile)|window\.open\(|location\.(href|assign|replace)\s*[=(]|<(a|Link)\b[^>]*\bhref=/s

function sources(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const path = join(dir, e.name)
    if (e.isDirectory()) {
      return sources(path)
    }
    return /\.tsx?$/.test(e.name) && !/\.test\.tsx?$/.test(e.name) ? [path] : []
  })
}

// A link leaves the window only through openPage, whose backend call opens http and https addresses and refuses every
// other scheme. A direct Browser.OpenURL, window.open or a link the webview follows would let remote data pick the scheme.
test('nothing opens an address except the opener service', () => {
  const offenders: string[] = []
  for (const path of sources(join(import.meta.dir, '..'))) {
    const text = readFileSync(path, 'utf8')
    if (OPENERS.test(text)) {
      offenders.push(path)
    }
  }
  expect(offenders).toEqual([])
})
