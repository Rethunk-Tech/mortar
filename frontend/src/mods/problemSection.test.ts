import { expect, test } from 'bun:test'
import { useProblemFooter } from './problemSection.ts'

test('a footer section opens per profile, and choosing it again closes it', () => {
  const { toggle } = useProblemFooter.getState()
  toggle('a', 'slow')
  toggle('b', 'dismissed')
  expect(useProblemFooter.getState().byProfile).toEqual({ a: 'slow', b: 'dismissed' })
  toggle('a', 'cleanup')
  toggle('b', 'dismissed')
  expect(useProblemFooter.getState().byProfile).toEqual({ a: 'cleanup', b: undefined })
})
