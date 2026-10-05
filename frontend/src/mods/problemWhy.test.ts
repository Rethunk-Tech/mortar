import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const here = import.meta.dir
const read = (rel: string) => readFileSync(join(here, rel), 'utf8')

// GitHub's heading anchors: lower case, spaces to hyphens, punctuation dropped.
const slug = (heading: string) =>
  heading
    .toLowerCase()
    .replace(/[^a-z0-9 -]/g, '')
    .replaceAll(' ', '-')

test('every problem row kind has a why paragraph and a user-guide section', () => {
  const kinds = [...read('lookup.ts').matchAll(/\| \{ kind: '(\w+)'/g)].map((m) => m[1] ?? '')
  kinds.push('drift')
  expect(kinds.length).toBeGreaterThanOrEqual(11)
  const why = read('problemWhy.ts')
  const anchors = new Set(
    [...read('../../../docs/user-guide.md').matchAll(/^#{2,4} (.+)$/gm)].map((m) =>
      slug(m[1] ?? ''),
    ),
  )
  for (const kind of kinds) {
    const block = new RegExp(
      `\\n  ${kind}: \\{\\n    anchor: '([a-z-]+)',\\n    text: \\(\\) =>`,
    ).exec(why)
    expect(block, `${kind} has an explanation`).not.toBeNull()
    expect(anchors.has(block?.[1] ?? ''), `${kind} anchor ${block?.[1]} is in the guide`).toBe(true)
  }
})
