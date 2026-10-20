// Builds the website's offline demo of the real frontend into a folder, from the committed recording: bun scripts/site-demo.ts <outDir>
import { spawnSync } from 'node:child_process'
import { resolve } from 'node:path'
import process from 'node:process'

const [, , out] = process.argv
if (!out) {
  console.error('usage: bun scripts/site-demo.ts <outDir>')
  process.exit(2)
}
const build = spawnSync(
  'bunx',
  ['vite', 'build', '--config', 'vite.demo.config.ts', '--outDir', resolve(out), '--emptyOutDir'],
  { cwd: new URL('../frontend', import.meta.url), stdio: 'inherit' },
)
process.exit(build.status ?? 1)
