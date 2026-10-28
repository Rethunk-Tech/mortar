import { readdirSync } from 'node:fs'
import { join } from 'node:path'

const SOURCE = /\.tsx?$/
const TEST = /\.test\.tsx?$/

export const SRC = join(import.meta.dir, '..', 'src')

/** Every non-test .ts and .tsx file under dir, leaving out the generated locale catalogs. */
export function srcFiles(dir = SRC): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((ent) => {
    const p = join(dir, ent.name)
    if (ent.isDirectory()) {
      return ent.name === 'locales' ? [] : srcFiles(p)
    }
    return SOURCE.test(ent.name) && !TEST.test(ent.name) ? [p] : []
  })
}
