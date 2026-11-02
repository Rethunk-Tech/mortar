import { join } from 'node:path'

const TEST = /\.test\.tsx?$/

export const SRC = join(import.meta.dir, '..', 'src')

/** Every non-test .ts and .tsx file under dir, leaving out the generated locale catalogs. */
export function srcFiles(dir = SRC): string[] {
  return [...new Bun.Glob('**/*.{ts,tsx}').scanSync({ cwd: dir, absolute: true })].filter(
    (p) => !(TEST.test(p) || p.slice(dir.length).split('/').includes('locales')),
  )
}
