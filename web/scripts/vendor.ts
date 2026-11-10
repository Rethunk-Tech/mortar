// Copies the share page's Brotli decoder out of its package into public/, which Astro serves verbatim, so the site
// carries no decoder binary in git. The dev server and the build both run this first.
import { copyFileSync, mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

// The ./web export's files sit in the package's pkg/, the licences at the package root.
const FILES = [
  'pkg/brotli_dec_wasm.js',
  'pkg/brotli_dec_wasm_bg.wasm',
  'LICENSE-MIT.txt',
  'LICENSE-Apache.txt',
]
const pkg = dirname(dirname(fileURLToPath(import.meta.resolve('brotli-dec-wasm/web'))))
const out = fileURLToPath(new URL('../public/vendor/brotli-dec-wasm', import.meta.url))

mkdirSync(out, { recursive: true })
for (const name of FILES) {
  copyFileSync(join(pkg, name), join(out, name.slice(name.lastIndexOf('/') + 1)))
}
