// Trims package.json and frontend/package.json in the current folder to the packages the app's own build reads:
// no site workspace, no lint or test tools. render.sh runs it before writing yarn.lock and the Flathub build runs it
// before `yarn install`, so both resolve the same set. Runs under node and bun alike.
import { existsSync, readFileSync, writeFileSync } from 'node:fs'

const tools = new Set([
  '@axe-core/playwright',
  '@biomejs/biome',
  '@happy-dom/global-registrator',
  '@playwright/test',
  '@rethunk/react-render-audit',
  '@testing-library/dom',
  '@testing-library/react',
  '@types/bun',
  '@types/react',
  '@types/react-dom',
  'knip',
  'pagefind',
  'typescript',
])

for (const path of ['package.json', 'frontend/package.json']) {
  const pkg = JSON.parse(readFileSync(path, 'utf8'))
  if (path === 'package.json') {
    pkg.workspaces = ['frontend']
    // yarn reads `resolutions` where bun reads `overrides`. Yarn resolves every entry even when nothing depends on
    // it, so an override of a package the trimmed lock dropped (the site's) would fail offline.
    const lock = existsSync('yarn.lock') ? readFileSync('yarn.lock', 'utf8') : ''
    pkg.resolutions = Object.fromEntries(
      Object.entries(pkg.overrides ?? {}).filter(([name]) => lock.includes(`"${name}@`)),
    )
  }
  pkg.devDependencies = Object.fromEntries(
    Object.entries(pkg.devDependencies ?? {}).filter(([name]) => !tools.has(name)),
  )
  writeFileSync(path, `${JSON.stringify(pkg, null, 2)}\n`)
}
