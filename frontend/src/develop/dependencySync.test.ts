import { expect, test } from 'bun:test'
import type { DepSync } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { outOfStep } from './dependencySync.ts'

test('outOfStep counts missing, undeclared and differing dependencies, and nothing without a sync', () => {
  expect(outOfStep(null)).toBe(0)
  expect(outOfStep({ missing: [] } as unknown as DepSync)).toBe(0)
  expect(
    outOfStep({ missing: ['a'], undeclared: ['b', 'c'], differs: ['d'] } as unknown as DepSync),
  ).toBe(4)
})
