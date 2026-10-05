import { expect, test } from 'bun:test'
import { Glob } from 'bun'

// The opening tag of each `<IconButton`: from the name to the `>` that closes it, skipping braces so an arrow
// function's `=>` inside a prop does not end it.
function openingTags(source: string): string[] {
  const tags: string[] = []
  for (const found of source.matchAll(/<IconButton(?=[\s>])/g)) {
    const at = found.index
    let depth = 0
    let end = at
    for (; end < source.length; end++) {
      const ch = source[end]
      if (ch === '{') {
        depth += 1
      } else if (ch === '}') {
        depth -= 1
      } else if (ch === '>' && depth === 0) {
        break
      }
    }
    tags.push(source.slice(at, end))
  }
  return tags
}

test('every IconButton carries an aria-label, since it has no visible text', async () => {
  const bad: string[] = []
  const files = [...new Glob('**/*.tsx').scanSync(new URL('..', import.meta.url).pathname)]
  for (const file of files.filter((f) => !f.endsWith('.test.tsx'))) {
    const source = await Bun.file(new URL(`../${file}`, import.meta.url)).text()
    for (const tag of openingTags(source)) {
      // A spread is the wrapper passing a label through (TipIconButton sets its own).
      if (!(tag.includes('aria-label=') || tag.includes('{...'))) {
        bad.push(`${file}: ${tag.split('\n')[0]}`)
      }
    }
  }
  expect(bad).toEqual([])
})
