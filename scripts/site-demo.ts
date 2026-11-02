// Builds the website's offline demo of the real frontend into a folder, from the committed recording: bun scripts/site-demo.ts <outDir>
import { resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const [, , out] = process.argv
if (!out) {
  console.error('usage: bun scripts/site-demo.ts <outDir>')
  process.exit(2)
}
const build = Bun.spawnSync(
  [
    'bunx',
    'vite',
    'build',
    '--config',
    'vite.demo.config.ts',
    '--outDir',
    resolve(out),
    '--emptyOutDir',
  ],
  {
    cwd: fileURLToPath(new URL('../frontend', import.meta.url)),
    stdio: ['inherit', 'inherit', 'inherit'],
  },
)
process.exit(build.exitCode ?? 1)
