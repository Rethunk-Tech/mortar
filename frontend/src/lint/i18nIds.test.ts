import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { relative } from 'node:path'
import { transform } from '@lingui/native-tools'
import { SRC, srcFiles } from '../../test/srcFiles.ts'
import { messages } from '../locales/en/messages.ts'

// Runs every file through the macro transform the production build uses, so a
// message the build looks up under an id the catalog lacks shows here, not as a
// raw id on screen.
test('every message id the build emits is in the catalog, and no macro is left untransformed', async () => {
  const problems: string[] = []
  for (const file of srcFiles()) {
    const code = readFileSync(file, 'utf8')
    if (!(code.includes('@lingui/') && code.includes('/macro'))) {
      continue
    }
    const out = (await transform(code, file, { macro: { descriptorFields: 'id-only' } })).code
    const name = relative(SRC, file)
    for (const [, id] of out.matchAll(/\bid: "([^"]+)"/g)) {
      if (id === undefined || !(id in messages)) {
        problems.push(`${name}: id ${id} is not in the catalog`)
      }
    }
    if (/\$__`|(?<![\w$.])(t|msg)`/.test(out)) {
      problems.push(`${name}: a t or msg template is nested in another and was not transformed`)
    }
  }
  expect(problems).toEqual([])
})
