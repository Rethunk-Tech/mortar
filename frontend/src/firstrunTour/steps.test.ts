import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { Glob } from 'bun'
import { TOUR_ANCHORS, TOUR_STEPS } from './steps.ts'

const root = new URL('..', import.meta.url).pathname
const source = [...new Glob('**/*.tsx').scanSync(root)]
  .filter((f) => !f.startsWith('firstrunTour/'))
  .map((f) => readFileSync(root + f, 'utf8'))
  .join('\n')

test('every tour step names a target, and each data attribute it selects is rendered by a component', () => {
  for (const step of TOUR_STEPS) {
    expect(TOUR_ANCHORS[step].length).toBeGreaterThan(0)
    for (const selector of TOUR_ANCHORS[step]) {
      for (const [, name, value] of selector.matchAll(/\[(data-[\w-]+)(?:="([^"]*)")?\]/g)) {
        const rendered = value === undefined ? `${name}=` : `${name}="${value}"`
        expect(source.includes(rendered) || source.includes(`${name}={`)).toBe(true)
      }
    }
  }
})
